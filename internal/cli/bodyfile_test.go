package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadBodyFileEmpty covers the silent-success defect: an empty (or
// whitespace-only) --body-file must be rejected loudly, not accepted and
// reported as "✓ created". This was the user's exact symptom: the CLI
// reported success while writing no content.
func TestReadBodyFileEmpty(t *testing.T) {
	dir := t.TempDir()

	empty := filepath.Join(dir, "empty.html")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	ws := filepath.Join(dir, "ws.html")
	if err := os.WriteFile(ws, []byte("   \n\t \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(dir, "real.html")
	want := []byte("<h1>x</h1>")
	if err := os.WriteFile(real, want, 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "nope.html")

	for _, tc := range []struct {
		name    string
		path    string
		wantErr bool
		substr  string
	}{
		{"empty file rejected", empty, true, "empty"},
		{"whitespace-only rejected", ws, true, "empty"},
		{"missing file rejected", missing, true, ""},
		{"non-empty accepted", real, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := readBodyFile(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got data=%q", tc.path, string(data))
				}
				if tc.substr != "" && !strings.Contains(err.Error(), tc.substr) {
					t.Errorf("error %q should contain %q", err.Error(), tc.substr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(data) != string(want) {
				t.Errorf("data = %q, want %q", string(data), string(want))
			}
		})
	}
}

// withWindows flips the goos seam to "windows" for the duration of a test and
// restores it. cygpath does not exist on non-Windows hosts, so the converter
// is also swapped via cygpathToWindows.
func withWindows(t *testing.T) {
	t.Helper()
	origGoos := goos
	origConv := cygpathToWindows
	goos = "windows"
	t.Cleanup(func() {
		goos = origGoos
		cygpathToWindows = origConv
	})
}

// TestReadBodyFileMsysConverted proves the happy path: on Windows, an MSYS
// path is run through cygpath and the content at the converted location is
// read. The converter seam points at a real temp file with real content.
func TestReadBodyFileMsysConverted(t *testing.T) {
	withWindows(t)
	want := []byte("<h1>real</h1>")
	convertedPath := filepath.Join(t.TempDir(), "real.html")
	if err := os.WriteFile(convertedPath, want, 0o644); err != nil {
		t.Fatal(err)
	}
	cygpathToWindows = func(string) (string, error) { return convertedPath, nil }

	data, err := readBodyFile("/tmp/lesson.html")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if string(data) != string(want) {
		t.Errorf("data = %q, want %q", string(data), string(want))
	}
}

// TestReadBodyFileMsysCygpathMissing covers the recovery path: cygpath is
// unavailable, conversion is skipped, and the raw read fails. The error must
// carry a Git Bash hint so the user knows to pass a Windows path.
func TestReadBodyFileMsysCygpathMissing(t *testing.T) {
	withWindows(t)
	cygpathToWindows = func(string) (string, error) { return "", fmt.Errorf("cygpath: not found") }

	_, err := readBodyFile("/c/Users/x/missing.html")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Git Bash") {
		t.Errorf("error should hint at Git Bash/MSYS, got: %v", err)
	}
}

// TestReadBodyFileMsysEmptyNoConversion covers the original silent-success
// defect on Windows: conversion was skipped (cygpath missing) and the raw
// read returned empty bytes. The error must flag it as empty AND mention the
// path-resolution risk.
func TestReadBodyFileMsysEmptyNoConversion(t *testing.T) {
	withWindows(t)
	cygpathToWindows = func(string) (string, error) { return "", fmt.Errorf("cygpath: not found") }

	empty := filepath.Join(t.TempDir(), "empty.html")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := readBodyFile(empty)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got: %v", err)
	}
	if !strings.Contains(err.Error(), "Git Bash") {
		t.Errorf("error should hint at Git Bash resolution, got: %v", err)
	}
}

// TestReadBodyFileMsysEmptyAfterConversion: when cygpath succeeded but the
// file at the converted path is genuinely empty, the error stays about
// emptiness without a misleading path hint.
func TestReadBodyFileMsysEmptyAfterConversion(t *testing.T) {
	withWindows(t)
	convertedPath := filepath.Join(t.TempDir(), "empty.html")
	if err := os.WriteFile(convertedPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cygpathToWindows = func(string) (string, error) { return convertedPath, nil }

	_, err := readBodyFile("/tmp/x.html")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got: %v", err)
	}
	if strings.Contains(err.Error(), "Git Bash") {
		t.Errorf("converted-but-empty must not give a path hint, got: %v", err)
	}
}

// TestLooksLikeMSYSPath pins the detection predicate.
func TestLooksLikeMSYSPath(t *testing.T) {
	cases := map[string]bool{
		"/c/Users/x":     true,
		"/tmp/foo":       true,
		"/etc/hosts":     true,
		"C:\\Users\\x":   false,
		"C:/Users/x":     false,
		"relative/path":  false,
		"//server/share": false,
		"":               false,
	}
	for p, want := range cases {
		if got := looksLikeMSYSPath(p); got != want {
			t.Errorf("looksLikeMSYSPath(%q) = %v, want %v", p, got, want)
		}
	}
}
