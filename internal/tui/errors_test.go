package tui

import (
	"strings"
	"testing"
)

func TestFriendlyKubeErrorConnectionRefused(t *testing.T) {
	windows := `Get "https://127.0.0.1:54690/api": dial tcp 127.0.0.1:54690: connectex: No connection could be made because the target machine actively refused it.`
	linux := `Get "https://127.0.0.1:54690/api": dial tcp 127.0.0.1:54690: connect: connection refused`
	for _, output := range []string{windows, linux} {
		msg := friendlyKubeError(output)
		if msg == "" || msg == "exit status 1" {
			t.Fatalf("friendlyKubeError(%q) = %q", output, msg)
		}
	}
}

func TestNormalizeCLIOutputCRLF(t *testing.T) {
	output := normalizeCLIOutput("api\r\nUp 2 hours\r\n")
	if !strings.Contains(output, "api\nUp") {
		t.Fatalf("normalizeCLIOutput() = %q", output)
	}
}

func TestFriendlyDockerErrorWindowsPipe(t *testing.T) {
	output := `error during connect: open //./pipe/dockerDesktopLinuxEngine: O sistema não pode encontrar o arquivo especificado.`
	msg := friendlyDockerError(output)
	if msg == "" || msg == "exit status 1" {
		t.Fatalf("friendlyDockerError() = %q", msg)
	}
}

func TestActionableErrorIncludesNextStep(t *testing.T) {
	msg := actionableError("docker", "Cannot connect to the Docker daemon")
	if !strings.Contains(msg, "Proximo passo:") {
		t.Fatalf("actionableError() = %q", msg)
	}
}

func TestFriendlyAWSErrorInvalidCredentials(t *testing.T) {
	output := "An error occurred (SignatureDoesNotMatch) when calling the GetCallerIdentity operation"
	msg := friendlyAWSError(output)
	if msg == "" || msg == "exit status 254" {
		t.Fatalf("friendlyAWSError() = %q", msg)
	}
}
