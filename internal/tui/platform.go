package tui

import (
	"os"
	"runtime"
	"strings"
	"unicode/utf16"

	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
)

func normalizeCLIOutput(output string) string {
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")
	return decodeUTF16LE(output)
}

func decodeUTF16LE(output string) string {
	bytes := []byte(output)
	if len(bytes) < 2 || !looksUTF16LE(bytes) {
		return output
	}

	u16 := make([]uint16, 0, len(bytes)/2)
	for i := 0; i+1 < len(bytes); i += 2 {
		value := uint16(bytes[i]) | uint16(bytes[i+1])<<8
		if value == 0xfeff {
			continue
		}
		u16 = append(u16, value)
	}
	return string(utf16.Decode(u16))
}

func looksUTF16LE(bytes []byte) bool {
	limit := len(bytes)
	if limit > 80 {
		limit = 80
	}
	zeros := 0
	for i := 1; i < limit; i += 2 {
		if bytes[i] == 0 {
			zeros++
		}
	}
	return zeros >= limit/4
}

func platformLabel() string {
	if runtime.GOOS == "linux" && isWSLEnv() {
		return "WSL"
	}
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	default:
		return runtime.GOOS
	}
}

func isWSLEnv() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}

func shouldFallbackToWSL(output string, err error) bool {
	if runtime.GOOS != "windows" || err == nil {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(output) + " " + err.Error())
	markers := []string{
		"not recognized",
		"cannot find the file",
		"no such file",
		"enoent",
		"pipe/docker",
		"dockerdesktop",
		"docker.sock",
		"error during connect",
		"cannot connect to the docker daemon",
		"is the docker daemon running",
		"executable file not found",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

type toolBackend struct {
	cfg    config.Config
	run    runner.Runner
	viaWSL bool
}

func (b *toolBackend) output(binary string, args []string) (string, error) {
	if b.viaWSL {
		return b.outputWSL(binary, args)
	}

	output, err := b.run.Output(binary, args)
	output = normalizeCLIOutput(output)
	if shouldFallbackToWSL(output, err) {
		wslOutput, wslErr := b.outputWSL(binary, args)
		if wslErr == nil {
			b.viaWSL = true
			return wslOutput, nil
		}
	}
	return output, err
}

func (b *toolBackend) runCommand(binary string, args []string, opts runner.Options) error {
	if b.viaWSL {
		return b.run.Run(b.cfg.WSLBin, wslCommand(binary, args), opts)
	}
	err := b.run.Run(binary, args, opts)
	if shouldFallbackToWSL("", err) {
		if wslErr := b.run.Run(b.cfg.WSLBin, wslCommand(binary, args), opts); wslErr == nil {
			b.viaWSL = true
			return nil
		}
	}
	return err
}

func (b *toolBackend) resolvedCommand(binary string, args []string) (string, []string) {
	if b.viaWSL {
		return b.cfg.WSLBin, wslCommand(binary, args)
	}
	return binary, args
}

func (b *toolBackend) outputWSL(binary string, args []string) (string, error) {
	output, err := b.run.Output(b.cfg.WSLBin, wslCommand(binary, args))
	return normalizeCLIOutput(output), err
}

func wslCommand(binary string, args []string) []string {
	return append([]string{"-e", binary}, args...)
}

func splitLines(output string) []string {
	output = normalizeCLIOutput(output)
	if strings.TrimSpace(output) == "" {
		return nil
	}
	return strings.Split(output, "\n")
}
