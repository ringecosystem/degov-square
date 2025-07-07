package internal

import (
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
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "json"
	}
	return strings.ToLower(strings.TrimSpace(logFormat))
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
