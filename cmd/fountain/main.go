package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/fountain/internal/parser"
	"github.com/user/fountain/internal/renderer"
)

const version = "1.0.0"

func main() {
	fs := flag.NewFlagSet("fountain", flag.ExitOnError)
	output := fs.String("o", "", "output PDF file path")
	lint := fs.Bool("lint", false, "print format warnings to stderr")

	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "-help" || arg == "--help" {
			printUsage(fs)
			os.Exit(0)
		}
		if arg == "-v" || arg == "-version" || arg == "--version" {
			fmt.Printf("fountain v%s\n", version)
			os.Exit(0)
		}
	}

	fs.Usage = func() {
		printUsage(fs)
	}

	fs.Parse(os.Args[1:])

	args := fs.Args()

	if len(args) < 1 {
		fs.Usage()
		os.Exit(1)
	}

	inputPath := args[0]
	var outPath string
	if *output != "" {
		outPath = *output
	} else if len(args) > 1 {
		outPath = args[1]
	}

	raw, doc, err := parseInput(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *lint {
		for _, w := range parser.Lint(raw) {
			fmt.Fprintf(os.Stderr, "warning: line %d [%s]: %s\n", w.Line, w.Code, w.Message)
		}
	}

	if outPath == "" {
		outPath = resolveOutputPath(inputPath)
	}

	r := renderer.New(doc)
	if err := r.RenderToFile(outPath); err != nil {
		fmt.Fprintf(os.Stderr, "error rendering PDF: %v\n", err)
		os.Exit(1)
	}
}

func printUsage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "Usage: fountain [options] <input.fountain> [output.pdf]\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fs.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  fountain script.fountain              # -> script.pdf\n")
	fmt.Fprintf(os.Stderr, "  fountain -o out.pdf script.fountain    # -> out.pdf\n")
	fmt.Fprintf(os.Stderr, "  fountain -                            # read from stdin\n")
}

func parseInput(inputPath string) (string, *parser.Fountain, error) {
	if inputPath == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", nil, fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), parser.Parse(string(data)), nil
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return "", nil, err
	}
	return string(data), parser.Parse(string(data)), nil
}

func resolveOutputPath(inputPath string) string {
	if inputPath == "-" {
		return "output.pdf"
	}
	ext := filepath.Ext(inputPath)
	return strings.TrimSuffix(inputPath, ext) + ".pdf"
}
