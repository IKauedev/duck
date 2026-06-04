package config

import "os"

type Config struct {
	DockerBin        string
	DockerComposeBin string
	KubectlBin       string
	AWSBin           string
	GoBin            string
	WSLBin           string
}

func Load() Config {
	return Config{
		DockerBin:        envOrDefault("DUCK_DOCKER_BIN", "docker"),
		DockerComposeBin: envOrDefault("DUCK_DOCKER_COMPOSE_BIN", "docker-compose"),
		KubectlBin:       envOrDefault("DUCK_KUBECTL_BIN", "kubectl"),
		AWSBin:           envOrDefault("DUCK_AWS_BIN", "aws"),
		GoBin:            envOrDefault("DUCK_GO_BIN", "go"),
		WSLBin:           envOrDefault("DUCK_WSL_BIN", "wsl"),
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
