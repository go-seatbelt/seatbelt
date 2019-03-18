package commands

import (
	"os"

	"github.com/mitchellh/cli"
)

// Commands is the mapping of all the available MetaBlog commands.
var Commands map[string]cli.CommandFactory

var ui = &cli.BasicUi{
	Reader:      os.Stdin,
	Writer:      os.Stdout,
	ErrorWriter: os.Stderr,
}

// New sets up the CLI.
func New(args []string) *cli.CLI {
	Commands := map[string]cli.CommandFactory{
		"init": func() (cli.Command, error) {
			return &InitCommand{
				ui: ui,
			}, nil
		},
		"extract": func() (cli.Command, error) {
			return &ExtractCommand{
				ui: ui,
			}, nil
		},
	}

	c := cli.NewCLI("seatbelt", "0.0.1")
	c.Args = args[1:]
	c.Commands = Commands

	return c
}
