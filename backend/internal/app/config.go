package app

import "os"

type Config struct {
	Port    string
	DataDir string
}

func LoadConfig() Config {
	port := os.Getenv("SMEDITOR_PORT")
	if port == "" {
		port = "8080"
	}
	dataDir := os.Getenv("SMEDITOR_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	return Config{Port: port, DataDir: dataDir}
}
