package runtime

import (
	"strings"
	"syscall"
	"testing"

	"github.com/jaylate/yacr/bootstrap"
	"github.com/jaylate/yacr/runtime/resources"
	"github.com/jaylate/yacr/runtime/rootfs"
)

func TestRuntime_SysProcAttr(t *testing.T) {
	root := t.TempDir()
	rt, err := CreateRuntime(resources.ResourceLimits{})
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	container, err := rt.CreateContainer(ContainerConfig{
		ContainerID: "test-container",
		RootFS:      root,
		Hostname:    "container",
	})
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	err = container.StartContainer("/bin/sh", "-l")
	if err != nil &&
		!strings.Contains(err.Error(), "failed to start process") &&
		!strings.Contains(err.Error(), "process exited with error") {
		t.Fatalf("StartContainer returned unexpected error: %v", err)
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

	expectedArgs := append([]string{self}, bootstrap.CommandArgs("container", root, "/bin/sh", []string{"-l"})...)
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
		cfg          ContainerConfig
		command      string
		args         []string
		wantHostname string
		wantCommand  string
		wantArgs     []string
	}{
		{
			name: "simple command",
			cfg: ContainerConfig{
				ContainerID: "test-container",
			},
			command:      "/bin/sh",
			args:         []string{},
			wantHostname: "container",
			wantCommand:  "/bin/sh",
			wantArgs:     nil,
		},
		{
			name: "command with args",
			cfg: ContainerConfig{
				ContainerID: "test-container",
			},
			command:      "/bin/sh",
			args:         []string{"-l", "-a"},
			wantHostname: "container",
			wantCommand:  "/bin/sh",
			wantArgs:     []string{"-l", "-a"},
		},
		{
			name: "custom hostname via provider",
			cfg: ContainerConfig{
				ContainerID:    "test-container",
				RootFSProvider: rootfs.StaticProvider{Path: t.TempDir()},
				Hostname:       "myhost",
			},
			command:      "/bin/sh",
			args:         []string{},
			wantHostname: "myhost",
			wantCommand:  "/bin/sh",
			wantArgs:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.RootFSProvider == nil && tt.cfg.RootFS == "" {
				tt.cfg.RootFS = t.TempDir()
			}

			rt, err := CreateRuntime(resources.ResourceLimits{})
			if err != nil {
				t.Fatalf("Failed to create runtime: %v", err)
			}

			container, err := rt.CreateContainer(tt.cfg)
			if err != nil {
				t.Fatalf("Failed to create container: %v", err)
			}

			err = container.StartContainer(tt.command, tt.args...)
			if err != nil &&
				!strings.Contains(err.Error(), "failed to start process") &&
				!strings.Contains(err.Error(), "process exited with error") {
				t.Fatalf("StartContainer returned unexpected error: %v", err)
			}

			cmd := container.cmd
			if cmd == nil {
				t.Fatal("cmd should not be nil after Start")
			}

			wantInitArgs := append([]string{self}, bootstrap.CommandArgs(
				tt.wantHostname,
				container.rootfsPath,
				tt.wantCommand,
				tt.wantArgs,
			)...)
			for i, want := range wantInitArgs {
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

func TestCreateContainer_MissingRootFS(t *testing.T) {
	rt, err := CreateRuntime(resources.ResourceLimits{})
	if err != nil {
		t.Fatalf("CreateRuntime: %v", err)
	}
	_, err = rt.CreateContainer(ContainerConfig{
		RootFS: "/no/such/rootfs-" + t.Name(),
	})
	if err == nil {
		t.Fatal("expected error for missing rootfs")
	}
	if !strings.Contains(err.Error(), "resolve rootfs") {
		t.Errorf("error = %v, want resolve rootfs", err)
	}
}

func TestCreateContainer_InjectsIO(t *testing.T) {
	var stdout, stderr strings.Builder
	stdin := strings.NewReader("")

	rt, err := CreateRuntime(resources.ResourceLimits{})
	if err != nil {
		t.Fatalf("CreateRuntime: %v", err)
	}
	container, err := rt.CreateContainer(ContainerConfig{
		RootFS: t.TempDir(),
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("CreateContainer: %v", err)
	}
	if container.cmd.Stdin != stdin {
		t.Fatal("Stdin was not injected")
	}
	if container.cmd.Stdout != &stdout {
		t.Fatal("Stdout was not injected")
	}
	if container.cmd.Stderr != &stderr {
		t.Fatal("Stderr was not injected")
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
