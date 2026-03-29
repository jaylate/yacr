package runtime

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jaylate/yacr/runtime/resources"
)

type Container struct {
	ID     string
	Config ContainerConfig
	mgr    resources.CgroupManager

	namespaceSetup NamespaceSetup
	processRunner  ProcessRunner
	idGenerator    IDGenerator

	cmd  *exec.Cmd
	args []string
}

func Create(
	cfg ContainerConfig,
	mgr resources.CgroupManager,
	namespaceSetup NamespaceSetup,
	idGenerator IDGenerator,
) (*Container, error) {
	if namespaceSetup == nil {
		namespaceSetup = &DefaultNamespaceSetup{}
	}
	if idGenerator == nil {
		idGenerator = &DefaultIDGenerator{}
	}

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
		containerID, err = idGenerator.Generate()
		if err != nil {
			return nil, fmt.Errorf("failed to generate container ID: %w", err)
		}
	}

	c := &Container{
		ID:             containerID,
		Config:         cfg,
		mgr:            mgr,
		namespaceSetup: namespaceSetup,
		processRunner:  &DefaultProcessRunner{},
		idGenerator:    idGenerator,
	}

	cmd := exec.Command(c.Config.InitBinary)
	cmd.Args = []string{c.Config.InitBinary}

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := c.namespaceSetup.Setup(c.Config, cmd); err != nil {
		if mgr != nil {
			mgr.Destroy(containerID)
		}
		return nil, fmt.Errorf("failed to setup namespace: %w", err)
	}

	c.cmd = cmd

	if mgr != nil {
		if err := mgr.Create(containerID, cfg.Limits); err != nil {
			return nil, fmt.Errorf("failed to create cgroup: %w", err)
		}
	}

	return c, nil
}

func (c *Container) Start(command string, args ...string) error {
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

	if err := c.processRunner.Run(c.cmd); err != nil {
		if c.mgr != nil {
			c.mgr.Destroy(c.ID)
		}
		return fmt.Errorf("failed to start process: %w", err)
	}

	if c.mgr != nil && c.cmd.Process != nil {
		if err := c.mgr.AddProcess(c.ID, c.cmd.Process.Pid); err != nil {
			c.cmd.Process.Kill()
			c.processRunner.Wait(c.cmd)
			if destroyErr := c.mgr.Destroy(c.ID); destroyErr != nil {
				return fmt.Errorf("failed to add process to cgroup: %w (additionally failed to destroy cgroup: %v)", err, destroyErr)
			}
			return fmt.Errorf("failed to add process to cgroup: %w", err)
		}
	}

	if err := c.processRunner.Wait(c.cmd); err != nil {
		var destroyErr error
		if c.mgr != nil {
			destroyErr = c.mgr.Destroy(c.ID)
		}
		if destroyErr != nil {
			return fmt.Errorf("process exited with error: %w (additionally failed to destroy cgroup: %v)", err, destroyErr)
		}
		return fmt.Errorf("process exited with error: %w", err)
	}

	if c.mgr != nil {
		if err := c.mgr.Destroy(c.ID); err != nil {
			return fmt.Errorf("failed to destroy cgroup for container %s: %w", c.ID, err)
		}
	}

	return nil
}

func (c *Container) Delete() error {
	if c.mgr != nil {
		return c.mgr.Destroy(c.ID)
	}
	return nil
}

func (c *Container) PID() int {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Pid
	}
	return 0
}
