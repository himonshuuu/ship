package chroot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func MountInChroot() bool {
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fmt.Printf("failed to mount /proc: %v\n", err)
	}
	if err := syscall.Mount("sys", "/sys", "sysfs", 0, ""); err != nil {
		fmt.Printf("failed to mount /sys: %v\n", err)
	}
	mountedDevtmpfs := true
	if err := syscall.Mount("devtmpfs", "/dev", "devtmpfs", 0, ""); err != nil {
		fmt.Printf("failed to mount /dev as devtmpfs (will fallback to device nodes): %v\n", err)
		mountedDevtmpfs = false
	}
	return mountedDevtmpfs
}

func RunChild(rootfs string, args []string) error {
	fmt.Printf("Running inside container! PID: %d\n", syscall.Getpid())

	if err := syscall.Chroot(rootfs); err != nil {
		return fmt.Errorf("Error chrooting: %w", err)
	}
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("Error chdir to /: %w", err)
	}

	mountedDevtmpfs := MountInChroot()
	if !mountedDevtmpfs {
		devices := map[string]struct {
			mode uint32
			dev  int
		}{
			"null":    {syscall.S_IFCHR | 0666, (1 << 8) | 3},
			"zero":    {syscall.S_IFCHR | 0666, (1 << 8) | 5},
			"tty":     {syscall.S_IFCHR | 0666, 5 << 8},
			"random":  {syscall.S_IFCHR | 0666, (1 << 8) | 8},
			"urandom": {syscall.S_IFCHR | 0666, (1 << 8) | 9},
		}
		for name, d := range devices {
			if err := syscall.Mknod(filepath.Join("/dev", name), d.mode, d.dev); err != nil && !os.IsExist(err) {
				fmt.Printf("Failed to mknod %s: %v\n", name, err)
			}
		}
	}

	// set sane PATH inside the chroot
	os.Setenv("PATH", "/usr/bin:/bin:/usr/sbin:/sbin")

	cmd := exec.Command("/bin/bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
