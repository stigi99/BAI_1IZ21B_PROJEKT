package config

import "os"

// AppConfig groups the runtime settings loaded from environment variables.
type AppConfig struct {
	DBPath          string
	Port            string
	SecurityEnabled bool
}

// Load reads the application configuration from the environment.
//
// Output:
//   - AppConfig: resolved database path, listen address, and security mode.
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
