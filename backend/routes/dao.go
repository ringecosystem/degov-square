package routes

import (
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/ringecosystem/degov-apps/internal/middleware"
	"github.com/ringecosystem/degov-apps/services"
	"github.com/ringecosystem/degov-apps/types"
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
	responseContent, err := d.daoConfigService.RawConfig(types.RawDaoConfigInput{
		Code:   daoCode,
		Format: format,
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
