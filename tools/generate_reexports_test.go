package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestMain_NoArguments(t *testing.T) {
	// Use subprocess to test the actual binary behavior
	cmd := exec.Command("go", "run", "generate_reexports.go")
	cmd.Dir = "."
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with error
	if err == nil {
		t.Error("Expected command to fail with no arguments, but it succeeded")
	}

	output := stderr.String()

	// Verify error message
	if !strings.Contains(output, "Usage:") {
		t.Errorf("Expected usage message in stderr, got: %s", output)
	}
	if !strings.Contains(output, "Example:") {
		t.Errorf("Expected example in stderr, got: %s", output)
	}
}

func TestMain_FiberAdapter(t *testing.T) {
	// Use subprocess to test the actual binary behavior
	cmd := exec.Command("go", "run", "generate_reexports.go", "fiber")
	cmd.Dir = "."
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String()

	// Verify output contains expected content
	expectedStrings := []string{
		"package fiber",
		"DefaultHeaderKey",
		"DefaultGenerator",
		"FastGenerator",
		"FromContext",
		"MustFromContext",
		"NewContext",
		"github.com/hiiamtin/goctxid",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
		}
	}

	// Verify it does NOT contain fibernative-specific comments
	if strings.Contains(output, "FromLocals") {
		t.Errorf("Fiber adapter should not contain fibernative-specific functions")
	}
}

func TestMain_FibernativeAdapter(t *testing.T) {
	// Use subprocess to test the actual binary behavior
	cmd := exec.Command("go", "run", "generate_reexports.go", "fibernative")
	cmd.Dir = "."
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String()

	// Verify output contains expected content
	expectedStrings := []string{
		"package fibernative",
		"DefaultHeaderKey",
		"DefaultGenerator",
		"FastGenerator",
		"github.com/hiiamtin/goctxid",
		"FromLocals",
		"MustFromLocals",
		"NOT re-exported",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
		}
	}

	// Verify it does NOT contain context functions
	unexpectedStrings := []string{
		"func FromContext",
		"func MustFromContext",
		"func NewContext",
	}

	for _, unexpected := range unexpectedStrings {
		if strings.Contains(output, unexpected) {
			t.Errorf("Fibernative adapter should not contain %q", unexpected)
		}
	}
}

func TestMain_GinAdapter(t *testing.T) {
	// Use subprocess to test the actual binary behavior
	cmd := exec.Command("go", "run", "generate_reexports.go", "gin")
	cmd.Dir = "."
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String()

	// Verify output contains expected content
	if !strings.Contains(output, "package gin") {
		t.Errorf("Expected output to contain 'package gin', got: %s", output)
	}
}

func TestMain_EchoAdapter(t *testing.T) {
	// Use subprocess to test the actual binary behavior
	cmd := exec.Command("go", "run", "generate_reexports.go", "echo")
	cmd.Dir = "."
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String()

	// Verify output contains expected content
	if !strings.Contains(output, "package echo") {
		t.Errorf("Expected output to contain 'package echo', got: %s", output)
	}
}

// TestTemplateData verifies the TemplateData struct
func TestTemplateData(t *testing.T) {
	data := TemplateData{
		Package: "test",
	}

	if data.Package != "test" {
		t.Errorf("Expected Package to be 'test', got %s", data.Package)
	}
}

// TestGenerateReexports_Fiber tests the generateReexports function with fiber adapter
func TestGenerateReexports_Fiber(t *testing.T) {
	var buf bytes.Buffer
	err := generateReexports("fiber", &buf)
	if err != nil {
		t.Fatalf("generateReexports failed: %v", err)
	}

	output := buf.String()

	// Verify output contains expected content
	expectedStrings := []string{
		"package fiber",
		"DefaultHeaderKey",
		"DefaultGenerator",
		"FastGenerator",
		"FromContext",
		"MustFromContext",
		"NewContext",
		"github.com/hiiamtin/goctxid",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't", expected)
		}
	}
}

// TestGenerateReexports_Fibernative tests the generateReexports function with fibernative adapter
func TestGenerateReexports_Fibernative(t *testing.T) {
	var buf bytes.Buffer
	err := generateReexports("fibernative", &buf)
	if err != nil {
		t.Fatalf("generateReexports failed: %v", err)
	}

	output := buf.String()

	// Verify output contains expected content
	expectedStrings := []string{
		"package fibernative",
		"DefaultHeaderKey",
		"DefaultGenerator",
		"FastGenerator",
		"FromLocals",
		"NOT re-exported",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't", expected)
		}
	}

	// Verify it does NOT contain context functions
	unexpectedStrings := []string{
		"func FromContext",
		"func MustFromContext",
		"func NewContext",
	}

	for _, unexpected := range unexpectedStrings {
		if strings.Contains(output, unexpected) {
			t.Errorf("Fibernative adapter should not contain %q", unexpected)
		}
	}
}

// TestGenerateReexports_Gin tests the generateReexports function with gin adapter
func TestGenerateReexports_Gin(t *testing.T) {
	var buf bytes.Buffer
	err := generateReexports("gin", &buf)
	if err != nil {
		t.Fatalf("generateReexports failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "package gin") {
		t.Errorf("Expected output to contain 'package gin'")
	}
}

// TestGenerateReexports_Echo tests the generateReexports function with echo adapter
func TestGenerateReexports_Echo(t *testing.T) {
	var buf bytes.Buffer
	err := generateReexports("echo", &buf)
	if err != nil {
		t.Fatalf("generateReexports failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "package echo") {
		t.Errorf("Expected output to contain 'package echo'")
	}
}

// errorWriter is a writer that always returns an error
type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, bytes.ErrTooLarge
}

// TestGenerateReexports_WriteError tests error handling when writing fails
func TestGenerateReexports_WriteError(t *testing.T) {
	writer := &errorWriter{}
	err := generateReexports("fiber", writer)
	if err == nil {
		t.Error("Expected error when writer fails, but got nil")
	}

	if !strings.Contains(err.Error(), "error executing template") {
		t.Errorf("Expected error message to contain 'error executing template', got: %v", err)
	}
}
