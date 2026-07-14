package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type downloadOptions struct {
	id     int
	output string // -o: "" auto-name, "-" stdout, else path/dir
	force  bool
	quiet  bool
	json   bool
	client func() (*planfix.Client, error)
	out    io.Writer
	errOut io.Writer
}

func newDownloadCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &downloadOptions{}
	cmd := &cobra.Command{
		Use:   "download <id>",
		Short: "Download a file's bytes",
		Long: "Download a file's bytes.\n\n" +
			"By default writes ./<file-name> (a metadata lookup resolves the name). -o <path>\n" +
			"writes to that path; -o <dir>/ writes <dir>/<file-name>; -o - streams to stdout.\n" +
			"Refuses to overwrite an existing file unless --force.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := cmdutil.ValidateID(args[0])
			if err != nil {
				return err
			}
			o.id = id
			o.quiet, o.json = g.Quiet, g.JSON
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			o.errOut = cmd.ErrOrStderr()
			return runDownload(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "Output path, a directory, or - for stdout")
	cmd.Flags().BoolVar(&o.force, "force", false, "Overwrite an existing file")
	return cmd
}

func runDownload(ctx context.Context, o *downloadOptions) error {
	// --jq implies --json (GlobalOpts.PreRun), so o.json covers both.
	if o.json {
		return fmt.Errorf("file download writes raw bytes; --json/--jq are not supported (use -o -)")
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	idPath := "file/" + strconv.Itoa(o.id)

	dest, toStdout, err := resolveDest(ctx, o, client, idPath)
	if err != nil {
		return err
	}

	var w io.Writer
	var f *os.File
	if toStdout {
		w = o.out
	} else {
		flag := os.O_WRONLY | os.O_CREATE | os.O_EXCL
		if o.force {
			flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		}
		f, err = os.OpenFile(dest, flag, 0o644)
		if err != nil {
			if os.IsExist(err) {
				return fmt.Errorf("%s exists (use --force to overwrite)", dest)
			}
			return err
		}
		w = f
	}

	resp, err := client.Stream(ctx, idPath+"/download")
	if err != nil {
		cleanupPartial(f)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		cleanupPartial(f)
		return cmdutil.DescribeAPIError(planfix.ParseError(resp.StatusCode, data))
	}

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		cleanupPartial(f)
		return err
	}
	if resp.ContentLength >= 0 && n != resp.ContentLength {
		cleanupPartial(f)
		return fmt.Errorf("download truncated: got %d bytes, expected %d", n, resp.ContentLength)
	}
	if f != nil {
		if err := f.Close(); err != nil {
			return err
		}
	}
	if !o.quiet && !toStdout {
		fmt.Fprintf(o.errOut, "Saved %s (%d bytes)\n", dest, n)
	}
	return nil
}

// resolveDest decides where bytes go. "-" → stdout. "" → ./<api-name>. A value
// that names an existing directory, or ends in a path separator, → <dir>/<api-name>.
// Any other value is a literal path. The API name is validated with SafeFileName
// only when pfix (not the user) supplies the final segment.
func resolveDest(ctx context.Context, o *downloadOptions, client *planfix.Client, idPath string) (string, bool, error) {
	if o.output == "-" {
		return "", true, nil
	}
	fetchName := func() (string, error) {
		raw, err := client.JSON(ctx, "GET", idPath, nil)
		if err != nil {
			return "", cmdutil.DescribeAPIError(err)
		}
		var env struct {
			File struct {
				Name string `json:"name"`
			} `json:"file"`
		}
		if err := cmdutil.DecodeJSON(raw, &env); err != nil {
			return "", err
		}
		return cmdutil.SafeFileName(env.File.Name)
	}
	if o.output == "" {
		name, err := fetchName()
		if err != nil {
			return "", false, err
		}
		return name, false, nil
	}
	isDir := strings.HasSuffix(o.output, string(os.PathSeparator))
	if !isDir {
		if fi, err := os.Stat(o.output); err == nil && fi.IsDir() {
			isDir = true
		}
	}
	if isDir {
		name, err := fetchName()
		if err != nil {
			return "", false, err
		}
		return filepath.Join(o.output, name), false, nil
	}
	return o.output, false, nil
}

func cleanupPartial(f *os.File) {
	if f != nil {
		f.Close()
		os.Remove(f.Name())
	}
}
