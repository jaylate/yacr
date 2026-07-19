package bootstrap

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestIsBootstrap(t *testing.T) {
	origArgs := os.Args
	origEnv := os.Getenv(EnvMarker)
	t.Cleanup(func() {
		os.Args = origArgs
		if origEnv == "" {
			os.Unsetenv(EnvMarker)
		} else {
			os.Setenv(EnvMarker, origEnv)
		}
	})

	os.Unsetenv(EnvMarker)
	os.Args = []string{"yacr", "init"}
	if IsBootstrap() {
		t.Fatal("IsBootstrap should be false without env marker")
	}

	os.Setenv(EnvMarker, "1")
	os.Args = []string{"yacr"}
	if IsBootstrap() {
		t.Fatal("IsBootstrap should be false without init arg")
	}

	os.Args = []string{"yacr", "init"}
	if !IsBootstrap() {
		t.Fatal("IsBootstrap should be true with env marker and init arg")
	}
}

func TestWriteReadConfig_RoundTrip(t *testing.T) {
	want := Config{
		Hostname: "myhost",
		RootFS:   "/tmp/root",
		WorkDir:  "/app",
		Env:      []string{"PATH=/bin", "HOME=/root"},
		UID:      1000,
		GID:      1000,
		Command:  "/bin/sh",
		Args:     []string{"-c", "echo hi"},
	}

	var buf bytes.Buffer
	if err := WriteConfig(&buf, want); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	got, err := ReadConfig(&buf)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if got.Hostname != want.Hostname {
		t.Errorf("Hostname = %q, want %q", got.Hostname, want.Hostname)
	}
	if got.RootFS != want.RootFS {
		t.Errorf("RootFS = %q, want %q", got.RootFS, want.RootFS)
	}
	if got.WorkDir != want.WorkDir {
		t.Errorf("WorkDir = %q, want %q", got.WorkDir, want.WorkDir)
	}
	if got.UID != want.UID || got.GID != want.GID {
		t.Errorf("UID/GID = %d/%d, want %d/%d", got.UID, got.GID, want.UID, want.GID)
	}
	if got.Command != want.Command {
		t.Errorf("Command = %q, want %q", got.Command, want.Command)
	}
	if len(got.Args) != len(want.Args) {
		t.Fatalf("Args = %#v, want %#v", got.Args, want.Args)
	}
	for i := range want.Args {
		if got.Args[i] != want.Args[i] {
			t.Errorf("Args[%d] = %q, want %q", i, got.Args[i], want.Args[i])
		}
	}
	if len(got.Env) != len(want.Env) {
		t.Fatalf("Env = %#v, want %#v", got.Env, want.Env)
	}
	for i := range want.Env {
		if got.Env[i] != want.Env[i] {
			t.Errorf("Env[%d] = %q, want %q", i, got.Env[i], want.Env[i])
		}
	}
}

func TestWriteConfig_EndsWithNewline(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteConfig(&buf, Config{Command: "/bin/true", RootFS: "/r"}); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Errorf("WriteConfig output missing trailing newline: %q", buf.String())
	}
}

func TestConfig_ApplyDefaultsAndValidate(t *testing.T) {
	cfg := Config{RootFS: "/r", Command: "/bin/true"}
	cfg.applyDefaults()
	if cfg.WorkDir != "/" {
		t.Errorf("WorkDir default = %q, want /", cfg.WorkDir)
	}
	if err := cfg.validate(); err != nil {
		t.Errorf("validate valid config: %v", err)
	}

	if err := (Config{Command: "/bin/true"}).validate(); err == nil {
		t.Error("expected error for missing rootfs")
	}
	if err := (Config{RootFS: "/r"}).validate(); err == nil {
		t.Error("expected error for missing command")
	}
}

func TestReadConfig_InvalidJSON(t *testing.T) {
	_, err := ReadConfig(strings.NewReader("{not-json"))
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestDefaultMounts_Documented(t *testing.T) {
	// Podman-compatible /dev layout is built in prepareDev/finishDev.
	names := map[string]bool{}
	for _, d := range hostDevices {
		names[d.Name] = true
	}
	for _, want := range []string{"null", "zero", "full", "random", "urandom", "tty", "console"} {
		if !names[want] {
			t.Errorf("hostDevices missing %s", want)
		}
	}
}
