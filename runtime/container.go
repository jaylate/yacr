package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jaylate/yacr/bootstrap"
	"github.com/jaylate/yacr/runtime/resources"
	"github.com/jaylate/yacr/runtime/rootfs"
)

type ContainerConfig struct {
	ContainerID string
	// RootFS is a path to an existing root filesystem directory.
	// Ignored when RootFSProvider is set.
	RootFS string
	// RootFSProvider resolves the container root filesystem.
	// When nil, RootFS (or "rootfs") is wrapped in rootfs.StaticProvider.
	RootFSProvider rootfs.Provider
	Hostname       string
	// Env is the container process environment as KEY=VAL entries.
	Env []string
	// WorkDir is the working directory inside the container. Default "/".
	WorkDir string
	UID     uint32
	GID     uint32
	Limits  resources.ResourceLimits

	// Stdin, Stdout, and Stderr are attached to the container process.
	// When nil, the corresponding os.Std* stream is used.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func DefaultContainerConfig() *ContainerConfig {
	return &ContainerConfig{
		RootFS:   "rootfs",
		Hostname: "container",
		WorkDir:  "/",
	}
}

// ProcessExitError is returned when the container process exits with a
// non-zero status. ExitCode is the process exit code.
type ProcessExitError struct {
	Code int
}

func (e *ProcessExitError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.Code)
}

// ExitCode returns the container process exit code.
func (e *ProcessExitError) ExitCode() int { return e.Code }

func generateContainerID() (string, error) {
	b := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("failed to generate random container ID: %w", err)
	}
	return fmt.Sprintf("yacr-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b)), nil
}

type Container struct {
	ID               string
	Config           ContainerConfig
	cgroupManager    resources.CgroupManager
	namespaceManager NamespaceManager

	cmd           *exec.Cmd
	args          []string
	started       bool
	rootfsPath    string
	rootfsCleanup func() error

	signalDone chan struct{}
	signalOnce sync.Once
}

func (r *Runtime) CreateContainer(cfg ContainerConfig) (*Container, error) {
	defaults := DefaultContainerConfig()
	if cfg.Hostname == "" {
		cfg.Hostname = defaults.Hostname
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = defaults.WorkDir
	}

	provider := cfg.RootFSProvider
	if provider == nil {
		provider = rootfs.FromPath(cfg.RootFS)
	}
	resolved, err := provider.Resolve()
	if err != nil {
		return nil, fmt.Errorf("resolve rootfs: %w", err)
	}
	cfg.RootFS = resolved.Path

	containerID := cfg.ContainerID
	if containerID == "" {
		var err error
		containerID, err = generateContainerID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate container ID: %w", err)
		}
	}

	self, err := bootstrap.Self()
	if err != nil {
		_ = cleanupResolved(resolved)
		return nil, err
	}

	c := &Container{
		ID:               containerID,
		Config:           cfg,
		cgroupManager:    r.cgroupManager,
		namespaceManager: r.namespaceManager,
		rootfsPath:       resolved.Path,
		rootfsCleanup:    resolved.Cleanup,
	}

	cmd := exec.Command(self)
	cmd.Args = []string{self}
	cmd.Env = bootstrap.Env()
	cmd.Stdin = cfg.Stdin
	cmd.Stdout = cfg.Stdout
	cmd.Stderr = cfg.Stderr
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	if err := c.namespaceManager.Create(c.Config, cmd); err != nil {
		_ = cleanupResolved(resolved)
		if r.cgroupManager != nil {
			r.cgroupManager.Destroy(containerID)
		}
		return nil, fmt.Errorf("failed to setup namespace: %w", err)
	}

	c.cmd = cmd

	if r.cgroupManager != nil {
		if err := r.cgroupManager.Create(containerID, cfg.Limits); err != nil {
			_ = cleanupResolved(resolved)
			return nil, fmt.Errorf("failed to create cgroup: %w", err)
		}
	}

	return c, nil
}

func cleanupResolved(r *rootfs.Resolved) error {
	if r != nil && r.Cleanup != nil {
		return r.Cleanup()
	}
	return nil
}

// Start starts the container process without waiting for it to exit.
// It creates the bootstrap config pipe, re-execs with argv [self, "init"],
// writes the config after Start, adds the process to its cgroup, and begins
// forwarding SIGINT/SIGTERM to the child.
func (c *Container) Start(command string, args ...string) error {
	if c.cmd == nil {
		return fmt.Errorf("container not created")
	}
	if c.started {
		return fmt.Errorf("container already started")
	}

	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("create config pipe: %w", err)
	}

	// First ExtraFiles entry becomes fd 3 in the child; subsequent entries are 4, 5, …
	configFD := 3 + len(c.cmd.ExtraFiles)
	c.cmd.ExtraFiles = append(c.cmd.ExtraFiles, r)
	c.cmd.Args = []string{c.cmd.Path, bootstrap.ArgName}
	c.cmd.Env = append(bootstrap.Env(), fmt.Sprintf("%s=%d", bootstrap.ConfigFDEnv, configFD))

	cfg := bootstrap.Config{
		Hostname: c.Config.Hostname,
		RootFS:   c.rootfsPath,
		WorkDir:  c.Config.WorkDir,
		Env:      c.Config.Env,
		UID:      c.Config.UID,
		GID:      c.Config.GID,
		Command:  command,
		Args:     args,
	}

	if err := c.cmd.Start(); err != nil {
		r.Close()
		w.Close()
		if c.cgroupManager != nil {
			c.cgroupManager.Destroy(c.ID)
		}
		return fmt.Errorf("failed to start process: %w", err)
	}
	// Parent no longer needs the read end; the child inherited it.
	_ = r.Close()

	if err := bootstrap.WriteConfig(w, cfg); err != nil {
		_ = w.Close()
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
		if c.cgroupManager != nil {
			c.cgroupManager.Destroy(c.ID)
		}
		return fmt.Errorf("write bootstrap config: %w", err)
	}
	if err := w.Close(); err != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
		if c.cgroupManager != nil {
			c.cgroupManager.Destroy(c.ID)
		}
		return fmt.Errorf("close bootstrap config pipe: %w", err)
	}

	c.args = args
	c.started = true

	if c.cgroupManager != nil && c.cmd.Process != nil {
		if err := c.cgroupManager.AddProcess(c.ID, c.cmd.Process.Pid); err != nil {
			_ = c.cmd.Process.Kill()
			_ = c.cmd.Wait()
			if destroyErr := c.cgroupManager.Destroy(c.ID); destroyErr != nil {
				return fmt.Errorf("failed to add process to cgroup: %w (additionally failed to destroy cgroup: %v)", err, destroyErr)
			}
			return fmt.Errorf("failed to add process to cgroup: %w", err)
		}
	}

	c.startSignalForwarding()
	return nil
}

// Wait blocks until the container process exits, stops signal forwarding,
// and destroys the container cgroup if one was created. It does not delete
// the container rootfs; call DeleteContainer for that.
func (c *Container) Wait() (int, error) {
	if c.cmd == nil || c.cmd.Process == nil {
		return -1, fmt.Errorf("container not started")
	}

	defer c.stopSignalForwarding()
	defer func() {
		if c.cgroupManager != nil {
			_ = c.cgroupManager.Destroy(c.ID)
		}
	}()

	err := c.cmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return -1, fmt.Errorf("wait process: %w", err)
	}

	code := 0
	if c.cmd.ProcessState != nil {
		code = c.cmd.ProcessState.ExitCode()
	}
	return code, nil
}

// Kill sends sig to the container process.
func (c *Container) Kill(sig syscall.Signal) error {
	if c.cmd == nil || c.cmd.Process == nil {
		return fmt.Errorf("container not started")
	}
	return c.cmd.Process.Signal(sig)
}

// StartContainer starts the container and waits for it to exit.
// It is a Start+Wait wrapper kept for Run() compatibility.
// Non-zero exits are returned as [*ProcessExitError].
func (c *Container) StartContainer(command string, args ...string) error {
	if err := c.Start(command, args...); err != nil {
		return err
	}
	code, err := c.Wait()
	if err != nil {
		return err
	}
	if code != 0 {
		return &ProcessExitError{Code: code}
	}
	return nil
}

func (c *Container) startSignalForwarding() {
	c.signalDone = make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	pid := c.cmd.Process.Pid
	done := c.signalDone
	go func() {
		defer signal.Stop(sigCh)
		for {
			select {
			case sig := <-sigCh:
				if s, ok := sig.(syscall.Signal); ok {
					_ = syscall.Kill(pid, s)
				}
			case <-done:
				return
			}
		}
	}()
}

func (c *Container) stopSignalForwarding() {
	c.signalOnce.Do(func() {
		if c.signalDone != nil {
			close(c.signalDone)
		}
	})
}

func (c *Container) DeleteContainer() error {
	var firstErr error
	if c.cgroupManager != nil {
		if err := c.cgroupManager.Destroy(c.ID); err != nil {
			firstErr = err
		}
	}
	if c.rootfsCleanup != nil {
		if err := c.rootfsCleanup(); err != nil && firstErr == nil {
			firstErr = err
		}
		c.rootfsCleanup = nil
	}
	return firstErr
}

func (c *Container) PID() int {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Pid
	}
	return 0
}
