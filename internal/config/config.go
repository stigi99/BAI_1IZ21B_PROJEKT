package config

import "os"

type AppConfig struct {
	DBPath          string
	Port            string
	SecurityEnabled bool
}

func Load() AppConfig {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "app.db"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}

	securityEnabled := os.Getenv("SECURITY_ENABLED") == "true"

	return AppConfig{
		DBPath:          dbPath,
		Port:            port,
		SecurityEnabled: securityEnabled,
	}
}
