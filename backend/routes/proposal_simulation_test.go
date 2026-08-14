package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProposalSimulationLimiterAppliesPerIPAndDAO(t *testing.T) {
	limiter := &proposalSimulationLimiter{limit: 2, counts: make(map[string]proposalSimulationLimit)}
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	if !limiter.Allow("192.0.2.1", "demo", now) || !limiter.Allow("192.0.2.1", "demo", now) {
		t.Fatal("first two requests should be allowed")
	}
	if limiter.Allow("192.0.2.1", "other", now) {
		t.Fatal("per-IP limit should reject another DAO")
	}
	if limiter.Allow("192.0.2.2", "demo", now) {
		t.Fatal("per-DAO limit should reject another IP")
	}
	if !limiter.Allow("192.0.2.1", "demo", now.Add(time.Minute)) {
		t.Fatal("next window should be allowed")
	}
}

func TestProposalSimulationHandlerRejectsWhenConcurrencyIsFull(t *testing.T) {
	route := &ProposalSimulationRoute{
		timeout: time.Second,
		limiter: &proposalSimulationLimiter{limit: 10, counts: make(map[string]proposalSimulationLimit)},
		slots:   make(chan struct{}, 1),
	}
	route.slots <- struct{}{}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/daos/demo/proposals/1/simulation", strings.NewReader(`{}`))
	request.SetPathValue("daoCode", "demo")
	recorder := httptest.NewRecorder()

	route.SimulationHandler(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "simulation_busy") {
		t.Fatalf("busy response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestProposalSimulationClientIPTrustsOnlyPrivateProxy(t *testing.T) {
	publicRequest := httptest.NewRequest(http.MethodPost, "/", nil)
	publicRequest.RemoteAddr = "203.0.113.10:1234"
	publicRequest.Header.Set("X-Real-IP", "198.51.100.20")
	if got := proposalSimulationClientIP(publicRequest); got != "203.0.113.10" {
		t.Fatalf("public direct client IP = %q", got)
	}

	proxyRequest := httptest.NewRequest(http.MethodPost, "/", nil)
	proxyRequest.RemoteAddr = "10.0.0.2:1234"
	proxyRequest.Header.Set("X-Forwarded-For", "198.51.100.20, 10.0.0.2")
	if got := proposalSimulationClientIP(proxyRequest); got != "198.51.100.20" {
		t.Fatalf("trusted proxy client IP = %q", got)
	}

	proxyRequest.Header.Set("X-Real-IP", "not-an-ip")
	proxyRequest.Header.Set("X-Forwarded-For", "also-invalid")
	if got := proposalSimulationClientIP(proxyRequest); got != "10.0.0.2" {
		t.Fatalf("invalid forwarded client IP = %q", got)
	}
}
