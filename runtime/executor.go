package runtime

type ProcessExecutor interface {
	Execute(command string, args ...string) error
}
