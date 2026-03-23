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

### Examples

```bash
./bin/yacr run /bin/sh
./bin/yacr --hostname myhost run /bin/sh -l
./bin/yacr --rootfs /path/to/rootfs run /bin/sh -c "echo hello"
```

## Library Usage

Import the runtime package to use yacr as a library:

```go
import "github.com/jaylate/yacr/runtime"
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
    RootFS:   "/path/to/rootfs",
    Hostname: "myhost",
})
```

## Configuration

### ContainerConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| InitBinary | string | Path to init binary | "./bin/init" |
| RootFS | string | Root filesystem path | "rootfs" |
| Hostname | string | Container hostname | "container" |

## Roadmap

- [ ] Resource Limits (Cgroups v2)
    - [ ] Add `runtime/resources` package with `CgroupManager` interface
    - [ ] Create cgroup directory for container ID and apply limits (memory, CPU, pids)
    - [ ] Attach child PID to cgroup after start
    - [ ] Cleanup cgroup directory on process exit
    - [ ] Add CLI flags: `--memory`, `--cpus`, `--pids`
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
