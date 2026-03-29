package runtime

import (
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

type LinuxExecutor struct {
	cfg ContainerConfig
	mgr resources.CgroupManager
}

func NewLinuxExecutor(cfg *ContainerConfig) *LinuxExecutor {
	defaults := DefaultContainerConfig()
	if cfg == nil {
		return &LinuxExecutor{cfg: *defaults}
	}
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
	container, err := Create(e.cfg, e.mgr, nil, nil)
	if err != nil {
		return err
	}
	return container.Start(command, args...)
}
