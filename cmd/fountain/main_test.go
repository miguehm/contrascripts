package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get test file path")
	}
	return filepath.Dir(filename)
}

func TestCLIFileInput(t *testing.T) {
	root := projectRoot(t)
	inputPath := filepath.Join(root, "testdata", "minimal.fountain")
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skipf("testdata file not found: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "output.pdf")

	// Flags must come before positional args with flag.NewFlagSet
	cmd := exec.Command("go", "run", ".", "-o", outPath, inputPath)
	cmd.Dir = root

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput: %s", err, output)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("output PDF was not created.\nCLI output: %s", output)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("failed to stat output: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output PDF is empty")
	}
}

func TestCLIDefaultOutput(t *testing.T) {
	root := projectRoot(t)
	inputPath := filepath.Join(root, "testdata", "minimal.fountain")
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skipf("testdata file not found: %v", err)
	}

	tmpDir := t.TempDir()
	tmpInput := filepath.Join(tmpDir, "test.fountain")
	data, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("failed to read input: %v", err)
	}
	if err := os.WriteFile(tmpInput, data, 0o644); err != nil {
		t.Fatalf("failed to write temp input: %v", err)
	}

	cmd := exec.Command("go", "run", ".", tmpInput)
	cmd.Dir = root

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput: %s", err, output)
	}

	expectedOutput := filepath.Join(tmpDir, "test.pdf")
	if _, err := os.Stat(expectedOutput); os.IsNotExist(err) {
		t.Fatalf("expected output PDF %s was not created", expectedOutput)
	}
}

func TestCLIOutputAsPositionalArg(t *testing.T) {
	root := projectRoot(t)
	inputPath := filepath.Join(root, "testdata", "minimal.fountain")
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skipf("testdata file not found: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "positional.pdf")

	// Output as second positional arg
	cmd := exec.Command("go", "run", ".", inputPath, outPath)
	cmd.Dir = root

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI failed: %v\nOutput: %s", err, output)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("output PDF was not created.\nCLI output: %s", output)
	}
}

func TestCLILint(t *testing.T) {
	root := projectRoot(t)
	inputPath := filepath.Join(root, "testdata", "minimal.fountain")
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skipf("testdata file not found: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "lint.pdf")

	cmd := exec.Command("go", "run", ".", "--lint", "-o", outPath, inputPath)
	cmd.Dir = root

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI --lint failed: %v\nOutput: %s", err, output)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("output PDF was not created with --lint.\nCLI output: %s", output)
	}
}

func TestCLILintEmitsWarnings(t *testing.T) {
	root := projectRoot(t)
	tmpInput := filepath.Join(t.TempDir(), "warn.fountain")
	data := "INT. X - DAY\n\n/*\nnever closed\n"
	if err := os.WriteFile(tmpInput, []byte(data), 0o644); err != nil {
		t.Fatalf("failed to write temp input: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "warn.pdf")

	cmd := exec.Command("go", "run", ".", "--lint", "-o", outPath, tmpInput)
	cmd.Dir = root

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI --lint failed: %v\nOutput: %s", err, output)
	}

	if len(output) == 0 {
		t.Error("expected warnings on stderr/stdout with --lint")
	}
}
func TestCLINoInput(t *testing.T) {
	root := projectRoot(t)
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = root

	err := cmd.Run()
	if err == nil {
		t.Error("expected error exit code for no input")
	}
}

func TestCLIHelp(t *testing.T) {
	root := projectRoot(t)
	cmd := exec.Command("go", "run", ".", "-h")
	cmd.Dir = root

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	if len(output) == 0 {
		t.Error("expected help output")
	}
}
