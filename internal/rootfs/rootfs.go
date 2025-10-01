package rootfs

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/himonshuuu/ship/utils"
)

func findLinkedLibs(binary string) []string {
	out, err := exec.Command("ldd", binary).Output()
	if err != nil {
		fmt.Println("Error running ldd:", err)
		return nil
	}

	var libs []string
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if strings.Contains(line, "=>") {
			parts := strings.SplitN(line, "=>", 2)
			rhs := strings.TrimSpace(parts[1])
			if strings.HasPrefix(rhs, "not found") {
				fmt.Println("ldd missing:", line)
				continue
			}
			for i, c := range rhs {
				if c == ' ' || c == '(' {
					rhs = rhs[:i]
					break
				}
			}
			if strings.HasPrefix(rhs, "/") {
				libs = append(libs, rhs)
			}
			continue
		}

		if strings.HasPrefix(line, "/") {
			for i, c := range line {
				if c == ' ' || c == '(' {
					line = line[:i]
					break
				}
			}
			libs = append(libs, line)
			continue
		}
	}
	return libs
}

func getELFInterpreter(path string) string {
	f, err := elf.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			data := make([]byte, p.Filesz)
			if _, err := p.ReadAt(data, 0); err != nil {
				return ""
			}
			interp := strings.TrimRight(string(data), "\x00\n\r ")
			return interp
		}
	}
	return ""
}

func CreateRootFs() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("error getting current directory:", err)
		return ""
	}

	rootfs := filepath.Join(cwd, "tmp", "rootfs")

	dirs := []string{
		filepath.Join(rootfs, "bin"),
		filepath.Join(rootfs, "usr", "bin"),
		filepath.Join(rootfs, "lib"),
		filepath.Join(rootfs, "lib64"),
		filepath.Join(rootfs, "usr"),
		filepath.Join(rootfs, "proc"),
		filepath.Join(rootfs, "sys"),
		filepath.Join(rootfs, "dev"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Printf("failed to create dir %s: %v\n", d, err)
		}
	}

	requiredBinaries := []string{"bash", "cd", "ls", "cat", "echo", "touch", "mv", "cp", "rm", "mkdir", "rmdir", "clear"}

	libsSet := make(map[string]struct{})
	addPath := func(p string) {
		if p == "" {
			return
		}
		libsSet[p] = struct{}{}
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			libsSet[resolved] = struct{}{}
		}
	}

	for _, bin := range requiredBinaries {
		binPath, err := exec.LookPath(bin)
		if err != nil {
			fmt.Printf("binary %s not found in PATH\n", bin)
			continue
		}

		dstBin := filepath.Join(rootfs, "usr", "bin", filepath.Base(binPath))
		fmt.Printf("Copying binary: %s -> %s\n", binPath, dstBin)
		if err := utils.CopyFile(binPath, dstBin); err != nil {
			fmt.Printf("failed to copy %s: %v\n", binPath, err)
		}

		for _, lib := range findLinkedLibs(binPath) {
			addPath(lib)
		}

		addPath(getELFInterpreter(binPath))
	}

	for lib := range libsSet {
		var dst string
		switch {
		case strings.HasPrefix(lib, "/lib64"):
			dst = filepath.Join(rootfs, "lib64", lib[len("/lib64/"):])
		case strings.HasPrefix(lib, "/lib"):
			dst = filepath.Join(rootfs, "lib", lib[len("/lib/"):])
		case strings.HasPrefix(lib, "/usr"):
			dst = filepath.Join(rootfs, "usr", lib[len("/usr/"):])
		default:
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			fmt.Printf("failed to mkdir for lib %s: %v\n", dst, err)
			continue
		}
		if err := utils.CopyFile(lib, dst); err != nil {
			fmt.Printf("failed to copy lib %s -> %s: %v\n", lib, dst, err)
		}
	}

	binLink := filepath.Join(rootfs, "bin")
	_ = os.Remove(binLink)
	if err := os.Symlink("/usr/bin", binLink); err != nil {
		fmt.Printf("warning: failed to symlink /bin -> /usr/bin: %v\n", err)
	}

	if term := os.Getenv("TERM"); term != "" {
		if len(term) > 0 {
			srcTi := filepath.Join("/usr/share/terminfo", string(term[0]), term)
			dstTi := filepath.Join(rootfs, "usr", "share", "terminfo", string(term[0]), term)
			if _, err := os.Stat(srcTi); err == nil {
				if err := utils.CopyFile(srcTi, dstTi); err != nil {
					fmt.Printf("warning: failed to copy terminfo %s: %v\n", term, err)
				}
			}
		}
	}

	fmt.Println("root filesystem created at:", rootfs)
	return rootfs
}
