package cmdutil

// GlobalOpts holds persistent flags shared by every subcommand.
type GlobalOpts struct {
	Profile string
	Domain  string
	JSON    bool
	Fields  string
	Quiet   bool
}
