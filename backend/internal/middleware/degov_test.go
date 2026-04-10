package middleware

import "testing"

func TestHostMatchesModeRequiresNextMode(t *testing.T) {
	m := &DegovMiddleware{}

	if m.hostMatchesMode("adventuregold.degov.ai", "adventuregold.next.degov.ai") {
		t.Fatal("hostMatchesMode() matched next host without next mode enabled")
	}
}

func TestHostMatchesModeMatchesNextDomainInNextMode(t *testing.T) {
	t.Setenv("DAO_CONFIG_MODE", "next")
	m := &DegovMiddleware{}

	if !m.hostMatchesMode("adventuregold.degov.ai", "adventuregold.next.degov.ai") {
		t.Fatal("hostMatchesMode() did not match next host in next mode")
	}
}

func TestNextModeHostRewritesCanonicalDegovDomain(t *testing.T) {
	t.Parallel()

	if got, want := nextModeHost("adventuregold.degov.ai"), "adventuregold.next.degov.ai"; got != want {
		t.Fatalf("nextModeHost() = %q, want %q", got, want)
	}
}

func TestNextModeHostRewritesRingDaoDomain(t *testing.T) {
	t.Parallel()

	if got, want := nextModeHost("gov.ringdao.com"), "gov.next.degov.ai"; got != want {
		t.Fatalf("nextModeHost() = %q, want %q", got, want)
	}
}
