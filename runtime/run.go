package runtime

func Run(command string, commandArgs []string, cfg *ContainerConfig) error {
	executor := NewLinuxExecutor(cfg)
	return RunWithExecutor(command, commandArgs, executor)
}

func RunWithExecutor(command string, commandArgs []string, executor ProcessExecutor) error {
	return executor.Execute(command, commandArgs...)
}
