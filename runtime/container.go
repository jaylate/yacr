package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	Limits         resources.ResourceLimits
}

func DefaultContainerConfig() *ContainerConfig {
	return &ContainerConfig{
		RootFS:   "rootfs",
		Hostname: "container",
	}
}

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
	rootfsPath    string
	rootfsCleanup func() error
}

func (r *Runtime) CreateContainer(cfg ContainerConfig) (*Container, error) {
	defaults := DefaultContainerConfig()
	if cfg.Hostname == "" {
		cfg.Hostname = defaults.Hostname
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
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

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

func (c *Container) StartContainer(command string, args ...string) error {
	if c.cmd == nil {
		return fmt.Errorf("container not created")
	}
	if len(c.args) > 0 {
		return fmt.Errorf("container already started")
	}

	c.cmd.Args = append(c.cmd.Args, bootstrap.CommandArgs(
		c.Config.Hostname,
		c.rootfsPath,
		command,
		args,
	)...)
	c.args = args

	if err := c.cmd.Start(); err != nil {
		if c.cgroupManager != nil {
			c.cgroupManager.Destroy(c.ID)
		}
		return fmt.Errorf("failed to start process: %w", err)
	}

	if c.cgroupManager != nil && c.cmd.Process != nil {
		if err := c.cgroupManager.AddProcess(c.ID, c.cmd.Process.Pid); err != nil {
			c.cmd.Process.Kill()
			c.cmd.Wait()
			if destroyErr := c.cgroupManager.Destroy(c.ID); destroyErr != nil {
				return fmt.Errorf("failed to add process to cgroup: %w (additionally failed to destroy cgroup: %v)", err, destroyErr)
			}
			return fmt.Errorf("failed to add process to cgroup: %w", err)
		}
	}

	if err := c.cmd.Wait(); err != nil {
		if c.cgroupManager != nil {
			c.cgroupManager.Destroy(c.ID)
		}
		return fmt.Errorf("process exited with error: %w", err)
	}

	return nil
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
