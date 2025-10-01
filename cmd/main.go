package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	chrootpkg "github.com/himonshuuu/ship/internal/chroot"
	rootfspkg "github.com/himonshuuu/ship/internal/rootfs"
)

func main() {
	var rootfs string
	if len(os.Args) > 2 && os.Args[1] == "child" {
		rootfs = os.Args[2]
	} else {
		rootfs = rootfspkg.CreateRootFs()
		if rootfs == "" {
			os.Exit(1)
		}
	}

	if len(os.Args) > 1 && os.Args[1] == "child" {
		if err := chrootpkg.RunChild(rootfs, os.Args[1:]); err != nil {
			fmt.Println("Child process error:", err)
		}
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cmd := exec.Command("/proc/self/exe", append([]string{"child", rootfs}, os.Args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS,
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting command: %v\n", err)
		os.Exit(1)
	}

	go func() {
		<-sigChan
		cmd.Process.Signal(syscall.SIGTERM)
	}()

	if err := cmd.Wait(); err != nil {
		fmt.Println("Child process error:", err)
	}
}
