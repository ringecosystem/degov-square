package internal

import (
	"log/slog"
	"os"
	"strings"
)

type Environment int

const (
	Production Environment = iota
	Development
	Staging
)

func (e Environment) String() string {
	switch e {
	case Development:
		return "development"
	case Staging:
		return "staging"
	case Production:
		return "production"
	default:
		return "production"
	}
}

func (e Environment) IsDevelopment() bool {
	return e == Development
}

func (e Environment) IsProduction() bool {
	return e == Production
}

func (e Environment) IsStaging() bool {
	return e == Staging
}

func GetAppEnv() Environment {
	envVars := []string{"GO_ENV", "APP_ENV"}
	for _, envVar := range envVars {
		if env := os.Getenv(envVar); env != "" {
			return parseEnvironment(strings.ToLower(strings.TrimSpace(env)))
		}
	}
	return Production
}

func GetLogFormat() string {
	return GetEnvString("LOG_FORMAT", "json")
}

func GetEnvString(name string, defaultValue ...string) string {
	value := os.Getenv(name)
	if value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	trimmed := strings.TrimSpace(value)
	return trimmed
}

func GetEnvStringRequired(name string, defaultValue ...string) string {
	value := os.Getenv(name)
	if value == "" {
		if len(defaultValue) > 0 {
			trimmed := strings.TrimSpace(defaultValue[0])
			return trimmed
		}
		slog.Error("Required environment variable is not set or empty", slog.String("name", name))
		os.Exit(1)
	}
	trimmed := strings.TrimSpace(value)
	return trimmed
}

func parseEnvironment(env string) Environment {
	switch env {
	case "development", "dev":
		return Development
	case "staging", "stage":
		return Staging
	case "production", "prod":
		return Production
	default:
		return Production
	}
}
