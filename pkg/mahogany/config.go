package mahogany

import (
	"os"
	"strconv"
	"strings"
	"time"

	db "github.com/mpoegel/mahogany/internal/db"
)

type Config struct {
	DbFile        string
	StaticDir     string
	Port          int
	Timeout       time.Duration
	DockerHost    string
	DockerVersion string
	TopologyFile  string
}

func LoadConfig() Config {
	return Config{
		DbFile:        loadStrEnv("DB_FILE", "mahogany.db"),
		StaticDir:     loadStrEnv("STATIC_DIR", "static"),
		Port:          loadIntEnv("PORT", 9090),
		Timeout:       time.Duration(loadIntEnv("TIMEOUT", 3)) * time.Second,
		DockerHost:    loadStrEnv("DOCKER_HOST", "localhost"),
		DockerVersion: loadStrEnv("DOCKER_VERSION", "3"),
		TopologyFile:  loadStrEnv("TOPOLOGY", "topology.toml"),
	}
}

type AgentConfig struct {
	ServerAddr        string
	HostName          string
	DownloadDir       string
	TelemetryEndpoint string
}

func LoadAgentConfig() AgentConfig {
	cfg := AgentConfig{
		ServerAddr:        loadStrEnv("SERVER_ADDR", "localhost:9091"),
		DownloadDir:       loadStrEnv("DOWNLOAD_DIR", "/tmp"),
		TelemetryEndpoint: loadStrEnv("TELEMETRY_ENDPOINT", "localhost:4317"),
	}
	hostname, err := os.ReadFile("/etc/hostname")
	if err == nil {
		cfg.HostName = strings.Trim(string(hostname), "\n")
	} else {
		cfg.HostName = loadStrEnv("HOSTNAME", "mahogany")
	}
	return cfg
}

type AppData struct {
	Packages        []db.Package `json:"packages"`
	Settings        []db.Setting `json:"settings"`
	WatchedServices []string     `json:"watched_services"`
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
