package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal/middleware"
	"github.com/ringecosystem/degov-apps/services"
)

type DaoRoute struct {
	daoConfigService *services.DaoConfigService
	configCache      *cache.Cache
}

// NewDaoRoute creates a new DAO route handler
func NewDaoRoute() *DaoRoute {
	// Create cache with 15 seconds TTL and 30 seconds cleanup interval
	c := cache.New(15*time.Second, 30*time.Second)

	return &DaoRoute{
		daoConfigService: services.NewDaoConfigService(),
		configCache:      c,
	}
}

func (d *DaoRoute) DetectDaoCode(w http.ResponseWriter, r *http.Request) {
	contextDaocode, ok := r.Context().Value(middleware.DegovDaocodeKey).(string)
	if !ok {
		http.Error(w, "Failed to detect dao code", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := `{"code": "` + contextDaocode + `"}`
	w.Write([]byte(response))
}

// ConfigHandler handles the /dao/config and /dao/config/{dao} endpoints
func (d *DaoRoute) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	// Get parameters from URL query
	queryParams := r.URL.Query()
	format := strings.ToLower(queryParams.Get("format"))

	// Default format is yml
	if format == "" {
		format = "yml"
	}

	// Validate format parameter
	if format != "yml" && format != "yaml" && format != "json" {
		http.Error(w, "Invalid format parameter. Supported formats: yml, yaml, json", http.StatusBadRequest)
		return
	}

	// Convert format string to gqlmodels.ConfigFormat
	var configFormat gqlmodels.ConfigFormat
	if strings.EqualFold(format, "json") {
		configFormat = gqlmodels.ConfigFormatJSON
	} else {
		// yml, yaml both map to YAML
		configFormat = gqlmodels.ConfigFormatYaml
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

	// Create cache key based on dao code and format
	cacheKey := daoCode + ":" + format

	var contentType string
	if format == "json" {
		contentType = "application/json"
	} else {
		contentType = "text/yaml"
	}
	w.Header().Set("Content-Type", contentType)

	// Try to get from cache first
	if cached, found := d.configCache.Get(cacheKey); found {
		if responseContent, ok := cached.(string); ok {
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(responseContent))
			return
		}
	}

	// Cache miss, fetch from service
	responseContent, err := d.daoConfigService.RawConfig(gqlmodels.GetDaoConfigInput{
		DaoCode: daoCode,
		Format:  &configFormat,
	})
	if err != nil {
		http.Error(w, "Failed to retrieve DAO configuration", http.StatusInternalServerError)
		return
	}

	// Store in cache
	d.configCache.Set(cacheKey, responseContent, cache.DefaultExpiration)

	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(responseContent))
}
