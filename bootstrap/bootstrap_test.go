package bootstrap

import (
	"os"
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

func TestParseArgs(t *testing.T) {
	hostname, rootfs, command, args, err := parseArgs([]string{
		"--hostname", "myhost",
		"--rootfs", "/tmp/root",
		"--",
		"/bin/sh", "-l",
	})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if hostname != "myhost" {
		t.Errorf("hostname = %q, want %q", hostname, "myhost")
	}
	if rootfs != "/tmp/root" {
		t.Errorf("rootfs = %q, want %q", rootfs, "/tmp/root")
	}
	if command != "/bin/sh" {
		t.Errorf("command = %q, want %q", command, "/bin/sh")
	}
	if len(args) != 1 || args[0] != "-l" {
		t.Errorf("args = %#v, want [-l]", args)
	}
}

func TestParseArgs_Defaults(t *testing.T) {
	hostname, rootfs, command, args, err := parseArgs([]string{"--", "/bin/true"})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if hostname != "container" {
		t.Errorf("hostname = %q, want %q", hostname, "container")
	}
	if rootfs != "rootfs" {
		t.Errorf("rootfs = %q, want %q", rootfs, "rootfs")
	}
	if command != "/bin/true" {
		t.Errorf("command = %q, want %q", command, "/bin/true")
	}
	if len(args) != 0 {
		t.Errorf("args = %#v, want empty", args)
	}
}

func TestCommandArgs(t *testing.T) {
	got := CommandArgs("h", "/r", "/bin/sh", []string{"-c", "echo"})
	want := []string{"init", "--hostname", "h", "--rootfs", "/r", "--", "/bin/sh", "-c", "echo"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%#v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Args[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
