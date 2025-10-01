package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/himonshuuu/ship/utils"
)

func findLinkedLibs(binary string) []string {
	out, err := exec.Command("ldd", binary).Output()
	if err != nil {
		fmt.Println("Error running ldd:", err)
		return nil
	}

	var libs []string
	for _, line := range utils.SplitLines(string(out)) {
		var libPath string
		n, _ := fmt.Sscanf(line, "\t%*s => %s", &libPath)
		if n == 1 && libPath != "not" {
			libs = append(libs, libPath)
		} else if n == 0 {
			n, _ = fmt.Sscanf(line, "\t%s", &libPath)
			if n == 1 && len(libPath) > 0 && libPath[0] == '/' {
				libs = append(libs, libPath)
			}
		}
	}
	return libs
}

func createRootFs() {
	cwd, err := os.Getwd() // temporary, ill change later
	if err != nil {
		fmt.Println("error getting current working directory:", err)
		return
	}

	tmpDirPath := filepath.Join(cwd, "tmp", "rootfs")

	requiredBinaries := []string{
		"bash",
		"ls",
		"mv",
		"cp",
		"rm",
		"mkdir",
		"rmdir",
		"cat",
		"echo",
		"touch",
	}

	binDst := filepath.Join(tmpDirPath, "bin")
	libDst := filepath.Join(tmpDirPath, "lib")
	lib64Dst := filepath.Join(tmpDirPath, "lib64")
	userDst := filepath.Join(tmpDirPath, "usr")
	procDst := filepath.Join(tmpDirPath, "proc")
	sysDst := filepath.Join(tmpDirPath, "sys")
	devDst := filepath.Join(tmpDirPath, "dev")

	dirs := []string{binDst, libDst, lib64Dst, userDst, procDst, sysDst, devDst}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Printf("failed to create dir %s: %v\n", d, err)
		}
	}

	libsSet := make(map[string]struct{})
	for _, bin := range requiredBinaries {
		binPath, err := exec.LookPath(bin)
		if err != nil {
			fmt.Printf("binary %s not found in PATH\n", bin)
			continue
		}
		fmt.Printf("binary %s found at %s\n", bin, binPath)
		dstPath := filepath.Join(binDst, filepath.Base(binPath))
		if err := utils.CopyFile(binPath, dstPath); err != nil {
			fmt.Printf("failed to copy %s: %v\n", binPath, err)
		}

		for _, lib := range findLinkedLibs(binPath) {
			libsSet[lib] = struct{}{}
		}
	}

	for lib := range libsSet {
		var dst string
		if strings.HasPrefix(lib, "/lib64") {
			dst = filepath.Join(lib64Dst, lib[len("/lib64/"):])
		} else if strings.HasPrefix(lib, "/lib") {
			dst = filepath.Join(libDst, lib[len("/lib/"):])
		} else if strings.HasPrefix(lib, "/usr") {
			dst = filepath.Join(userDst, lib[len("/usr/"):])
		} else {
			continue
		}
		if err := utils.CopyFile(lib, dst); err != nil {
			fmt.Printf("failed to copy lib %s: %v\n", lib, err)
		}
	}

	syscall.Mount("proc", filepath.Join(tmpDirPath, "proc"), "proc", 0, "")
	syscall.Mount("sys", filepath.Join(tmpDirPath, "sys"), "sysfs", 0, "")
	syscall.Mount("dev", filepath.Join(tmpDirPath, "dev"), "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "")

	syscall.Mknod(filepath.Join(tmpDirPath, "dev", "null"), syscall.S_IFCHR|0666, int((1<<8)|3))
	syscall.Mknod(filepath.Join(tmpDirPath, "dev", "zero"), syscall.S_IFCHR|0666, int((1<<8)|5))
	syscall.Mknod(filepath.Join(tmpDirPath, "dev", "tty"), syscall.S_IFCHR|0666, int((5<<8)|0))
	syscall.Mknod(filepath.Join(tmpDirPath, "dev", "random"), syscall.S_IFCHR|0666, int((1<<8)|8))
	syscall.Mknod(filepath.Join(tmpDirPath, "dev", "urandom"), syscall.S_IFCHR|0666, int((1<<8)|9))

	fmt.Println("root fs created at", tmpDirPath)

}

func main() {
	createRootFs()

	if len(os.Args) > 1 && os.Args[1] == "child" {
		fmt.Printf("Running inside the container! PID: %d\n", syscall.Getpid())

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory:", err)
			os.Exit(1)
		}

		rootfs := filepath.Join(cwd, "tmp", "rootfs")
		if err := syscall.Chroot(rootfs); err != nil {
			fmt.Printf("Error changing root to %s: %v\n", rootfs, err)
			os.Exit(1)
		}
		if err := os.Chdir("/"); err != nil {
			fmt.Println("Error changing directory:", err)
			os.Exit(1)
		}

		cmd := exec.Command("/bin/bash")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("Error:", err)
		}
		return
	}
}
