package runtime

import (
	"errors"
	"testing"
)

type fakeExecutor struct {
	gotCommand string
	gotArgs    []string
	err        error
}

func (f *fakeExecutor) Execute(command string, args ...string) error {
	f.gotCommand = command
	f.gotArgs = append([]string(nil), args...)
	return f.err
}

func TestRunWithExecutor_PassesCommandAndArgs(t *testing.T) {
	fake := &fakeExecutor{}
	err := RunWithExecutor("/bin/sh", []string{"-l", "-a"}, fake)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fake.gotCommand != "/bin/sh" {
		t.Fatalf("command = %q, want %q", fake.gotCommand, "/bin/sh")
	}
	if len(fake.gotArgs) != 2 || fake.gotArgs[0] != "-l" || fake.gotArgs[1] != "-a" {
		t.Fatalf("args = %#v, want %#v", fake.gotArgs, []string{"-l", "-a"})
	}
}

func TestRunWithExecutor_PropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeExecutor{err: wantErr}
	err := RunWithExecutor("/bin/sh", nil, fake)

	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}
