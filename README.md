## ship

Minimal, educational container-like environment using chroot and mount namespaces.

This tool constructs a tiny root filesystem with a handful of userland tools and their required shared libraries, then starts a bash shell inside a new mount namespace and chroot. It is intended for learning, not production isolation.

### How it works (high level)

-   Builds a rootfs under `tmp/rootfs` with basic directories and copies selected binaries (e.g., `bash`, `ls`, `cp`, etc.).
-   Resolves linked libraries via `ldd` and copies them, plus the ELF interpreter, into the rootfs.
-   Launches a child process in a new mount namespace, `chroot`s into the rootfs, mounts `proc`, `sysfs`, and attempts `devtmpfs` (or creates a few device nodes if that fails).
-   Spawns `/bin/bash` inside the chroot with a minimal `PATH`.

### Requirements

-   Linux host with `sudo` available
-   Go toolchain (as specified in `go.mod`)
-   `ldd` on PATH (usually provided by glibc)

### Build

```bash
make build
```

This produces the `ship` binary at the repo root.

### Run

Running requires root privileges for namespaces, mounts, device nodes, and chroot:

```bash
sudo ./ship child
```

Alternatively, use the Makefile target:

```bash
make run
```

You should land in a bash shell inside the new rootfs. Type `exit` to leave.

### Cleanup

Unmount mounted filesystems and remove the temporary rootfs:

```bash
make cleantmp
```

Remove the built binary:

```bash
make clean
```

### Makefile targets

-   **build**: `go build -o ship cmd/main.go`
-   **run**: `sudo ./ship child`
-   **cleantmp**: Unmounts `proc`, `sys`, `dev` if mounted under `tmp/rootfs` and removes the directory
-   **clean**: Removes the `ship` binary

### Project structure

```
cmd/
  main.go            # entrypoint; wires rootfs creation and child exec
internal/
  rootfs/
    rootfs.go        # builds the minimal root filesystem
  chroot/
    chroot.go        # chroot, mounts, and child shell exec
utils/
  utils.go           # small file/dir helpers
Makefile             # build/run/cleanup helpers
```

### Notes and caveats

-   This is not a container runtime and provides limited isolation. It uses only mount namespaces and chroot.
-   `sudo` is required; ensure you trust the code you run as root.
-   On some distributions, `devtmpfs` may be restricted; the tool falls back to creating a few device nodes.
-   TERM and terminfo: if `TERM` is set, a matching terminfo entry is copied to improve shell UX.

### Troubleshooting

-   Shell fails to start or is blank: ensure `bash` exists on host and `ldd` is available.
-   Permission errors: re-run with `sudo`.
-   Garbled terminal: try `export TERM=xterm-256color` on host before running; or verify terminfo was copied.
-   Cleanup issues: if `make cleantmp` reports busy mounts, ensure no process is still running inside the chroot.
