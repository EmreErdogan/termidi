package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCollectFiles(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"b.mid", "a.MIDI", "notes.txt"} {
		os.WriteFile(filepath.Join(dir, n), nil, 0o644)
	}
	os.Mkdir(filepath.Join(dir, "sub.mid"), 0o755)

	got, err := collectFiles([]string{"example.mid", dir})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"example.mid", filepath.Join(dir, "a.MIDI"), filepath.Join(dir, "b.mid")}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := collectFiles([]string{filepath.Join(dir, "missing.mid")}); err == nil {
		t.Error("missing file: want error")
	}
}
