package routes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/services"
)

const maxProposalSimulationBody = 512 << 10

type ProposalSimulationRoute struct {
	service *services.ProposalSimulationService
	timeout time.Duration
	limiter *proposalSimulationLimiter
	slots   chan struct{}
}

type proposalSimulationLimit struct {
	window time.Time
	count  int
}

type proposalSimulationLimiter struct {
	mu     sync.Mutex
	limit  int
	counts map[string]proposalSimulationLimit
}

func NewProposalSimulationRoute() *ProposalSimulationRoute {
	cfg := config.GetConfig()
	timeout := cfg.GetDuration("SIMULATION_TIMEOUT")
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	limit := cfg.GetInt("SIMULATION_RATE_LIMIT_PER_MINUTE")
	if limit <= 0 {
		limit = 10
	}
	return &ProposalSimulationRoute{
		service: services.NewProposalSimulationService(),
		timeout: timeout,
		limiter: &proposalSimulationLimiter{limit: limit, counts: make(map[string]proposalSimulationLimit)},
		slots:   make(chan struct{}, 4),
	}
}

func (route *ProposalSimulationRoute) CapabilityHandler(w http.ResponseWriter, request *http.Request) {
	capability, err := route.service.Capability(request.Context(), request.PathValue("daoCode"))
	if err != nil {
		writeProposalSimulationError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeProposalSimulationJSON(w, http.StatusOK, capability)
}

func (route *ProposalSimulationRoute) SimulationHandler(w http.ResponseWriter, request *http.Request) {
	daoCode := request.PathValue("daoCode")
	if !route.limiter.Allow(proposalSimulationClientIP(request), daoCode, time.Now().UTC()) {
		writeProposalSimulationError(w, http.StatusTooManyRequests, "rate_limited")
		return
	}
	select {
	case route.slots <- struct{}{}:
		defer func() { <-route.slots }()
	default:
		writeProposalSimulationError(w, http.StatusServiceUnavailable, "simulation_busy")
		return
	}

	request.Body = http.MaxBytesReader(w, request.Body, maxProposalSimulationBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input services.ProposalSimulationRequest
	if err := decoder.Decode(&input); err != nil {
		writeProposalSimulationError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeProposalSimulationError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), route.timeout)
	defer cancel()
	result, err := route.service.Simulate(ctx, daoCode, request.PathValue("proposalId"), input)
	if err != nil {
		writeProposalSimulationServiceError(w, err)
		return
	}
	writeProposalSimulationJSON(w, http.StatusOK, result)
}

func (limiter *proposalSimulationLimiter) Allow(ip, daoCode string, now time.Time) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	window := now.Truncate(time.Minute)
	keys := []string{"ip:" + ip, "dao:" + daoCode}
	for _, key := range keys {
		entry := limiter.counts[key]
		if entry.window.Equal(window) && entry.count >= limiter.limit {
			return false
		}
	}
	for _, key := range keys {
		entry := limiter.counts[key]
		if !entry.window.Equal(window) {
			entry = proposalSimulationLimit{window: window}
		}
		entry.count++
		limiter.counts[key] = entry
	}
	if len(limiter.counts) > 1000 {
		for key, entry := range limiter.counts {
			if entry.window.Before(window) {
				delete(limiter.counts, key)
			}
		}
	}
	return true
}

func proposalSimulationClientIP(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	directIP := net.ParseIP(strings.TrimSpace(host))
	if directIP == nil {
		return request.RemoteAddr
	}
	if directIP.IsPrivate() || directIP.IsLoopback() {
		for _, value := range []string{
			request.Header.Get("X-Real-IP"),
			strings.Split(request.Header.Get("X-Forwarded-For"), ",")[0],
		} {
			if forwardedIP := net.ParseIP(strings.TrimSpace(value)); forwardedIP != nil {
				return forwardedIP.String()
			}
		}
	}
	return directIP.String()
}

func writeProposalSimulationServiceError(w http.ResponseWriter, err error) {
	var simulationErr *services.ProposalSimulationError
	if !errors.As(err, &simulationErr) {
		writeProposalSimulationError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	status := http.StatusBadGateway
	switch simulationErr.Code {
	case "invalid_request":
		status = http.StatusBadRequest
	case "proposal_mismatch":
		status = http.StatusUnprocessableEntity
	case "simulation_disabled":
		status = http.StatusConflict
	case "provider_rate_limited":
		status = http.StatusTooManyRequests
	case "provider_timeout":
		status = http.StatusGatewayTimeout
	}
	writeProposalSimulationError(w, status, simulationErr.Code)
}

func writeProposalSimulationError(w http.ResponseWriter, status int, code string) {
	writeProposalSimulationJSON(w, status, map[string]string{"error": code})
}

func writeProposalSimulationJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
