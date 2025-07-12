package tasks

import (
	"os"
	"strconv"
	"time"
)

// TaskConfig represents the configuration for a background task
type TaskConfig struct {
	Name     string
	Interval time.Duration
	Enabled  bool
}

// TaskDefinition combines configuration with constructor
type TaskDefinition struct {
	Config      TaskConfig
	Constructor func() Task
}

// GetTaskDefinitions returns all task definitions with their configurations
func GetTaskDefinitions() []TaskDefinition {
	return []TaskDefinition{
		{
			Config: TaskConfig{
				Name:     "dao-sync",
				Interval: getEnvDuration("TASK_DAO_SYNC_INTERVAL", 5*time.Minute),
				Enabled:  getEnvBool("TASK_DAO_SYNC_ENABLED", true),
			},
			Constructor: func() Task { return NewDaoSyncTask() },
		},
		{
			Config: TaskConfig{
				Name:     "notification-cleanup",
				Interval: getEnvDuration("TASK_NOTIFICATION_CLEANUP_INTERVAL", 30*time.Minute),
				Enabled:  getEnvBool("TASK_NOTIFICATION_CLEANUP_ENABLED", true),
			},
			Constructor: func() Task { return NewNotificationTask() },
		},
		// Add more task definitions here
	}
}

// getEnvDuration gets duration from environment variable with fallback
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}

// getEnvBool gets boolean from environment variable with fallback
func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}

// TaskRegistry holds all available task constructors (deprecated)
// Use GetTaskDefinitions() instead for better configuration management
type TaskRegistry struct {
	constructors map[string]func() Task
}

// NewTaskRegistry creates a new task registry (deprecated)
func NewTaskRegistry() *TaskRegistry {
	registry := &TaskRegistry{
		constructors: make(map[string]func() Task),
	}

	// Auto-register tasks from definitions
	for _, def := range GetTaskDefinitions() {
		registry.Register(def.Config.Name, def.Constructor)
	}

	return registry
}

// Register adds a task constructor to the registry
func (tr *TaskRegistry) Register(name string, constructor func() Task) {
	tr.constructors[name] = constructor
}

// Create creates a task instance by name
func (tr *TaskRegistry) Create(name string) Task {
	if constructor, exists := tr.constructors[name]; exists {
		return constructor()
	}
	return nil
}

// GetAvailableTasks returns a list of all registered task names
func (tr *TaskRegistry) GetAvailableTasks() []string {
	names := make([]string, 0, len(tr.constructors))
	for name := range tr.constructors {
		names = append(names, name)
	}
	return names
}
