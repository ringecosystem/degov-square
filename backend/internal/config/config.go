package config

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ringecosystem/degov-square/types"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm/logger"
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

// Config holds all application configuration
type Config struct {
	viper *viper.Viper
}

// Global configuration instance
var globalConfig *Config

// InitConfig initializes the global configuration using viper
func InitConfig() error {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure viper to read environment variables
	v.AutomaticEnv()
	v.AllowEmptyEnv(true)

	// Set key replacer to handle case conversion
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	globalConfig = &Config{viper: v}

	slog.Info("Configuration initialized successfully")
	return nil
}

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	if globalConfig == nil {
		if err := InitConfig(); err != nil {
			slog.Error("Failed to initialize config", "error", err)
			os.Exit(1)
		}
	}
	return globalConfig
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("PORT", "8080")

	// Database defaults
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "5432")
	v.SetDefault("DB_SSLMODE", "disable")

	// Log defaults
	v.SetDefault("LOG_FORMAT", "json")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("DB_LOG_LEVEL", "warn")

	// Environment defaults
	v.SetDefault("APP_ENV", "production")

	// Task defaults
	v.SetDefault("TASK_DAO_SYNC_ENABLED", true)
	v.SetDefault("TASK_DAO_SYNC_INTERVAL", "5m")
	v.SetDefault("TASK_VOTE_TRACKING_ENABLED", false)
	v.SetDefault("TASK_VOTE_TRACKING_INTERVAL", "3m")
	v.SetDefault("TASK_VOTE_END_TRACKING_ENABLED", true)
	v.SetDefault("TASK_VOTE_END_TRACKING_INTERVAL", "4m")
	v.SetDefault("TASK_PROPOSAL_TRACKING_ENABLED", true)
	v.SetDefault("TASK_PROPOSAL_TRACKING_INTERVAL", "3m")
	v.SetDefault("TASK_PROPOSAL_FULFILL_ENABLED", false)
	v.SetDefault("TASK_PROPOSAL_FULFILL_INTERVAL", "30s")
	v.SetDefault("TASK_NOTIFICATION_EVENT_ENABLED", true)
	v.SetDefault("TASK_NOTIFICATION_EVENT_INTERVAL", "10s")
	v.SetDefault("TASK_NOTIFICATION_DISPATCHER_ENABLED", true)
	v.SetDefault("TASK_NOTIFICATION_DISPATCHER_INTERVAL", "5s")

	// sendgrid
	v.SetDefault("SENDGRID_FROM_USER", "DeGov Notifications")
	v.SetDefault("SENDGRID_FROM_EMAIL", "notifications@degov.ai")

	// Governor voter defaults
	v.SetDefault("DEGOV_AGENT_GAS_BUFFER_PERCENT", 20)
}

// Server configuration methods
func (c *Config) GetPort() string {
	return c.viper.GetString("PORT")
}

// Database configuration methods
func (c *Config) GetDBHost() string {
	return c.viper.GetString("DB_HOST")
}

func (c *Config) GetDBUser() string {
	return c.viper.GetString("DB_USER")
}

func (c *Config) GetDBPassword() string {
	return c.viper.GetString("DB_PASSWORD")
}

func (c *Config) GetDBName() string {
	return c.viper.GetString("DB_NAME")
}

func (c *Config) GetDBPort() string {
	return c.viper.GetString("DB_PORT")
}

func (c *Config) GetDBSSLMode() string {
	return c.viper.GetString("DB_SSLMODE")
}

// Log configuration methods
func (c *Config) GetLogFormat() string {
	return c.viper.GetString("LOG_FORMAT")
}

func (c *Config) GetLogLevel() zapcore.Level {
	levelStr := strings.ToLower(c.viper.GetString("LOG_LEVEL"))
	switch levelStr {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func (c *Config) GetDBLogLevel() logger.LogLevel {
	levelStr := strings.ToLower(c.viper.GetString("DB_LOG_LEVEL"))
	switch levelStr {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}

// Environment configuration methods
func (c *Config) GetAppEnv() Environment {
	// Try GO_ENV first, then APP_ENV
	env := c.viper.GetString("APP_ENV")
	return parseEnvironment(strings.ToLower(strings.TrimSpace(env)))
}

// Task configuration methods
func (c *Config) GetTaskDAOSyncEnabled() bool {
	return c.viper.GetBool("TASK_DAO_SYNC_ENABLED")
}

func (c *Config) GetTaskDAOSyncInterval() time.Duration {
	return c.viper.GetDuration("TASK_DAO_SYNC_INTERVAL")
}

func (c *Config) GetTaskVoteTrackingEnabled() bool {
	return c.viper.GetBool("TASK_VOTE_TRACKING_ENABLED")
}

func (c *Config) GetTaskVoteTrackingInterval() time.Duration {
	return c.viper.GetDuration("TASK_VOTE_TRACKING_INTERVAL")
}

func (c *Config) GetTaskVoteEndTrackingEnabled() bool {
	return c.viper.GetBool("TASK_VOTE_END_TRACKING_ENABLED")
}

func (c *Config) GetTaskVoteEndTrackingInterval() time.Duration {
	return c.viper.GetDuration("TASK_VOTE_END_TRACKING_INTERVAL")
}

func (c *Config) GetTaskProposalTrackingEnabled() bool {
	return c.viper.GetBool("TASK_PROPOSAL_TRACKING_ENABLED")
}

func (c *Config) GetTaskProposalTrackingInterval() time.Duration {
	return c.viper.GetDuration("TASK_PROPOSAL_TRACKING_INTERVAL")
}

func (c *Config) GetTaskProposalFulfillEnabled() bool {
	return c.viper.GetBool("TASK_PROPOSAL_FULFILL_ENABLED")
}

func (c *Config) GetTaskProposalFulfillInterval() time.Duration {
	return c.viper.GetDuration("TASK_PROPOSAL_FULFILL_INTERVAL")
}

func (c *Config) GetTaskNotificationEventEnabled() bool {
	return c.viper.GetBool("TASK_NOTIFICATION_EVENT_ENABLED")
}

func (c *Config) GetTaskNotificationEventInterval() time.Duration {
	return c.viper.GetDuration("TASK_NOTIFICATION_EVENT_INTERVAL")
}

func (c *Config) GetTaskNotificationDispatcherEnabled() bool {
	return c.viper.GetBool("TASK_NOTIFICATION_DISPATCHER_ENABLED")
}

func (c *Config) GetTaskNotificationDispatcherInterval() time.Duration {
	return c.viper.GetDuration("TASK_NOTIFICATION_DISPATCHER_INTERVAL")
}

// Generic configuration methods
func (c *Config) GetString(key string) string {
	return c.viper.GetString(key)
}

func (c *Config) GetStringWithDefault(key, defaultValue string) string {
	value := c.viper.GetString(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (c *Config) GetStringRequired(key string) string {
	value := c.viper.GetString(key)
	if value == "" {
		slog.Error("Required configuration key is not set or empty", slog.String("key", key))
		os.Exit(1)
	}
	return value
}

func (c *Config) GetBool(key string) bool {
	return c.viper.GetBool(key)
}

func (c *Config) GetInt(key string) int {
	return c.viper.GetInt(key)
}

func (c *Config) GetDuration(key string) time.Duration {
	return c.viper.GetDuration(key)
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

// Convenience functions for backward compatibility
func GetAppEnv() Environment {
	return GetConfig().GetAppEnv()
}

func GetLogFormat() string {
	return GetConfig().GetLogFormat()
}

func GetLogLevel() zapcore.Level {
	return GetConfig().GetLogLevel()
}

func GetDBLogLevel() logger.LogLevel {
	return GetConfig().GetDBLogLevel()
}

func GetString(key string) string {
	return GetConfig().GetString(key)
}

func GetStringWithDefault(key, defaultValue string) string {
	return GetConfig().GetStringWithDefault(key, defaultValue)
}

func GetStringRequired(key string) string {
	return GetConfig().GetStringRequired(key)
}

func GetBool(key string) bool {
	return GetConfig().GetBool(key)
}

func GetInt(key string) int {
	return GetConfig().GetInt(key)
}

func GetDuration(key string) time.Duration {
	return GetConfig().GetDuration(key)
}

// GetGasBufferPercent returns the gas buffer percentage for governor voting
func GetGasBufferPercent() int {
	return GetConfig().GetInt("DEGOV_AGENT_GAS_BUFFER_PERCENT")
}

func GetEmailStyle() types.EmailStyle {
	return types.EmailStyle{
		ContainerMaxWidth: "600px",
	}
}

func GetDegovSiteConfig() types.DegovSiteConfig {
	emailProposalIncludeDescription := GetStringWithDefault("DEGOV_SITE_EMAIL_PROPOSAL_INCLUDE_DESCRIPTION", "false")
	return types.DegovSiteConfig{
		EmailTheme:                      GetStringWithDefault("DEGOV_SITE_EMAIL_THEME", "dark"),
		EmailProposalIncludeDescription: emailProposalIncludeDescription == "true",
		Logo:                            GetStringWithDefault("DEGOV_SITE_LOGO", "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/light-degov-4x.png"),
		LogoLight:                       GetStringWithDefault("DEGOV_SITE_LOGO_LIGHT", "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/light-degov-4x.png"),
		LogoDark:                        GetStringWithDefault("DEGOV_SITE_LOGO_DARK", "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/dark-degov-4x.png"),
		Name:                            GetStringWithDefault("DEGOV_SITE_NAME", "DeGov.AI"),
		Home:                            GetStringWithDefault("DEGOV_SITE_HOME", "https://degov.ai"),
		Square:                          GetStringWithDefault("DEGOV_SITE_SQUARE", "https://square.degov.ai"),
		Docs:                            GetStringWithDefault("DEGOV_SITE_DOCS", "https://docs.degov.ai"),
		Socials: []types.DegovSiteConfigSocial{
			{
				Name:      "Twitter",
				Icon:      "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/light-x-4x.png",
				IconLight: "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/light-x-4x.png",
				IconDark:  "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/dark-x-4x.png",
				Link:      "https://x.com/ai_degov",
			},
			{
				Name:      "Telegram",
				Icon:      "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/light-telegram-4x.png",
				IconLight: "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/light-telegram-4x.png",
				IconDark:  "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/dark-telegram-4x.png",
				Link:      "https://t.me/RingDAO_Hub",
			},
			{
				Name:      "GitHub",
				Icon:      "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/light-github-4x.png",
				IconLight: "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/light-github-4x.png",
				IconDark:  "https://cdn.jsdelivr.net/gh/ringecosystem/degov-registry@main/assets/common/dark-github-4x.png",
				Link:      "https://github.com/ringecosystem/degov",
			},
		},
	}
}
