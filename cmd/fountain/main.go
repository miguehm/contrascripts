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

// cliFlags holds the parsed command line. It is produced by newFlagSet so that
// main and the tests register exactly the same flags.
type cliFlags struct {
	output      *string
	lint        *bool
	showHelp    *bool
	showVersion *bool
}

func main() {
	fs := flag.NewFlagSet("fountain", flag.ExitOnError)
	opts := newFlagSet(fs)

	// Usage printed because of an error goes to stderr; an explicit -h goes
	// to stdout so it can be piped.
	fs.Usage = func() { printUsage(os.Stderr, fs) }

	positional := parseArgs(fs, os.Args[1:])

	if *opts.showHelp {
		printUsage(os.Stdout, fs)
		os.Exit(0)
	}
	if *opts.showVersion {
		fmt.Printf("fountain v%s\n", version)
		os.Exit(0)
	}

	if len(positional) < 1 {
		fs.Usage()
		os.Exit(1)
	}

	inputPath := positional[0]
	var outPath string
	if *opts.output != "" {
		outPath = *opts.output
	} else if len(positional) > 1 {
		outPath = positional[1]
	}

	raw, doc, err := parseInput(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *opts.lint {
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

// newFlagSet registers the command line flags on fs. Both -h/-help and
// -v/-version are registered so the long spellings keep working; they used to
// be handled by a hand-rolled prescan that only matched exact tokens.
func newFlagSet(fs *flag.FlagSet) *cliFlags {
	showHelp := fs.Bool("h", false, "show help")
	showVersion := fs.Bool("v", false, "show version")
	fs.BoolVar(showHelp, "help", false, "show help")
	fs.BoolVar(showVersion, "version", false, "show version")

	return &cliFlags{
		output:      fs.String("o", "", "output PDF file path"),
		lint:        fs.Bool("lint", false, "print format warnings to stderr"),
		showHelp:    showHelp,
		showVersion: showVersion,
	}
}

// parseArgs collects positional arguments from argv, accepting flags before,
// between or after them.
//
// Go's flag package stops parsing at the first non-flag argument, so parsing
// argv in one pass made "fountain script.fountain -o out.pdf" treat -o as a
// second positional and write the PDF to a file literally named "-o". Parsing
// repeatedly and peeling off one positional at a time lets flags appear
// anywhere, which is what the README promised.
//
// Boolean flags never consume the following argument, and a lone "-" or a
// "--" terminator both land in the positional list, so stdin and literal
// leading-dash filenames keep working. Flag errors terminate the process via
// the FlagSet error handling, so no error is returned.
func parseArgs(fs *flag.FlagSet, argv []string) []string {
	var positional []string

	for rest := argv; len(rest) > 0; {
		if err := fs.Parse(rest); err != nil {
			return positional
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		rest = rest[1:]
	}

	return positional
}

// printUsage writes the usage text to w. flag.PrintDefaults writes to the
// FlagSet output rather than to a caller-supplied writer, so the output is
// redirected and restored around the call.
func printUsage(w io.Writer, fs *flag.FlagSet) {
	fmt.Fprintf(w, "Usage: fountain [options] <input.fountain> [output.pdf]\n\n")
	fmt.Fprintf(w, "Options:\n")

	prev := fs.Output()
	fs.SetOutput(w)
	fs.PrintDefaults()
	fs.SetOutput(prev)

	fmt.Fprintf(w, "\nExamples:\n")
	fmt.Fprintf(w, "  fountain script.fountain              # -> script.pdf\n")
	fmt.Fprintf(w, "  fountain -o out.pdf script.fountain    # -> out.pdf\n")
	fmt.Fprintf(w, "  fountain script.fountain -o out.pdf    # -> out.pdf\n")
	fmt.Fprintf(w, "  fountain -                            # read from stdin\n")
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
