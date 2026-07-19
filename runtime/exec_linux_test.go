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

	// Start may fail before exec if bootstrap API is incomplete, or after
	// exec when the minimal rootfs cannot run the user process.
	err = container.StartContainer("/bin/sh", "-l")
	if err == nil {
		t.Fatal("Expected error from StartContainer (minimal rootfs), got nil")
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

	wantArgs := []string{self, bootstrap.ArgName}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("Args = %#v, want %#v", cmd.Args, wantArgs)
	}
	for i, arg := range wantArgs {
		if cmd.Args[i] != arg {
			t.Errorf("Args[%d] = %q, want %q", i, cmd.Args[i], arg)
		}
	}

	foundMarker := false
	foundConfigFD := false
	for _, env := range cmd.Env {
		if env == bootstrap.EnvMarker+"=1" {
			foundMarker = true
		}
		if strings.HasPrefix(env, bootstrap.ConfigFDEnv+"=") {
			foundConfigFD = true
			if env != bootstrap.ConfigFDEnv+"=3" {
				t.Errorf("config fd env = %q, want %s=3", env, bootstrap.ConfigFDEnv)
			}
		}
	}
	if !foundMarker {
		t.Errorf("cmd.Env missing %s=1", bootstrap.EnvMarker)
	}
	if !foundConfigFD {
		t.Errorf("cmd.Env missing %s", bootstrap.ConfigFDEnv)
	}

	if len(cmd.ExtraFiles) < 1 {
		t.Error("expected ExtraFiles to include config pipe read end")
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

func TestRuntime_BootstrapArgs(t *testing.T) {
	self, err := bootstrap.Self()
	if err != nil {
		t.Fatalf("bootstrap.Self: %v", err)
	}

	tests := []struct {
		name    string
		cfg     ContainerConfig
		command string
		args    []string
	}{
		{
			name: "simple command",
			cfg: ContainerConfig{
				ContainerID: "test-container",
			},
			command: "/bin/sh",
			args:    []string{},
		},
		{
			name: "command with args",
			cfg: ContainerConfig{
				ContainerID: "test-container",
			},
			command: "/bin/sh",
			args:    []string{"-l", "-a"},
		},
		{
			name: "custom hostname via provider",
			cfg: ContainerConfig{
				ContainerID:    "test-container",
				RootFSProvider: rootfs.StaticProvider{Path: t.TempDir()},
				Hostname:       "myhost",
			},
			command: "/bin/sh",
			args:    []string{},
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
			if err == nil {
				t.Fatal("Expected error from StartContainer (minimal rootfs), got nil")
			}

			cmd := container.cmd
			if cmd == nil {
				t.Fatal("cmd should not be nil after Start")
			}

			wantArgs := []string{self, bootstrap.ArgName}
			if len(cmd.Args) != len(wantArgs) {
				t.Fatalf("Args = %#v, want %#v", cmd.Args, wantArgs)
			}
			for i, want := range wantArgs {
				if cmd.Args[i] != want {
					t.Errorf("Args[%d] = %q, want %q", i, cmd.Args[i], want)
				}
			}
		})
	}
}

func TestCreateContainer_PreparesCmdWithoutStart(t *testing.T) {
	rt, err := CreateRuntime(resources.ResourceLimits{})
	if err != nil {
		t.Fatalf("CreateRuntime: %v", err)
	}
	root := t.TempDir()
	container, err := rt.CreateContainer(ContainerConfig{
		ContainerID: "prep-only",
		RootFS:      root,
		Hostname:    "prep-host",
		WorkDir:     "/tmp",
		Env:         []string{"FOO=bar"},
		UID:         1000,
		GID:         1000,
	})
	if err != nil {
		t.Fatalf("CreateContainer: %v", err)
	}

	if container.started {
		t.Fatal("CreateContainer should not start the process")
	}
	if container.cmd == nil {
		t.Fatal("cmd should be prepared")
	}
	if container.cmd.Process != nil {
		t.Fatal("Process should be nil before Start")
	}

	self, err := bootstrap.Self()
	if err != nil {
		t.Fatalf("bootstrap.Self: %v", err)
	}
	if len(container.cmd.Args) != 1 || container.cmd.Args[0] != self {
		t.Errorf("pre-Start Args = %#v, want [%q]", container.cmd.Args, self)
	}
	if container.Config.WorkDir != "/tmp" {
		t.Errorf("WorkDir = %q, want /tmp", container.Config.WorkDir)
	}
	if container.Config.UID != 1000 || container.Config.GID != 1000 {
		t.Errorf("UID/GID = %d/%d, want 1000/1000", container.Config.UID, container.Config.GID)
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
	if cfg.WorkDir != "/" {
		t.Errorf("WorkDir = %q, want %q", cfg.WorkDir, "/")
	}
}
