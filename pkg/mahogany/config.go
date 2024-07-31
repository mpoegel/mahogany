package mahogany

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	StaticDir       string
	Port            int
	Timeout         time.Duration
	DockerHost      string
	DockerVersion   string
	WatchtowerAddr  string
	WatchtowerToken string
	RegistryAddr    string
}

func LoadConfig() Config {
	return Config{
		StaticDir:       loadStrEnv("STATIC_DIR", "static"),
		Port:            loadIntEnv("PORT", 9090),
		Timeout:         time.Duration(loadIntEnv("TIMEOUT", 3)) * time.Second,
		DockerHost:      loadStrEnv("DOCKER_HOST", "localhost"),
		DockerVersion:   loadStrEnv("DOCKER_VERSION", "3"),
		WatchtowerAddr:  loadStrEnv("WATCHTOWER_ADDR", "localhost:8080"),
		WatchtowerToken: loadStrEnv("WATCHTOWER_TOKEN", ""),
		RegistryAddr:    loadStrEnv("REGISTRY_ADDR", "localhost:5000"),
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
