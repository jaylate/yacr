package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/jaylate/yacr/runtime"
)

func TestRun_HelpReturnsZero(t *testing.T) {
	var stderr bytes.Buffer
	called := false

	exitCode := run([]string{"yacr", "--help"}, &stderr, func(command string, args []string, cfg *runtime.ContainerConfig) error {
		called = true
		return nil
	})

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if called {
		t.Fatal("runner should not be called for help")
	}
}

func TestRun_InvalidUsageShowsErrorThenUsage(t *testing.T) {
	var stderr bytes.Buffer

	exitCode := run([]string{"yacr", "run"}, &stderr, func(command string, args []string, cfg *runtime.ContainerConfig) error {
		return nil
	})

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	out := stderr.String()
	if !strings.HasPrefix(out, "Invalid usage\n") {
		t.Fatalf("stderr should start with error line, got %q", out)
	}
	if !strings.Contains(out, "Usage: yacr [options] run <command> [arguments]") {
		t.Fatalf("stderr missing usage, got %q", out)
	}
}

func TestRun_PassesConfigToRunner(t *testing.T) {
	var stderr bytes.Buffer
	called := false

	exitCode := run([]string{"yacr", "--hostname", "myhost", "--rootfs", "/tmp/root", "run", "/bin/sh", "-l"}, &stderr, func(command string, args []string, cfg *runtime.ContainerConfig) error {
		called = true
		if command != "/bin/sh" {
			t.Fatalf("command = %q, want %q", command, "/bin/sh")
		}
		if len(args) != 1 || args[0] != "-l" {
			t.Fatalf("args = %#v, want %#v", args, []string{"-l"})
		}
		if cfg == nil {
			t.Fatal("cfg should not be nil")
		}
		if cfg.Hostname != "myhost" {
			t.Fatalf("cfg.Hostname = %q, want %q", cfg.Hostname, "myhost")
		}
		if cfg.RootFS != "/tmp/root" {
			t.Fatalf("cfg.RootFS = %q, want %q", cfg.RootFS, "/tmp/root")
		}
		return nil
	})

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if !called {
		t.Fatal("runner should be called")
	}
}

func TestRun_RuntimeErrorIsReturnedAsExitOne(t *testing.T) {
	var stderr bytes.Buffer
	wantErr := errors.New("runner failed")

	exitCode := run([]string{"yacr", "run", "/bin/sh"}, &stderr, func(command string, args []string, cfg *runtime.ContainerConfig) error {
		return wantErr
	})

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "runner failed") {
		t.Fatalf("stderr missing runtime error, got %q", stderr.String())
	}
}
