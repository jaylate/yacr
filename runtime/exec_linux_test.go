package runtime

import (
	"strings"
	"syscall"
	"testing"

	"github.com/jaylate/yacr/bootstrap"
	"github.com/jaylate/yacr/runtime/resources"
)

func TestRuntime_SysProcAttr(t *testing.T) {
	rt, err := CreateRuntime(resources.ResourceLimits{})
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	container, err := rt.CreateContainer(ContainerConfig{
		ContainerID: "test-container",
		RootFS:      "rootfs",
		Hostname:    "container",
	})
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	err = container.StartContainer("/bin/sh", "-l")
	if err == nil {
		t.Fatal("Expected error from Start (no real rootfs), got nil")
	}

	cmd := container.cmd
	if cmd == nil {
		t.Fatal("cmd should not be nil after Start")
	}

	self, err := bootstrap.Self()
	if err != nil {
		t.Fatalf("bootstrap.Self: %v", err)
	}
	if cmd.Path != self {
		t.Errorf("cmd.Path = %q, want %q", cmd.Path, self)
	}

	expectedArgs := append([]string{self}, bootstrap.CommandArgs("container", "rootfs", "/bin/sh", []string{"-l"})...)
	for i, arg := range expectedArgs {
		if cmd.Args[i] != arg {
			t.Errorf("Args[%d] = %q, want %q", i, cmd.Args[i], arg)
		}
	}

	foundMarker := false
	for _, env := range cmd.Env {
		if env == bootstrap.EnvMarker+"=1" {
			foundMarker = true
			break
		}
	}
	if !foundMarker {
		t.Errorf("cmd.Env missing %s=1", bootstrap.EnvMarker)
	}

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should not be nil")
	}

	wantFlags := uintptr(syscall.CLONE_NEWUSER | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID)
	if cmd.SysProcAttr.Cloneflags != wantFlags {
		t.Errorf("Cloneflags = %v, want %v", cmd.SysProcAttr.Cloneflags, wantFlags)
	}

	if len(cmd.SysProcAttr.UidMappings) != 1 {
		t.Errorf("UidMappings length = %d, want 1", len(cmd.SysProcAttr.UidMappings))
	}

	if cmd.SysProcAttr.UidMappings[0].ContainerID != 0 {
		t.Errorf("UidMappings[0].ContainerID = %d, want 0", cmd.SysProcAttr.UidMappings[0].ContainerID)
	}

	if cmd.SysProcAttr.UidMappings[0].Size != 1 {
		t.Errorf("UidMappings[0].Size = %d, want 1", cmd.SysProcAttr.UidMappings[0].Size)
	}

	if len(cmd.SysProcAttr.GidMappings) != 1 {
		t.Errorf("GidMappings length = %d, want 1", len(cmd.SysProcAttr.GidMappings))
	}

	if cmd.SysProcAttr.GidMappings[0].ContainerID != 0 {
		t.Errorf("GidMappings[0].ContainerID = %d, want 0", cmd.SysProcAttr.GidMappings[0].ContainerID)
	}
}

func TestRuntime_ConfigToArgs(t *testing.T) {
	self, err := bootstrap.Self()
	if err != nil {
		t.Fatalf("bootstrap.Self: %v", err)
	}

	tests := []struct {
		name         string
		cfg          *ContainerConfig
		command      string
		args         []string
		wantInitArgs []string
	}{
		{
			name: "simple command",
			cfg: &ContainerConfig{
				ContainerID: "test-container",
			},
			command:      "/bin/sh",
			args:         []string{},
			wantInitArgs: append([]string{self}, bootstrap.CommandArgs("container", "rootfs", "/bin/sh", nil)...),
		},
		{
			name: "command with args",
			cfg: &ContainerConfig{
				ContainerID: "test-container",
			},
			command:      "/bin/sh",
			args:         []string{"-l", "-a"},
			wantInitArgs: append([]string{self}, bootstrap.CommandArgs("container", "rootfs", "/bin/sh", []string{"-l", "-a"})...),
		},
		{
			name: "rootfs and hostname passed to bootstrap",
			cfg: &ContainerConfig{
				ContainerID: "test-container",
				RootFS:      "/custom/rootfs",
				Hostname:    "myhost",
			},
			command:      "/bin/sh",
			args:         []string{},
			wantInitArgs: append([]string{self}, bootstrap.CommandArgs("myhost", "/custom/rootfs", "/bin/sh", nil)...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt, err := CreateRuntime(resources.ResourceLimits{})
			if err != nil {
				t.Fatalf("Failed to create runtime: %v", err)
			}

			container, err := rt.CreateContainer(*tt.cfg)
			if err != nil {
				t.Fatalf("Failed to create container: %v", err)
			}

			err = container.StartContainer(tt.command, tt.args...)
			if err == nil {
				t.Fatal("Expected error from Start (no real rootfs), got nil")
			}

			cmd := container.cmd
			if cmd == nil {
				t.Fatal("cmd should not be nil after Start")
			}

			for i, want := range tt.wantInitArgs {
				if cmd.Args[i] != want {
					t.Errorf("Args[%d] = %q, want %q", i, cmd.Args[i], want)
				}
			}
			if !strings.Contains(strings.Join(cmd.Args, " "), bootstrap.ArgName) {
				t.Errorf("args should contain bootstrap token %q", bootstrap.ArgName)
			}
		})
	}
}

func TestDefaultContainerConfig(t *testing.T) {
	cfg := DefaultContainerConfig()

	if cfg.RootFS != "rootfs" {
		t.Errorf("RootFS = %q, want %q", cfg.RootFS, "rootfs")
	}
	if cfg.Hostname != "container" {
		t.Errorf("Hostname = %q, want %q", cfg.Hostname, "container")
	}
}
