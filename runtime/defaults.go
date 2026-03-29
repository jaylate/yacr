package runtime

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/jaylate/yacr/runtime/resources"
)

type DefaultNamespaceSetup struct{}

func (s *DefaultNamespaceSetup) Setup(cfg ContainerConfig, cmd *exec.Cmd) error {
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
	return nil
}

type DefaultProcessRunner struct{}

func (r *DefaultProcessRunner) Run(cmd *exec.Cmd) error {
	return cmd.Start()
}

func (r *DefaultProcessRunner) Wait(cmd *exec.Cmd) error {
	return cmd.Wait()
}

type DefaultIDGenerator struct{}

func (g *DefaultIDGenerator) Generate() (string, error) {
	return resources.GenerateContainerID()
}
