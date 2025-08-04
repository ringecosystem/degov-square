package routes

import (
	"encoding/json"
	"net/http"

	"github.com/ringecosystem/degov-apps/internal/middleware"
	"github.com/ringecosystem/degov-apps/services"
	"gopkg.in/yaml.v3"
)

type DaoRoute struct {
	daoConfigService *services.DaoConfigService
}

// NewDaoRoute creates a new DAO route handler
func NewDaoRoute() *DaoRoute {
	return &DaoRoute{
		daoConfigService: services.NewDaoConfigService(),
	}
}

// ConfigHandler handles the /dao/config and /dao/config/{dao} endpoints
func (d *DaoRoute) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	// Get parameters from URL query
	queryParams := r.URL.Query()
	format := queryParams.Get("format")

	// Default format is yml
	if format == "" {
		format = "yml"
	}

	// Validate format parameter
	if format != "yml" && format != "yaml" && format != "json" {
		http.Error(w, "Invalid format parameter. Supported formats: yml, yaml, json", http.StatusBadRequest)
		return
	}

	var daoCode string

	// First try to get DAO code from path parameter (Go 1.22+ syntax)
	pathDao := r.PathValue("dao")
	if pathDao != "" {
		daoCode = pathDao
	} else {
		// Fallback to query parameter for backward compatibility
		daoCode = queryParams.Get("dao")

		// If still no dao code, try to get from context (set by DegovMiddleware)
		if daoCode == "" {
			contextDaocode, ok := r.Context().Value(middleware.DegovDaocodeKey).(string)
			if ok {
				daoCode = contextDaocode
			}
		}
	}

	if daoCode == "" {
		http.Error(w, "DAO code is required", http.StatusBadRequest)
		return
	}

	daoConfig, err := d.daoConfigService.Inspect(daoCode)
	if err != nil {
		http.Error(w, "Failed to retrieve DAO configuration", http.StatusInternalServerError)
		return
	}

	var responseContent []byte
	var contentType string

	if format == "json" {
		// Convert YAML to JSON
		var yamlData interface{}
		err := yaml.Unmarshal([]byte(daoConfig.Config), &yamlData)
		if err != nil {
			http.Error(w, "Failed to parse YAML configuration", http.StatusInternalServerError)
			return
		}

		jsonData, err := json.MarshalIndent(yamlData, "", "  ")
		if err != nil {
			http.Error(w, "Failed to convert to JSON", http.StatusInternalServerError)
			return
		}

		responseContent = jsonData
		contentType = "application/json"
	} else {
		// Default YAML format
		responseContent = []byte(daoConfig.Config)
		contentType = "text/yaml"
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(responseContent)
}
