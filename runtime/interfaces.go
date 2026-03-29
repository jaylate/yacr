package runtime

import "os/exec"

type NamespaceSetup interface {
	Setup(config ContainerConfig, cmd *exec.Cmd) error
}

type ProcessRunner interface {
	Run(cmd *exec.Cmd) error
	Wait(cmd *exec.Cmd) error
}

type IDGenerator interface {
	Generate() (string, error)
}
