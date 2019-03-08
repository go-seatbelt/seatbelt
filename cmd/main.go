package main

import (
	"os"

	"github.com/go-seatbelt/seatbelt/commands"
)

func main() {
	cli := commands.New(os.Args)
	code, err := cli.Run()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}
