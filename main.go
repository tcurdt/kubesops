package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define flags
	verbose := flag.Bool("verbose", false, "Verbose output (for diff command)")
	force := flag.Bool("force", false, "Force upload even if no changes detected (for upload command)")
	doit := flag.Bool("doit", false, "Actually perform the upload; default is dry-run (for upload command)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <command> [path...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  upload [path]         Upload secrets to Kubernetes (default: secrets/)\n")
		fmt.Fprintf(os.Stderr, "  download [path]       Download secrets from Kubernetes (default: secrets/)\n")
		fmt.Fprintf(os.Stderr, "  diff [path1] [path2]  Compare secrets (1 path: local vs remote, 2 paths: local vs local)\n")
		fmt.Fprintf(os.Stderr, "  manifest [path]       Print secrets as YAML manifests (default: secrets/)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s upload                                    # Upload all secrets (dry-run)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -doit upload                              # Actually upload changed secrets\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -doit -force upload                       # Force upload all secrets\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s upload secrets                            # Upload all secrets (dry-run)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -doit upload secrets/test/dotenv.env      # Upload one secret\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s download                                  # Download all secrets\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s download secrets/test/dotenv.env          # Download one secret\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s diff                                      # Diff all (local vs remote)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s diff secrets/test/dotenv.env              # Diff one (local vs remote)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s diff secrets/live/db.env secrets/test/db.env # Diff two local files\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -verbose diff                             # Diff with full values\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s manifest                                  # Print all manifests\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s manifest secrets/test                     # Print manifests for test namespace\n", os.Args[0])
	}

	flag.Parse()

	// Get command
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: missing command\n\n")
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	// Validate command
	if command != "upload" && command != "download" && command != "diff" && command != "manifest" {
		fmt.Fprintf(os.Stderr, "Error: invalid command '%s'\n\n", command)
		flag.Usage()
		os.Exit(1)
	}

	// Get path arguments with defaults
	path1 := "secrets"
	path2 := ""

	if flag.NArg() > 1 {
		path1 = flag.Arg(1)
	}
	if flag.NArg() > 2 {
		path2 = flag.Arg(2)
	}

	// Execute command
	switch command {
	case "upload":
		if err := handleUpload(path1, *force, *doit, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Upload failed: %v\n", err)
			os.Exit(1)
		}

	case "download":
		if err := handleDownload(path1); err != nil {
			fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
			os.Exit(1)
		}

	case "diff":
		if err := handleDiff(path1, path2, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Diff failed: %v\n", err)
			os.Exit(1)
		}

	case "manifest":
		if err := handleManifest(path1); err != nil {
			fmt.Fprintf(os.Stderr, "Manifest failed: %v\n", err)
			os.Exit(1)
		}
	}
}
