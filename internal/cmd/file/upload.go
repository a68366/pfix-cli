package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

// maxUploadSize is the API's hard upload limit, measured live: exactly 100 MiB
// is accepted, one byte more is rejected (code 7002) — but only after the whole
// body has been sent, so pfix checks it before transferring anything.
const maxUploadSize = 100 << 20

// attachTargets are the object types a file can be attached to, in flag order.
var attachTargets = []string{"task", "contact", "project"}

type uploadOptions struct {
	path    string // local file; empty when fromURL is set
	fromURL string
	name    string
	nameSet bool
	attach  map[string]string // flag name → raw id, only for flags the user set
	json    bool
	quiet   bool
	jq      string
	client  func() (*planfix.Client, error)
	out     io.Writer
}

func newUploadCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &uploadOptions{}
	ids := map[string]*string{}
	cmd := &cobra.Command{
		Use:   "upload [<path>] [--from-url <url>]",
		Short: "Upload a file, optionally attaching it to a task, contact, or project",
		Long: "Upload a local file, or have Planfix fetch one from a URL, and print the new\n" +
			"file id.\n\n" +
			"Pass exactly one of <path> or --from-url. The uploaded name defaults to the\n" +
			"local file's base name (with --from-url, the API derives it from the URL);\n" +
			"--name overrides it and may not contain a path separator. The API limits a\n" +
			"file to 100 MiB; a larger local file is refused before anything is sent.\n\n" +
			"--from-url stores whatever the URL returns and reports success regardless —\n" +
			"a URL that answers with a 404 or login page is saved as that error page.\n" +
			"Check the result with 'pfix file view <id>'.\n\n" +
			"--task, --contact, or --project (at most one) attaches the uploaded file to\n" +
			"that object in a second request. The upload is not rolled back if attaching\n" +
			"fails; the error names the uploaded file id so it can be attached later.\n" +
			"--json/--jq emit the upload response only, not the attach response.\n\n" +
			"An upload is never retried. Re-uploading the same name and content within a\n" +
			"few minutes returns the same file id.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				o.path = args[0]
			}
			o.nameSet = cmd.Flags().Changed("name")
			o.attach = map[string]string{}
			for _, t := range attachTargets {
				if cmd.Flags().Changed(t) {
					o.attach[t] = *ids[t]
				}
			}
			o.json, o.quiet, o.jq = g.JSON, g.Quiet, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runUpload(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&o.fromURL, "from-url", "", "Have Planfix fetch the file from this URL instead of uploading a local file")
	cmd.Flags().StringVar(&o.name, "name", "", "File name to store (default: the local file's base name)")
	for _, t := range attachTargets {
		ids[t] = new(string)
		cmd.Flags().StringVar(ids[t], t, "", "Attach the uploaded file to this "+t+" id")
	}
	return cmd
}

func runUpload(ctx context.Context, o *uploadOptions) error {
	if (o.path == "") == (o.fromURL == "") {
		return fmt.Errorf("pass exactly one of <path> or --from-url")
	}
	attachType, attachID, err := resolveAttach(o.attach)
	if err != nil {
		return err
	}
	if o.nameSet {
		if err := validateUploadName(o.name, "--name"); err != nil {
			return err
		}
	}

	var raw []byte
	var label string
	if o.path != "" {
		raw, label, err = uploadLocal(ctx, o)
	} else {
		raw, label, err = uploadFromURL(ctx, o)
	}
	if err != nil {
		return err
	}

	var env struct {
		ID int `json:"id"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	if env.ID <= 0 {
		return fmt.Errorf("upload response carried no file id: %s", raw)
	}

	if attachType != "" {
		if err := attach(ctx, o, env.ID, attachType, attachID); err != nil {
			return fmt.Errorf("uploaded %s as file %d, but attaching it to %s %d failed: %w",
				label, env.ID, attachType, attachID, err)
		}
	}

	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", env.ID)
		return nil
	}
	msg := fmt.Sprintf("Uploaded %s as file %d", label, env.ID)
	if attachType != "" {
		msg += fmt.Sprintf(", attached to %s %d", attachType, attachID)
	}
	fmt.Fprintln(o.out, msg)
	return nil
}

// resolveAttach validates the attach flags: at most one, with a positive id.
func resolveAttach(set map[string]string) (string, int, error) {
	if len(set) > 1 {
		return "", 0, fmt.Errorf("pass at most one of --task, --contact, --project")
	}
	for t, raw := range set {
		id, err := cmdutil.ValidateID(raw)
		if err != nil {
			return "", 0, fmt.Errorf("--%s: %w", t, err)
		}
		return t, id, nil
	}
	return "", 0, nil
}

// validateUploadName rejects names the API would store unsafely. The server
// keeps the multipart filename verbatim, path separators included, so a name
// like "../x" would be saved as-is and resurface on download.
func validateUploadName(name, what string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return fmt.Errorf("%s must not be empty", what)
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf("%s must not contain a path separator, got %q", what, name)
	case name == "." || name == "..":
		return fmt.Errorf("%s must be a file name, got %q", what, name)
	case strings.ContainsFunc(name, unicode.IsControl):
		// A raw CR/LF would break the multipart part header.
		return fmt.Errorf("%s must not contain control characters, got %q", what, name)
	}
	return nil
}

func uploadLocal(ctx context.Context, o *uploadOptions) ([]byte, string, error) {
	fi, err := os.Stat(o.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", fmt.Errorf("%s: no such file", o.path)
		}
		return nil, "", err
	}
	if fi.IsDir() {
		return nil, "", fmt.Errorf("%s is a directory; upload a single file", o.path)
	}
	if !fi.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%s is not a regular file", o.path)
	}
	if fi.Size() > maxUploadSize {
		return nil, "", fmt.Errorf("%s is %d bytes; the API limits uploads to 100 MiB (%d bytes)",
			o.path, fi.Size(), maxUploadSize)
	}
	name := o.name
	if !o.nameSet {
		name = filepath.Base(o.path)
		if err := validateUploadName(name, "file name"); err != nil {
			return nil, "", fmt.Errorf("%w (use --name)", err)
		}
	}

	f, err := os.Open(o.path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	client, err := o.client()
	if err != nil {
		return nil, "", err
	}
	raw, err := client.Upload(ctx, "file/", name, f, fi.Size())
	if err != nil {
		return nil, "", describeUploadError(err)
	}
	return raw, name, nil
}

func uploadFromURL(ctx context.Context, o *uploadOptions) ([]byte, string, error) {
	body := map[string]string{"url": o.fromURL}
	label := o.fromURL
	if o.nameSet {
		body["name"] = o.name
		label = o.name
	}
	client, err := o.client()
	if err != nil {
		return nil, "", err
	}
	// Not retried: the fetch-and-store is not idempotent, and an unresolvable
	// host answers 500.
	raw, err := client.JSONOnce(ctx, "POST", "file/from-url/", body)
	if err != nil {
		return nil, "", describeUploadError(err)
	}
	return raw, label, nil
}

// attach links an uploaded file to an object. The target id is a query
// parameter (also named id, shadowing the path's file id); there is no body.
// Re-attaching is a no-op on the server, so the retrying JSON path is safe.
func attach(ctx context.Context, o *uploadOptions, fileID int, target string, targetID int) error {
	client, err := o.client()
	if err != nil {
		return err
	}
	path := "file/" + strconv.Itoa(fileID) + "/attach/" + target + "?id=" + strconv.Itoa(targetID)
	if _, err := client.JSON(ctx, "POST", path, nil); err != nil {
		return describeUploadError(err)
	}
	return nil
}

// describeUploadError adds a hint for the API codes the upload and attach
// endpoints are known to return; anything else is returned as is (auth
// failures get their hint at the root).
func describeUploadError(err error) error {
	var apiErr *planfix.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	switch apiErr.Code {
	case 7002:
		return fmt.Errorf("%w — the file exceeds the API's 100 MiB upload limit", err)
	case 7000:
		return fmt.Errorf("%w — no file with that id", err)
	case 7001:
		return fmt.Errorf("%w — the request carried no URL to fetch", err)
	case 1000:
		return fmt.Errorf("%w — no task with that id", err)
	case 43:
		return fmt.Errorf("%w — the Idempotency-Key was already used", err)
	}
	return err
}
