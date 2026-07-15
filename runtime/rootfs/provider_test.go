package rootfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStaticProvider_Resolve(t *testing.T) {
	dir := t.TempDir()
	resolved, err := StaticProvider{Path: dir}.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Path != dir {
		t.Errorf("Path = %q, want %q", resolved.Path, dir)
	}
	if resolved.Cleanup != nil {
		t.Error("Cleanup should be nil for StaticProvider")
	}
}

func TestStaticProvider_Missing(t *testing.T) {
	_, err := StaticProvider{Path: filepath.Join(t.TempDir(), "missing")}.Resolve()
	if err == nil {
		t.Fatal("expected error for missing rootfs")
	}
}

func TestStaticProvider_NotDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := StaticProvider{Path: file}.Resolve()
	if err == nil {
		t.Fatal("expected error when rootfs is a file")
	}
}

func TestStaticProvider_Empty(t *testing.T) {
	_, err := StaticProvider{}.Resolve()
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestFromPath(t *testing.T) {
	p := FromPath("")
	sp, ok := p.(StaticProvider)
	if !ok {
		t.Fatalf("got %T, want StaticProvider", p)
	}
	if sp.Path != "rootfs" {
		t.Errorf("Path = %q, want rootfs", sp.Path)
	}
}
