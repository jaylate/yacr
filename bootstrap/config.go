package bootstrap

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	// ConfigFDEnv names the environment variable holding the config pipe fd.
	ConfigFDEnv = "_YACR_CONFIGFD"
)

// Config is the bootstrap configuration passed from the parent over a pipe.
// Env is KEY=VAL; a nil or empty slice means an empty environment (current behavior).
type Config struct {
	Hostname string   `json:"hostname"`
	RootFS   string   `json:"rootfs"`
	WorkDir  string   `json:"cwd"` // default "/"
	Env      []string `json:"env"` // KEY=VAL; empty means empty env
	UID      uint32   `json:"uid"` // container uid after setup (default 0)
	GID      uint32   `json:"gid"`
	Command  string   `json:"command"`
	Args     []string `json:"args"`
}

// WriteConfig encodes cfg as a single JSON line to w.
func WriteConfig(w io.Writer, cfg Config) error {
	enc := json.NewEncoder(w)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encode bootstrap config: %w", err)
	}
	return nil
}

// ReadConfig decodes a single JSON Config from r.
func ReadConfig(r io.Reader) (Config, error) {
	var cfg Config
	dec := json.NewDecoder(r)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode bootstrap config: %w", err)
	}
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.WorkDir == "" {
		c.WorkDir = "/"
	}
}

func (c Config) validate() error {
	if c.RootFS == "" {
		return fmt.Errorf("bootstrap config: rootfs is required")
	}
	if c.Command == "" {
		return fmt.Errorf("bootstrap config: command is required")
	}
	return nil
}
