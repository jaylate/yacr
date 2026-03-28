package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/jaylate/yacr/runtime/resources"
)

type ContainerConfig struct {
	ContainerID string // Optional: if empty, one will be generated
	InitBinary  string
	RootFS      string
	Hostname    string
	Limits      resources.ResourceLimits
}

func DefaultContainerConfig() *ContainerConfig {
	return &ContainerConfig{
		InitBinary: "./bin/init",
		RootFS:     "rootfs",
		Hostname:   "container",
	}
}

type LinuxExecutor struct {
	cfg ContainerConfig
	mgr resources.CgroupManager
}

func NewLinuxExecutor(cfg *ContainerConfig) *LinuxExecutor {
	defaults := DefaultContainerConfig()
	if cfg == nil {
		return &LinuxExecutor{cfg: *defaults}
	}
	// Merge with defaults
	executorCfg := *defaults
	if cfg.InitBinary != "" {
		executorCfg.InitBinary = cfg.InitBinary
	}
	if cfg.RootFS != "" {
		executorCfg.RootFS = cfg.RootFS
	}
	if cfg.Hostname != "" {
		executorCfg.Hostname = cfg.Hostname
	}
	if cfg.Limits.MemoryBytes != 0 || cfg.Limits.CPUCores != 0 || cfg.Limits.PIDsMax != 0 {
		executorCfg.Limits = cfg.Limits
	}
	return &LinuxExecutor{cfg: executorCfg, mgr: nil}
}

func NewLinuxExecutorWithCgroups(cfg *ContainerConfig, mgr resources.CgroupManager) *LinuxExecutor {
	executor := NewLinuxExecutor(cfg)
	executor.mgr = mgr

	return executor
}

func (e *LinuxExecutor) Execute(command string, args ...string) error {
	// Use ContainerID from config if provided, otherwise generate one
	if e.cfg.ContainerID == "" {
		containerID, err := resources.GenerateContainerID()
		if err != nil {
			return fmt.Errorf("failed to generate container ID: %w", err)
		}
		e.cfg.ContainerID = containerID
	}
	containerID := e.cfg.ContainerID

	cmd, err := e.setupContainer(command, args)
	if err != nil {
		return fmt.Errorf("failed to setup container: %w", err)
	}

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		if e.mgr != nil {
			e.mgr.Destroy(containerID)
		}
		return fmt.Errorf("failed to start child process: %w", err)
	}

	if e.mgr != nil && cmd.Process != nil {
		if err := e.mgr.AddProcess(containerID, cmd.Process.Pid); err != nil {
			cmd.Process.Kill()
			cmd.Wait()
			if destroyErr := e.mgr.Destroy(containerID); destroyErr != nil {
				return fmt.Errorf("failed to add process to cgroup: %w (additionally failed to destroy cgroup: %v)", err, destroyErr)
			}
			return fmt.Errorf("failed to add process to cgroup: %w", err)
		}
	}

	if err := cmd.Wait(); err != nil {
		var destroyErr error
		if e.mgr != nil {
			destroyErr = e.mgr.Destroy(containerID)
		}
		if destroyErr != nil {
			return fmt.Errorf("child process exited with error: %w (additionally failed to destroy cgroup: %v)", err, destroyErr)
		}
		return fmt.Errorf("child process exited with error: %w", err)
	}

	// Cleanup cgroup on success
	if e.mgr != nil {
		if err := e.mgr.Destroy(containerID); err != nil {
			return fmt.Errorf("failed to destroy cgroup for container %s: %w", containerID, err)
		}
	}

	return nil
}

func (e *LinuxExecutor) setupContainer(command string, args []string) (*exec.Cmd, error) {
	if e.mgr != nil {
		if err := e.mgr.Create(e.cfg.ContainerID, e.cfg.Limits); err != nil {
			return nil, err
		}
	}

	initArgs := []string{
		e.cfg.InitBinary,
		"--hostname", e.cfg.Hostname,
		"--rootfs", e.cfg.RootFS,
		"--",
		command,
	}
	initArgs = append(initArgs, args...)

	cmd := exec.Command(e.cfg.InitBinary, initArgs[1:]...)
	cmd.Args = initArgs
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	return cmd, nil
}
