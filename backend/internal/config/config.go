package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	GTFS        GTFSConfig
	Auth        AuthConfig
	Planner     PlannerConfig
	PlaceSearch PlaceSearchConfig
}

type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  int
	WriteTimeout int
}

type DatabaseConfig struct {
	URL      string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type GTFSConfig struct {
	DataPath string
}

type AuthConfig struct {
	GoogleClientID       string
	SessionTokenSecret   string
	SessionDurationHours int
}

type PlannerConfig struct {
	Adapter string
}

type PlaceSearchConfig struct {
	Provider         string
	GoogleAPIKey     string
	GoogleRegionCode string
}

func Load() *Config {
	loadLocalEnvFiles()

	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 30),
		},
		Database: DatabaseConfig{
			URL:      getEnv("DATABASE_URL", ""),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "traveller"),
			Password: getEnv("DB_PASSWORD", "traveller"),
			DBName:   getEnv("DB_NAME", "traveller"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		GTFS: GTFSConfig{
			DataPath: getEnv("GTFS_DATA_PATH", "../DMRC_GTFS"),
		},
		Auth: AuthConfig{
			GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
			SessionTokenSecret:   getEnv("SESSION_TOKEN_SECRET", "traveller-dev-session-secret"),
			SessionDurationHours: getEnvAsInt("SESSION_DURATION_HOURS", 24*30),
		},
		Planner: PlannerConfig{
			Adapter: strings.ToLower(getEnv("PLANNER_ADAPTER", "in_memory")),
		},
		PlaceSearch: PlaceSearchConfig{
			Provider:         strings.ToLower(getEnv("PLACE_SEARCH_PROVIDER", "stop_local")),
			GoogleAPIKey:     getEnv("GOOGLE_MAPS_API_KEY", ""),
			GoogleRegionCode: strings.ToLower(getEnv("PLACE_SEARCH_GOOGLE_REGION_CODE", "in")),
		},
	}
}

func loadLocalEnvFiles() {
	candidates := []string{
		".env",
		".env.local",
		filepath.Join("backend", ".env"),
		filepath.Join("backend", ".env.local"),
	}

	for _, path := range candidates {
		loadEnvFile(path)
	}
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}

		value = strings.Trim(value, `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
