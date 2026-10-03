// Package main provides the command-line administration tool for Domus.
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	// Version is populated at build time via -ldflags.
	Version = "1.0.0-dev"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("domus-cli version %s\n", Version)

	case "help":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Domus Administration CLI

Usage:
  domus-cli <command> [arguments]

Available Commands:
  version     Display Domus CLI version
  backup      Create an encrypted or plain backup archive (MS-09)
  restore     Restore Domus state from backup archive (MS-09)
  migrate     Apply pending database migrations
  help        Show this help message`)
	flag.PrintDefaults()
}
