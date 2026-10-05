package config

import (
	"cmp"
	"os"
	"strconv"
)

type ApplicationConfig struct {
	ServerAddress         string
	SqliteDatabasePath    string
	BackgroundJobsEnabled bool
}

// Load reads the process environment once at startup; every setting has a default so an empty .env still boots.
func Load() ApplicationConfig {
	return ApplicationConfig{
		ServerAddress:         cmp.Or(os.Getenv("SERVER_ADDRESS"), ":8080"),
		SqliteDatabasePath:    cmp.Or(os.Getenv("SQLITE_DB_PATH"), "./data/go-coin-hunter.sqlite3"),
		BackgroundJobsEnabled: parseBoolWithDefault(os.Getenv("BACKGROUND_JOBS_ENABLED"), true),
	}
}

func parseBoolWithDefault(rawValue string, defaultValue bool) bool {
	parsedValue, parseError := strconv.ParseBool(rawValue)
	if parseError != nil {
		return defaultValue
	}

	return parsedValue
}
