package runtime

import (
	"syscall"
	"testing"
)

func TestLinuxExecutor_SysProcAttr(t *testing.T) {
	executor := NewLinuxExecutor(&ContainerConfig{
		ContainerID: "test-container",
		InitBinary:  "./bin/init",
		RootFS:      "rootfs",
		Hostname:    "container",
	})

	cmd, err := executor.setupContainer("/bin/sh", []string{"-l"})

	if err != nil {
		t.Errorf("Failed to setup container: %v", err)
	}

	if cmd.Path != "./bin/init" {
		t.Errorf("cmd.Path = %q, want %q", cmd.Path, "./bin/init")
	}

	expectedArgs := []string{"./bin/init", "--hostname", "container", "--rootfs", "rootfs", "--", "/bin/sh", "-l"}
	for i, arg := range expectedArgs {
		if cmd.Args[i] != arg {
			t.Errorf("Args[%d] = %q, want %q", i, cmd.Args[i], arg)
		}
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

func TestLinuxExecutor_ConfigToArgs(t *testing.T) {
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
				InitBinary:  "./bin/init",
			},
			command:      "/bin/sh",
			args:         []string{},
			wantInitArgs: []string{"./bin/init", "--hostname", "container", "--rootfs", "rootfs", "--", "/bin/sh"},
		},
		{
			name: "command with args",
			cfg: &ContainerConfig{
				ContainerID: "test-container",
				InitBinary:  "./bin/init",
			},
			command:      "/bin/sh",
			args:         []string{"-l", "-a"},
			wantInitArgs: []string{"./bin/init", "--hostname", "container", "--rootfs", "rootfs", "--", "/bin/sh", "-l", "-a"},
		},
		{
			name: "rootfs and hostname passed to init",
			cfg: &ContainerConfig{
				ContainerID: "test-container",
				InitBinary:  "./bin/init",
				RootFS:      "/custom/rootfs",
				Hostname:    "myhost",
			},
			command:      "/bin/sh",
			args:         []string{},
			wantInitArgs: []string{"./bin/init", "--hostname", "myhost", "--rootfs", "/custom/rootfs", "--", "/bin/sh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewLinuxExecutor(tt.cfg)

			cmd, err := executor.setupContainer(tt.command, tt.args)
			if err != nil {
				t.Errorf("Failed to setup container: %v", err)
			}

			for i, want := range tt.wantInitArgs {
				if cmd.Args[i] != want {
					t.Errorf("Args[%d] = %q, want %q", i, cmd.Args[i], want)
				}
			}
		})
	}
}

func TestDefaultContainerConfig(t *testing.T) {
	cfg := DefaultContainerConfig()

	if cfg.InitBinary != "./bin/init" {
		t.Errorf("InitBinary = %q, want %q", cfg.InitBinary, "./bin/init")
	}
	if cfg.RootFS != "rootfs" {
		t.Errorf("RootFS = %q, want %q", cfg.RootFS, "rootfs")
	}
	if cfg.Hostname != "container" {
		t.Errorf("Hostname = %q, want %q", cfg.Hostname, "container")
	}
}
