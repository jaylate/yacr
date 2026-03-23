package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type ContainerConfig struct {
	InitBinary string
	RootFS     string
	Hostname   string
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
	return &LinuxExecutor{cfg: executorCfg}
}

func (e *LinuxExecutor) Execute(command string, args ...string) error {
	cmd := e.setupContainer(command, args)

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Failed to start child process: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Child process exited with error: %w", err)
	}

	return nil
}

func (e *LinuxExecutor) setupContainer(command string, args []string) *exec.Cmd {
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

	return cmd
}
