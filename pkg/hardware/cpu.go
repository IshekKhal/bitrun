package hardware

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Info holds host hardware specifications relevant for 1-bit inference.
type Info struct {
	OS          string
	Arch        string
	NumCPU      int
	Threads     int
	TotalRAMGB  float64
	HasDotProd  bool
	HasAVX512   bool
	HasAVX2     bool
}

// Detect gathers hardware capabilities of the current host machine.
func Detect() *Info {
	info := &Info{
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
		NumCPU: runtime.NumCPU(),
	}

	// Heuristic: default threads = min(8, max(1, NumCPU - 2)) to prevent UI lockup
	threads := info.NumCPU - 2
	if threads < 1 {
		threads = 1
	}
	if threads > 8 {
		threads = 8
	}
	info.Threads = threads

	if runtime.GOOS == "darwin" {
		detectDarwin(info)
	} else if runtime.GOOS == "linux" {
		detectLinux(info)
	} else if runtime.GOOS == "windows" {
		detectWindows(info)
	}

	return info
}

func runSysctl(key string) string {
	out, err := exec.Command("sysctl", "-n", key).Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	out, err = exec.Command("/usr/sbin/sysctl", "-n", key).Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

func detectDarwin(info *Info) {
	// Query total RAM: sysctl -n hw.memsize
	val := runSysctl("hw.memsize")
	if val != "" {
		bytes, parseErr := strconv.ParseInt(val, 10, 64)
		if parseErr == nil {
			info.TotalRAMGB = float64(bytes) / (1024 * 1024 * 1024)
		}
	}

	if info.Arch == "arm64" {
		// Apple Silicon supports ARM NEON with dot product
		dotProd := runSysctl("hw.optional.arm.FEAT_DotProd")
		if dotProd == "1" {
			info.HasDotProd = true
		}
	}
}

func detectLinux(info *Info) {
	// Linux implementation can read /proc/meminfo and /proc/cpuinfo
	// Default fallbacks:
	info.TotalRAMGB = 8.0
}

func detectWindows(info *Info) {
	// On modern x86_64 Windows (e.g. Intel Core i5-12600K), AVX2 is supported
	if info.Arch == "amd64" {
		info.HasAVX2 = true
	}
	// Query total RAM via PowerShell CIM query
	out, err := exec.Command("powershell", "-NoProfile", "-Command", "(Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory").Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		bytes, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr == nil {
			info.TotalRAMGB = float64(bytes) / (1024 * 1024 * 1024)
			return
		}
	}
	// Fallback heuristic
	info.TotalRAMGB = 16.0
}
