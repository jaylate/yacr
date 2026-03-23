package cmd

import (
	"io"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantCfg   *RunConfig
		wantError bool
	}{
		{
			name: "valid run with command",
			args: []string{"yacr", "run", "/bin/sh"},
			wantCfg: &RunConfig{
				Command:  "/bin/sh",
				Args:     []string{},
				RootFS:   "rootfs",
				Hostname: "container",
				Help:     false,
			},
			wantError: false,
		},
		{
			name: "valid run with command and args",
			args: []string{"yacr", "run", "/bin/sh", "-l", "-a"},
			wantCfg: &RunConfig{
				Command:  "/bin/sh",
				Args:     []string{"-l", "-a"},
				RootFS:   "rootfs",
				Hostname: "container",
				Help:     false,
			},
			wantError: false,
		},
		{
			name:      "missing subcommand",
			args:      []string{"yacr"},
			wantCfg:   nil,
			wantError: true,
		},
		{
			name:      "missing command after run",
			args:      []string{"yacr", "run"},
			wantCfg:   nil,
			wantError: true,
		},
		{
			name:      "unknown subcommand",
			args:      []string{"yacr", "invalid"},
			wantCfg:   nil,
			wantError: true,
		},
		{
			name: "help flag shows help",
			args: []string{"yacr", "--help"},
			wantCfg: &RunConfig{
				Command:  "",
				Args:     []string{},
				RootFS:   "rootfs",
				Hostname: "container",
				Help:     true,
			},
			wantError: false,
		},
		{
			name: "short help flag shows help",
			args: []string{"yacr", "-h"},
			wantCfg: &RunConfig{
				Command:  "",
				Args:     []string{},
				RootFS:   "rootfs",
				Hostname: "container",
				Help:     true,
			},
			wantError: false,
		},
		{
			name: "hostname flag",
			args: []string{"yacr", "--hostname", "myhost", "run", "/bin/sh"},
			wantCfg: &RunConfig{
				Command:  "/bin/sh",
				Args:     []string{},
				RootFS:   "rootfs",
				Hostname: "myhost",
				Help:     false,
			},
			wantError: false,
		},
		{
			name: "rootfs flag",
			args: []string{"yacr", "--rootfs", "/path/to/rootfs", "run", "/bin/sh"},
			wantCfg: &RunConfig{
				Command:  "/bin/sh",
				Args:     []string{},
				RootFS:   "/path/to/rootfs",
				Hostname: "container",
				Help:     false,
			},
			wantError: false,
		},
		{
			name: "multiple flags before command",
			args: []string{"yacr", "--hostname", "myhost", "--rootfs", "/tmp/root", "run", "/bin/sh", "-l"},
			wantCfg: &RunConfig{
				Command:  "/bin/sh",
				Args:     []string{"-l"},
				RootFS:   "/tmp/root",
				Hostname: "myhost",
				Help:     false,
			},
			wantError: false,
		},
		{
			name: "terminator before run",
			args: []string{"yacr", "--", "run", "/bin/sh", "-l"},
			wantCfg: &RunConfig{
				Command:  "/bin/sh",
				Args:     []string{"-l"},
				RootFS:   "rootfs",
				Hostname: "container",
				Help:     false,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseArgsWithWriter(tt.args, io.Discard)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg.Command != tt.wantCfg.Command {
				t.Errorf("Command = %q, want %q", cfg.Command, tt.wantCfg.Command)
			}
			if cfg.Hostname != tt.wantCfg.Hostname {
				t.Errorf("Hostname = %q, want %q", cfg.Hostname, tt.wantCfg.Hostname)
			}
			if cfg.RootFS != tt.wantCfg.RootFS {
				t.Errorf("RootFS = %q, want %q", cfg.RootFS, tt.wantCfg.RootFS)
			}
			if cfg.Help != tt.wantCfg.Help {
				t.Errorf("Help = %v, want %v", cfg.Help, tt.wantCfg.Help)
			}
			if len(cfg.Args) != len(tt.wantCfg.Args) {
				t.Errorf("Args length = %d, want %d", len(cfg.Args), len(tt.wantCfg.Args))
			}
			for i := range cfg.Args {
				if cfg.Args[i] != tt.wantCfg.Args[i] {
					t.Errorf("Args[%d] = %q, want %q", i, cfg.Args[i], tt.wantCfg.Args[i])
				}
			}
		})
	}
}

func TestUsageIncludesRunSubcommand(t *testing.T) {
	var b strings.Builder
	Usage(&b)

	output := b.String()
	if !strings.Contains(output, "Usage: yacr [options] run <command> [arguments]") {
		t.Fatalf("usage output missing run subcommand: %q", output)
	}
}

func TestParseArgs_MissingFlagValues(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{
			name:        "missing hostname value",
			args:        []string{"yacr", "--hostname"},
			wantMessage: "Missing value for --hostname",
		},
		{
			name:        "missing rootfs value",
			args:        []string{"yacr", "--rootfs"},
			wantMessage: "Missing value for --rootfs",
		},
		{
			name:        "rootfs consumes run without value",
			args:        []string{"yacr", "--rootfs", "run", "/bin/sh"},
			wantMessage: "Missing value for --rootfs",
		},
		{
			name:        "hostname followed by rootfs flag",
			args:        []string{"yacr", "--hostname", "--rootfs", "/tmp/root", "run", "/bin/sh"},
			wantMessage: "Missing value for --hostname",
		},
		{
			name:        "rootfs followed by terminator",
			args:        []string{"yacr", "--rootfs", "--", "run", "/bin/sh"},
			wantMessage: "Missing value for --rootfs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseArgsWithWriter(tt.args, io.Discard)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantMessage)
			}
		})
	}
}
