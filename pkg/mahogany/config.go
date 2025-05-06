package mahogany

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	StaticDir         string
	Port              int
	Timeout           time.Duration
	DockerHost        string
	DockerVersion     string
	WatchtowerAddr    string
	WatchtowerToken   string
	WatchtowerTimeout time.Duration
	RegistryAddr      string
	RegistryTimeout   time.Duration
	TopologyFile      string
}

func LoadConfig() Config {
	return Config{
		StaticDir:         loadStrEnv("STATIC_DIR", "static"),
		Port:              loadIntEnv("PORT", 9090),
		Timeout:           time.Duration(loadIntEnv("TIMEOUT", 3)) * time.Second,
		DockerHost:        loadStrEnv("DOCKER_HOST", "localhost"),
		DockerVersion:     loadStrEnv("DOCKER_VERSION", "3"),
		WatchtowerAddr:    loadStrEnv("WATCHTOWER_ADDR", "localhost:8080"),
		WatchtowerToken:   loadStrEnv("WATCHTOWER_TOKEN", ""),
		WatchtowerTimeout: time.Duration(loadIntEnv("WATCHTOWER_TIMEOUT", 2)) * time.Second,
		RegistryAddr:      loadStrEnv("REGISTRY_ADDR", "localhost:5000"),
		RegistryTimeout:   time.Duration(loadIntEnv("REGISTRY_TIMEOUT", 2)) * time.Second,
		TopologyFile:      loadStrEnv("TOPOLOGY", "topology.toml"),
	}
}

type AgentConfig struct {
	ServerAddr  string
	HostName    string
	DownloadDir string
}

func LoadAgentConfig() AgentConfig {
	return AgentConfig{
		ServerAddr:  loadStrEnv("SERVER_ADDR", "localhost:9091"),
		HostName:    loadStrEnv("HOSTNAME", "mahogany"),
		DownloadDir: loadStrEnv("DOWNLOAD_DIR", "/tmp"),
	}
}

func loadStrEnv(name, defaultVal string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	return val
}

func loadIntEnv(name string, defaultVal int) int {
	valStr, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
