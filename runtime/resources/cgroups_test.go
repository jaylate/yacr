package resources

import (
	"fmt"
	"os"
	"testing"
)

func TestParseMemoryString(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		ok    bool
	}{
		{"100M", 100 * 1024 * 1024, true},
		{"1G", 1024 * 1024 * 1024, true},
		{"1024", 1024, true},
		{"max", 0, true},
		{"invalid", 0, false},
	}
	for _, tt := range tests {
		got, ok := ParseMemoryString(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseMemoryString(%q) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if ok && got != tt.want {
			t.Errorf("ParseMemoryString(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseCPUString(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"1.0", "100000 100000", true},
		{"0.5", "50000 100000", true},
		{"2.0", "200000 100000", true},
		{"max", "max 100000", true},
		{"invalid", "", false},
	}
	for _, tt := range tests {
		got, ok := ParseCPUString(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseCPUString(%q) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if ok && got != tt.want {
			t.Errorf("ParseCPUString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParsePIDsString(t *testing.T) {
	tests := []struct {
		input string
		want  int
		ok    bool
	}{
		{"50", 50, true},
		{"100", 100, true},
		{"max", 0, true},
		{"invalid", 0, false},
	}
	for _, tt := range tests {
		got, ok := ParsePIDsString(tt.input)
		if ok != tt.ok {
			t.Errorf("ParsePIDsString(%q) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if ok && got != tt.want {
			t.Errorf("ParsePIDsString(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestDetectCgroupVersion(t *testing.T) {
	version := DetectCgroupVersion()
	if version != "v1" && version != "v2" {
		t.Errorf("DetectCgroupVersion() = %q, want v1 or v2", version)
	}
}

func TestDetectUserCgroupPath(t *testing.T) {
	uid := os.Getuid()
	expected := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/user.slice/", uid, uid)

	path := DetectUserCgroupPath()
	if path != expected {
		t.Errorf("DetectUserCgroupPath() = %q, want %q", path, expected)
	}
}

func TestIsCgroupWritable(t *testing.T) {
	path := DetectUserCgroupPath()
	if path == "" {
		t.Skip("DetectUserCgroupPath returned empty, skipping test")
	}

	writable := IsCgroupWritable(path)
	if !writable {
		t.Errorf("IsCgroupWritable(%q) = false, want true (path should be writable for current user)", path)
	}
}

func TestCgroupsManager_Create(t *testing.T) {
	dir := t.TempDir()
	limits := ResourceLimits{
		MemoryBytes: 100 * 1024 * 1024,
		CPUCores:    1.0,
		PIDsMax:     50,
	}
	mgr := NewCgroupsManager(dir, limits)
	err := mgr.Create("test-container", limits)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Check directory exists (manager adds "yacr-" prefix)
	_, err = os.Stat(dir + "/yacr-test-container")
	if os.IsNotExist(err) {
		t.Fatalf("directory not created: %s", dir+"/yacr-test-container")
	}

	// Check memory.max
	data, err := os.ReadFile(dir + "/yacr-test-container/memory.max")
	if err != nil {
		t.Fatalf("failed to read memory.max: %v", err)
	}
	if string(data) != "104857600\n" {
		t.Errorf("memory.max = %q, want %q", string(data), "104857600\n")
	}

	// Check cpu.max
	data, err = os.ReadFile(dir + "/yacr-test-container/cpu.max")
	if err != nil {
		t.Fatalf("failed to read cpu.max: %v", err)
	}
	if string(data) != "100000 100000\n" {
		t.Errorf("cpu.max = %q, want %q", string(data), "100000 100000\n")
	}

	// Check pids.max
	data, err = os.ReadFile(dir + "/yacr-test-container/pids.max")
	if err != nil {
		t.Fatalf("failed to read pids.max: %v", err)
	}
	if string(data) != "50\n" {
		t.Errorf("pids.max = %q, want %q", string(data), "50\n")
	}
}

func TestCgroupsManager_AddProcess(t *testing.T) {
	dir := t.TempDir()
	limits := ResourceLimits{
		MemoryBytes: 100 * 1024 * 1024,
		CPUCores:    1.0,
		PIDsMax:     50,
	}
	mgr := NewCgroupsManager(dir, limits)
	err := mgr.Create("test-container", limits)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	err = mgr.AddProcess(25)
	if err != nil {
		t.Fatalf("AddProcess() failed: %v", err)
	}

	data, err := os.ReadFile(dir + "/yacr-test-container/cgroup.procs")
	if err != nil {
		t.Fatalf("failed to read cgroup.procs: %v", err)
	}

	if string(data) != "25\n" {
		t.Errorf("cgroup.procs = %q, want %q", string(data), "25\n")
	}
}
func TestCgroupsManager_Destroy(t *testing.T) {
	dir := t.TempDir()
	limits := ResourceLimits{}
	mgr := NewCgroupsManager(dir, limits)

	err := mgr.Create("test-container", limits)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify directory exists
	_, err = os.Stat(dir + "/yacr-test-container")
	if os.IsNotExist(err) {
		t.Fatal("directory should exist before Destroy")
	}

	// Destroy
	err = mgr.Destroy("test-container")
	if err != nil {
		t.Fatalf("Destroy() failed: %v", err)
	}

	// Verify directory removed
	_, err = os.Stat(dir + "/yacr-test-container")
	if !os.IsNotExist(err) {
		t.Fatalf("directory should be removed after Destroy")
	}
}

func TestCgroupsManager_UnlimitedValues(t *testing.T) {
	dir := t.TempDir()
	limits := ResourceLimits{
		MemoryBytes: 0, // unlimited
		CPUCores:    0, // unlimited
		PIDsMax:     0, // unlimited
	}
	mgr := NewCgroupsManager(dir, limits)

	err := mgr.Create("test-container", limits)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Check memory.max writes "max" when MemoryBytes is 0
	data, err := os.ReadFile(dir + "/yacr-test-container/memory.max")
	if err != nil {
		t.Fatalf("failed to read memory.max: %v", err)
	}
	if string(data) != "max\n" {
		t.Errorf("memory.max = %q, want %q", string(data), "max\n")
	}

	// Check cpu.max writes "max" when CPUCores is 0
	data, err = os.ReadFile(dir + "/yacr-test-container/cpu.max")
	if err != nil {
		t.Fatalf("failed to read cpu.max: %v", err)
	}
	if string(data) != "max 100000\n" {
		t.Errorf("cpu.max = %q, want %q", string(data), "max 100000\n")
	}

	// Check pids.max writes "max" when PIDsMax is 0
	data, err = os.ReadFile(dir + "/yacr-test-container/pids.max")
	if err != nil {
		t.Fatalf("failed to read pids.max: %v", err)
	}
	if string(data) != "max\n" {
		t.Errorf("pids.max = %q, want %q", string(data), "max\n")
	}
}
