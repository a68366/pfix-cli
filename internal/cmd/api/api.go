package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type apiOptions struct {
	method    string
	methodSet bool
	fields    []string // -F typed
	rawFields []string // -f string
	headers   []string // -H
	inputFile string   // --input
	include   bool     // -i
	silent    bool

	client func() (*planfix.Client, error)
	in     io.Reader
	out    io.Writer
}

// NewCmd builds the `api` command.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &apiOptions{}
	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "Make an authenticated request to any Planfix REST endpoint",
		Long: "Make an authenticated request to any Planfix REST endpoint and print the response.\n\n" +
			"The default method is GET, or POST when a body or fields are supplied. Use --input to send a\n" +
			"raw JSON body (the primary path for nested Planfix requests), or -F/-f to set simple parameters.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.methodSet = cmd.Flags().Changed("method")
			o.client = func() (*planfix.Client, error) {
				c, _, err := g.Client()
				return c, err
			}
			o.in = cmd.InOrStdin()
			o.out = cmd.OutOrStdout()
			return runAPI(cmd.Context(), o, args[0])
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.method, "method", "X", "GET", "HTTP method")
	f.StringArrayVarP(&o.fields, "field", "F", nil, "Add a typed parameter key=value (ints/true/false/null/@file)")
	f.StringArrayVarP(&o.rawFields, "raw-field", "f", nil, "Add a string parameter key=value")
	f.StringArrayVarP(&o.headers, "header", "H", nil, "Add a request header key:value")
	f.StringVar(&o.inputFile, "input", "", "Request body from a file; '-' for stdin")
	f.BoolVarP(&o.include, "include", "i", false, "Include response status and headers in the output")
	f.BoolVar(&o.silent, "silent", false, "Do not print the response body")
	return cmd
}

func runAPI(ctx context.Context, o *apiOptions, path string) error {
	if o.inputFile != "" && (len(o.fields) > 0 || len(o.rawFields) > 0) {
		return fmt.Errorf("--input cannot be combined with -F/--field or -f/--raw-field")
	}

	params, err := parseFields(o.fields, o.rawFields, o.in)
	if err != nil {
		return err
	}

	method := o.method
	var body []byte
	switch {
	case o.inputFile != "":
		body, err = readInput(o.inputFile, o.in)
		if err != nil {
			return err
		}
		if !o.methodSet {
			method = http.MethodPost
		}
	case len(params) > 0:
		body, err = json.Marshal(params)
		if err != nil {
			return err
		}
		if !o.methodSet {
			method = http.MethodPost
		}
	}

	headers, err := parseHeaders(o.headers)
	if err != nil {
		return err
	}

	client, err := o.client()
	if err != nil {
		return err
	}

	resp, err := client.Do(ctx, method, path, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if o.include {
		fmt.Fprintf(o.out, "%s %s\n", resp.Proto, resp.Status)
		writeHeaders(o.out, resp.Header)
		fmt.Fprintln(o.out)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if !o.silent {
		printBody(o.out, data)
	}
	if resp.StatusCode >= 300 {
		return planfix.ParseError(resp.StatusCode, data)
	}
	return nil
}

func parseFields(typed, raw []string, stdin io.Reader) (map[string]any, error) {
	params := map[string]any{}
	for _, f := range raw {
		k, v, err := splitField(f)
		if err != nil {
			return nil, err
		}
		params[k] = v
	}
	for _, f := range typed {
		k, v, err := splitField(f)
		if err != nil {
			return nil, err
		}
		val, err := magicValue(v, stdin)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", k, err)
		}
		params[k] = val
	}
	return params, nil
}

func splitField(f string) (string, string, error) {
	i := strings.IndexByte(f, '=')
	if i < 0 {
		return "", "", fmt.Errorf("field %q must be key=value", f)
	}
	return f[:i], f[i+1:], nil
}

func magicValue(v string, stdin io.Reader) (any, error) {
	if strings.HasPrefix(v, "@") {
		b, err := readFileOrStdin(v[1:], stdin)
		return string(b), err
	}
	switch v {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n, nil
	}
	return v, nil
}

func readInput(name string, stdin io.Reader) ([]byte, error) {
	return readFileOrStdin(name, stdin)
}

func readFileOrStdin(name string, stdin io.Reader) ([]byte, error) {
	if name == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(name)
}

func parseHeaders(hs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, h := range hs {
		i := strings.IndexByte(h, ':')
		if i < 0 {
			return nil, fmt.Errorf("header %q must be key:value", h)
		}
		out[strings.TrimSpace(h[:i])] = strings.TrimSpace(h[i+1:])
	}
	return out, nil
}

func writeHeaders(w io.Writer, h http.Header) {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "%s: %s\n", k, strings.Join(h[k], ", "))
	}
}

func printBody(w io.Writer, data []byte) {
	if json.Valid(data) {
		var pretty bytes.Buffer
		if json.Indent(&pretty, data, "", "  ") == nil {
			w.Write(pretty.Bytes())
			fmt.Fprintln(w)
			return
		}
	}
	w.Write(data)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		fmt.Fprintln(w)
	}
}
