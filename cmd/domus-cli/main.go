// Package main provides the command-line administration tool for Domus.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/domus-hub/domus/internal/application/backup"
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

	case "backup":
		runBackup(os.Args[2:])

	case "restore":
		runRestore(os.Args[2:])

	case "help":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// ─── backup ───────────────────────────────────────────────────────────────────

func runBackup(args []string) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	dbPath := fs.String("db", "./data/domus.db", "Path to the SQLite database file")
	cfgPath := fs.String("config", "./domus.toml", "Path to the TOML configuration file")
	outDir := fs.String("out", ".", "Directory where the backup archive will be written")
	passphrase := fs.String("passphrase", "", "Encryption passphrase (omit for plain backup)")
	_ = fs.Parse(args)

	svc := backup.NewService()
	result, err := svc.Create(backup.CreateOptions{
		DBPath:     *dbPath,
		ConfigPath: *cfgPath,
		OutputDir:  *outDir,
		Passphrase: *passphrase,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "backup failed: %v\n", err)
		os.Exit(1)
	}

	encStatus := "plain"
	if result.Encrypted {
		encStatus = "encrypted (AES-256-GCM)"
	}
	fmt.Printf("Backup created successfully\n")
	fmt.Printf("  File:      %s\n", result.FilePath)
	fmt.Printf("  Size:      %d bytes\n", result.SizeBytes)
	fmt.Printf("  Encrypted: %s\n", encStatus)
	fmt.Printf("  Timestamp: %s\n", result.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
}

// ─── restore ──────────────────────────────────────────────────────────────────

func runRestore(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	archivePath := fs.String("archive", "", "Path to the backup archive (.tar.gz or .tar.gz.enc)")
	outDir := fs.String("out", ".", "Directory where the restored files will be written")
	passphrase := fs.String("passphrase", "", "Decryption passphrase (required for encrypted archives)")
	_ = fs.Parse(args)

	if *archivePath == "" {
		fmt.Fprintln(os.Stderr, "restore: -archive flag is required")
		fs.Usage()
		os.Exit(1)
	}

	svc := backup.NewService()
	err := svc.Restore(backup.RestoreOptions{
		ArchivePath: *archivePath,
		OutputDir:   *outDir,
		Passphrase:  *passphrase,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "restore failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Restore completed successfully\n")
	fmt.Printf("  Archive:   %s\n", *archivePath)
	fmt.Printf("  Output:    %s\n", *outDir)
}

// ─── help ─────────────────────────────────────────────────────────────────────

func printUsage() {
	fmt.Println(`Domus Administration CLI

Usage:
  domus-cli <command> [arguments]

Available Commands:
  version     Display Domus CLI version
  backup      Create a backup archive of the database and configuration
  restore     Restore Domus state from a backup archive
  help        Show this help message

backup flags:
  -db         Path to the SQLite database file   (default: ./data/domus.db)
  -config     Path to the TOML config file       (default: ./domus.toml)
  -out        Output directory                   (default: .)
  -passphrase Encryption passphrase              (omit for plain .tar.gz)

restore flags:
  -archive    Path to the archive file           (required)
  -out        Output directory                   (default: .)
  -passphrase Decryption passphrase              (required for .tar.gz.enc)

Examples:
  # Plain backup
  domus-cli backup -db /var/lib/domus/domus.db -out /backups

  # Encrypted backup
  domus-cli backup -db /var/lib/domus/domus.db -out /backups -passphrase "s3cr3t"

  # Restore plain
  domus-cli restore -archive /backups/domus-backup-20260101-120000.tar.gz -out /var/lib/domus

  # Restore encrypted
  domus-cli restore -archive /backups/domus-backup-20260101-120000.tar.gz.enc -out /var/lib/domus -passphrase "s3cr3t"`)
	flag.PrintDefaults()
}
