package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/jaylate/yacr/runtime/resources"
)

type ContainerConfig struct {
	ContainerID string
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

	cmd  *exec.Cmd
	args []string
}

func (r *Runtime) CreateContainer(cfg ContainerConfig) (*Container, error) {
	defaults := DefaultContainerConfig()
	if cfg.InitBinary == "" {
		cfg.InitBinary = defaults.InitBinary
	}
	if cfg.RootFS == "" {
		cfg.RootFS = defaults.RootFS
	}
	if cfg.Hostname == "" {
		cfg.Hostname = defaults.Hostname
	}

	containerID := cfg.ContainerID
	if containerID == "" {
		var err error
		containerID, err = generateContainerID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate container ID: %w", err)
		}
	}

	c := &Container{
		ID:               containerID,
		Config:           cfg,
		cgroupManager:    r.cgroupManager,
		namespaceManager: r.namespaceManager,
	}

	cmd := exec.Command(c.Config.InitBinary)
	cmd.Args = []string{c.Config.InitBinary}

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := c.namespaceManager.Create(c.Config, cmd); err != nil {
		if r.cgroupManager != nil {
			r.cgroupManager.Destroy(containerID)
		}
		return nil, fmt.Errorf("failed to setup namespace: %w", err)
	}

	c.cmd = cmd

	if r.cgroupManager != nil {
		if err := r.cgroupManager.Create(containerID, cfg.Limits); err != nil {
			return nil, fmt.Errorf("failed to create cgroup: %w", err)
		}
	}

	return c, nil
}

func (c *Container) StartContainer(command string, args ...string) error {
	if c.cmd == nil {
		return fmt.Errorf("container not created")
	}
	if len(c.args) > 0 {
		return fmt.Errorf("container already started")
	}

	c.cmd.Args = append(c.cmd.Args,
		"--hostname", c.Config.Hostname,
		"--rootfs", c.Config.RootFS,
		"--",
		command,
	)
	c.cmd.Args = append(c.cmd.Args, args...)
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
	if c.cgroupManager != nil {
		return c.cgroupManager.Destroy(c.ID)
	}
	return nil
}

func (c *Container) PID() int {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Pid
	}
	return 0
}
