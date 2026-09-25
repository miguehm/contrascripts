package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// fountainBin is the CLI built once by TestMain, so the end-to-end tests exec
// a real binary instead of paying for `go run` on every case.
var fountainBin string

// moduleRoot returns the module root. This file lives in cmd/fountain, so
// that is Dir(Dir(filename)); the module root is one level above it again.
func moduleRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to locate test file")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fountain-cli")
	if err != nil {
		fmt.Fprintf(os.Stderr, "temp dir: %v\n", err)
		os.Exit(1)
	}
	fountainBin = filepath.Join(dir, "fountain")

	build := exec.Command("go", "build", "-o", fountainBin, "./cmd/fountain")
	build.Dir = moduleRoot()
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()

	// Not deferred: os.Exit does not run deferred functions, and a deferred
	// cleanup would leak a temp dir on every test run.
	os.RemoveAll(dir)
	os.Exit(code)
}

// runCLI executes the built binary and returns its combined output.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	out, err := exec.Command(fountainBin, args...).CombinedOutput()
	return string(out), err
}

// fixture returns the path of a testdata script, failing the test if it is
// missing. These fixtures are versioned, so an absent one is a real problem
// and must be loud: an earlier t.Skipf here silently dropped CLI coverage when
// the test file moved and stopped resolving the module root.
func fixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(moduleRoot(), "testdata", name)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture %s: %v", path, err)
	}
	return path
}

func TestCLIFileInput(t *testing.T) {
	inputPath := fixture(t, "minimal.fountain")
	outPath := filepath.Join(t.TempDir(), "output.pdf")

	output, err := runCLI(t, "-o", outPath, inputPath)
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput: %s", err, output)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output PDF was not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output PDF is empty")
	}
}

// TestCLIFlagAfterPositional is the regression test for the bug that made
// `fountain script.fountain -o out.pdf` write the PDF to a file named "-o".
func TestCLIFlagAfterPositional(t *testing.T) {
	inputPath := fixture(t, "minimal.fountain")
	outPath := filepath.Join(t.TempDir(), "after.pdf")

	output, err := runCLI(t, inputPath, "-o", outPath)
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("trailing -o was not honoured: %v", err)
	}
}

func TestCLIDefaultOutput(t *testing.T) {
	data, err := os.ReadFile(fixture(t, "minimal.fountain"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	tmpDir := t.TempDir()
	tmpInput := filepath.Join(tmpDir, "test.fountain")
	if err := os.WriteFile(tmpInput, data, 0o644); err != nil {
		t.Fatalf("failed to write temp input: %v", err)
	}

	output, err := runCLI(t, tmpInput)
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput: %s", err, output)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "test.pdf")); err != nil {
		t.Fatalf("expected test.pdf next to the input: %v", err)
	}
}

func TestCLIOutputAsPositionalArg(t *testing.T) {
	inputPath := fixture(t, "minimal.fountain")
	outPath := filepath.Join(t.TempDir(), "positional.pdf")

	output, err := runCLI(t, inputPath, outPath)
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output PDF was not created.\nCLI output: %s", output)
	}
}

func TestCLILint(t *testing.T) {
	inputPath := fixture(t, "minimal.fountain")
	outPath := filepath.Join(t.TempDir(), "lint.pdf")

	output, err := runCLI(t, "--lint", "-o", outPath, inputPath)
	if err != nil {
		t.Fatalf("CLI --lint failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output PDF was not created with --lint.\nCLI output: %s", output)
	}
}

func TestCLILintEmitsWarnings(t *testing.T) {
	tmpInput := filepath.Join(t.TempDir(), "warn.fountain")
	if err := os.WriteFile(tmpInput, []byte("INT. X - DAY\n\n/*\nnever closed\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp input: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "warn.pdf")

	output, err := runCLI(t, "--lint", "-o", outPath, tmpInput)
	if err != nil {
		t.Fatalf("CLI --lint failed: %v\nOutput: %s", err, output)
	}
	if len(output) == 0 {
		t.Error("expected warnings on stderr/stdout with --lint")
	}
}

func TestCLINoInput(t *testing.T) {
	if _, err := runCLI(t); err == nil {
		t.Error("expected error exit code for no input")
	}
}

func TestCLIHelp(t *testing.T) {
	output, err := runCLI(t, "-h")
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}
	if len(output) == 0 {
		t.Error("expected help output")
	}
}

// TestCLIHelpGoesToStdout checks that explicit -h is on stdout, so it can be
// piped, while usage after an error stays on stderr.
func TestCLIHelpGoesToStdout(t *testing.T) {
	out, err := exec.Command(fountainBin, "-h").Output()
	if err != nil {
		t.Fatalf("-h failed: %v", err)
	}
	if !strings.Contains(string(out), "Usage: fountain") {
		t.Errorf("-h did not print usage on stdout, got %q", out)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(fountainBin)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Error("expected a non-zero exit for no input")
	}
	if strings.Contains(stdout.String(), "Usage: fountain") {
		t.Errorf("usage leaked to stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: fountain") {
		t.Errorf("usage missing from stderr: %q", stderr.String())
	}
}

func TestCLIVersion(t *testing.T) {
	for _, flag := range []string{"-v", "--version", "-version"} {
		output, err := runCLI(t, flag)
		if err != nil {
			t.Fatalf("%s failed: %v", flag, err)
		}
		if !strings.Contains(output, "fountain v"+version) {
			t.Errorf("%s printed %q, want the version", flag, output)
		}
	}
}

// TestParseArgs covers the flag/positional matrix in-process, which is cheap
// enough to be exhaustive. Cases mirror what users actually type, including
// the orders that used to break.
func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		argv        []string
		wantPos     []string
		wantOutput  string
		wantLint    bool
		wantHelp    bool
		wantVersion bool
	}{
		{
			name:       "flag before positional",
			argv:       []string{"-o", "out.pdf", "in.fountain"},
			wantPos:    []string{"in.fountain"},
			wantOutput: "out.pdf",
		},
		{
			name:       "flag after positional",
			argv:       []string{"in.fountain", "-o", "out.pdf"},
			wantPos:    []string{"in.fountain"},
			wantOutput: "out.pdf",
		},
		{
			name:       "flag between positionals",
			argv:       []string{"in.fountain", "-o", "out.pdf", "extra.pdf"},
			wantPos:    []string{"in.fountain", "extra.pdf"},
			wantOutput: "out.pdf",
		},
		{
			name:    "two positionals, no flag",
			argv:    []string{"in.fountain", "out.pdf"},
			wantPos: []string{"in.fountain", "out.pdf"},
		},
		{
			name:     "lint before positional",
			argv:     []string{"--lint", "in.fountain"},
			wantPos:  []string{"in.fountain"},
			wantLint: true,
		},
		{
			name:     "lint after positional",
			argv:     []string{"in.fountain", "--lint"},
			wantPos:  []string{"in.fountain"},
			wantLint: true,
		},
		{
			name:       "double dash flag spelling",
			argv:       []string{"in.fountain", "--o", "out.pdf"},
			wantPos:    []string{"in.fountain"},
			wantOutput: "out.pdf",
		},
		{
			name:       "equals form",
			argv:       []string{"in.fountain", "-o=out.pdf"},
			wantPos:    []string{"in.fountain"},
			wantOutput: "out.pdf",
		},
		{
			name:    "stdin is a positional",
			argv:    []string{"-", "-o", "out.pdf"},
			wantPos: []string{"-"},
			// main uses the first positional as input and -o as output, so
			// "-" must not be mistaken for a flag.
			wantOutput: "out.pdf",
		},
		{
			name:    "terminator protects leading dash filename",
			argv:    []string{"--", "-weird.fountain"},
			wantPos: []string{"-weird.fountain"},
		},
		{
			name:     "help before everything",
			argv:     []string{"-h", "in.fountain"},
			wantPos:  []string{"in.fountain"},
			wantHelp: true,
		},
		{
			name:     "help after everything",
			argv:     []string{"in.fountain", "--help"},
			wantPos:  []string{"in.fountain"},
			wantHelp: true,
		},
		{
			name:        "version",
			argv:        []string{"--version"},
			wantPos:     nil,
			wantVersion: true,
		},
		{
			name:    "no arguments",
			argv:    nil,
			wantPos: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("fountain", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			opts := newFlagSet(fs)

			got := parseArgs(fs, tt.argv)

			if !reflect.DeepEqual(got, tt.wantPos) && !(len(got) == 0 && len(tt.wantPos) == 0) {
				t.Errorf("positional = %#v, want %#v", got, tt.wantPos)
			}
			if *opts.output != tt.wantOutput {
				t.Errorf("output = %q, want %q", *opts.output, tt.wantOutput)
			}
			if *opts.lint != tt.wantLint {
				t.Errorf("lint = %v, want %v", *opts.lint, tt.wantLint)
			}
			if *opts.showHelp != tt.wantHelp {
				t.Errorf("showHelp = %v, want %v", *opts.showHelp, tt.wantHelp)
			}
			if *opts.showVersion != tt.wantVersion {
				t.Errorf("showVersion = %v, want %v", *opts.showVersion, tt.wantVersion)
			}
		})
	}
}
