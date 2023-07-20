package main

import (
	"os"
	"wp-go-static/cmd/wp-go-static/commands"
)

func main() {
	if err := commands.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
