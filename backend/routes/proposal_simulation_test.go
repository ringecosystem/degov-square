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
