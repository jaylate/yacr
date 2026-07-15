// Package rootfs abstracts where the container root filesystem comes from.
// Callers can use a static path today; image unpackers can implement Provider later.
package rootfs

import (
	"fmt"
	"os"
)

// Resolved is a ready-to-use root filesystem tree.
type Resolved struct {
	Path string
	// Cleanup is optional. Called when the container is deleted; nil for
	// caller-owned directories (e.g. a pre-existing rootfs).
	Cleanup func() error
}

// Provider resolves a root filesystem for a container.
type Provider interface {
	Resolve() (*Resolved, error)
}

// StaticProvider uses an existing directory as the rootfs.
type StaticProvider struct {
	Path string
}

// Resolve checks that Path exists and is a directory.
func (p StaticProvider) Resolve() (*Resolved, error) {
	if p.Path == "" {
		return nil, fmt.Errorf("rootfs path is empty")
	}
	info, err := os.Stat(p.Path)
	if err != nil {
		return nil, fmt.Errorf("rootfs %q: %w", p.Path, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("rootfs %q is not a directory", p.Path)
	}
	return &Resolved{Path: p.Path}, nil
}

// FromPath returns a StaticProvider for path. Empty path uses "rootfs".
func FromPath(path string) Provider {
	if path == "" {
		path = "rootfs"
	}
	return StaticProvider{Path: path}
}
