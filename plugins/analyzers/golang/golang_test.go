package golang_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mparvin/repo-miner/plugins/analyzers/golang"
)

func TestAnalyzeSimpleGoProject(t *testing.T) {
	dir := t.TempDir()
	src := `package main

// Add sums two integers.
func Add(a, b int) int {
	if a < 0 {
		return b
	}
	return a + b
}

type Config struct {
	Host string
}

type Reader interface {
	Read() error
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(`package main

func TestAdd(t *testing.T) {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ana, err := golang.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ana.Analyze(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Functions < 1 {
		t.Errorf("expected at least 1 function, got %d", result.Functions)
	}
	if !result.HasTests {
		t.Error("expected HasTests true")
	}
	if result.Structs < 1 {
		t.Errorf("expected at least 1 struct, got %d", result.Structs)
	}
	if result.Interfaces < 1 {
		t.Errorf("expected at least 1 interface, got %d", result.Interfaces)
	}
}
