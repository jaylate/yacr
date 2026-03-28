# yacr

Yet Another Container Runtime - a minimal container runtime written in Go

## CLI Usage

```bash
make build
./bin/yacr run /bin/sh
```

**Note:** Download an [Alpine rootfs](https://alpinelinux.org/downloads/) first

### Options

- `--hostname` - Set container hostname (default: `container`)
- `--rootfs` - Set root filesystem path (default: `rootfs`)
- `--memory` - Set memory limit (e.g., `100M`, `1G`, `max`)
- `--cpus` - Set CPU limit (e.g., `0.5`, `2`, `max`)
- `--pids` - Set PID limit (e.g., `50`, `max`)

### Examples

```bash
# Basic usage
./bin/yacr run /bin/sh

# With hostname
./bin/yacr --hostname myhost run /bin/sh -l

# With resource limits
./bin/yacr --memory 512M --cpus 2 --pids 50 run /bin/sh

# All options
./bin/yacr --hostname myhost --rootfs /path/to/rootfs --memory 1G --cpus 4 --pids 100 run /bin/sh
```

## Library Usage

Import the runtime package to use yacr as a library:

```go
import (
    "github.com/jaylate/yacr/runtime"
    "github.com/jaylate/yacr/runtime/resources"  // needed for ResourceLimits
)
```

### Run

Execute a command in an isolated environment:

```go
// Basic usage
runtime.Run("/bin/sh", nil, nil)

// With command arguments
runtime.Run("/bin/sh", []string{"-c", "echo hello"}, nil)

// With configuration
runtime.Run("/bin/sh", nil, &runtime.ContainerConfig{
    ContainerID: "my-container",  // optional, auto-generated if empty
    RootFS:     "/path/to/rootfs",
    Hostname:   "myhost",
    Limits: resources.ResourceLimits{
        MemoryBytes: 512 * 1024 * 1024,  // 512M
        CPUCores:    2.0,
        PIDsMax:     50,
    },
})
```

## Configuration

### ContainerConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| ContainerID | string | Unique container identifier | auto-generated |
| InitBinary | string | Path to init binary | "./bin/init" |
| RootFS | string | Root filesystem path | "rootfs" |
| Hostname | string | Container hostname | "container" |
| Limits | ResourceLimits | Cgroup resource limits | unlimited |

### ResourceLimits

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| MemoryBytes | int64 | Memory limit in bytes | 0 (unlimited) |
| CPUCores | float64 | CPU cores (0.5 = 50%) | 0 (unlimited) |
| PIDsMax | int | Max number of processes | 0 (unlimited) |

## Roadmap

- [ ] Init (Separate C binary vs Go approach)
    - [ ] Move to integrated Go init
    - [ ] Add signal forwarding (SIGINT/SIGTERM) parent -> child
    - [ ] Ensure cleanup hooks run on all exit paths
- [ ] Image handling (OCI)
    - [ ] Add `image` package with reference parsing
    - [ ] Support local OCI archives (`oci-archive:/path/to/image.tar`)
    - [ ] Unpack image layers to per-container rootfs workspace
    - [ ] Set resolved rootfs into runtime config
- [ ] Registry Pull (OCI)
    - [ ] Support registry references (e.g., `docker.io/library/alpine:latest`)
    - [ ] Pull manifest + blobs and cache locally
    - [ ] Extract layers in order and apply config metadata
    - [ ] Add digest/media-type validation
- [ ] Testing
    - [ ] Add comprehensive testing (OCI runtime spec compliance tests, integration tests)
