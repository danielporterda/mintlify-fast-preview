package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/danielporterda/mintlify-fast-preview/internal/preview"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "validate":
		validate(os.Args[2:])
	case "render":
		render(os.Args[2:])
	case "dev":
		dev(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func validate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	root := fs.String("root", ".", "docs root containing docs.json")
	_ = fs.Parse(args)
	if err := preview.Validate(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func render(args []string) {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	root := fs.String("root", ".", "docs root containing docs.json")
	out := fs.String("out", "dist", "output directory")
	_ = fs.Parse(args)
	if err := preview.RenderStatic(*root, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dev(args []string) {
	fs := flag.NewFlagSet("dev", flag.ExitOnError)
	root := fs.String("root", ".", "docs root containing docs.json")
	host := fs.String("host", "127.0.0.1", "host to bind")
	port := fs.Int("port", 3000, "port to bind")
	noOpen := fs.Bool("no-open", false, "do not open a browser")
	_ = fs.Parse(args)
	if err := preview.Serve(*root, *host, *port, *noOpen); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: mintfast <dev|validate|render> [options]")
}
