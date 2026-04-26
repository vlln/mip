package state

import (
	"testing"

	"github.com/vlln/mip/internal/probe"
	"github.com/vlln/mip/internal/registry"
	"github.com/vlln/mip/internal/rewrite"
)

func TestRecordUpdatesMirrorHealth(t *testing.T) {
	store := Store{Mirrors: map[string]MirrorHealth{}}
	store = store.Record([]probe.Result{
		{Image: "mirror.example/library/nginx:latest", Mirror: "example", OK: true, StatusCode: 200, LatencyMS: 123, Digest: "sha256:abc"},
		{Image: "bad.example/library/nginx:latest", Mirror: "bad", OK: false, StatusCode: 500, LatencyMS: 456, Error: "HTTP 500"},
		{Image: "login.example/library/nginx:latest", Mirror: "login", AuthRequired: true, StatusCode: 401, LatencyMS: 300, Warning: "authentication required"},
	})

	okHealth := store.Mirrors["mirror.example/library/nginx:latest"]
	if okHealth.Successes != 1 || okHealth.Failures != 0 || !okHealth.LastOK {
		t.Fatalf("unexpected ok health: %+v", okHealth)
	}

	badHealth := store.Mirrors["bad.example/library/nginx:latest"]
	if badHealth.Successes != 0 || badHealth.Failures != 1 || badHealth.LastOK {
		t.Fatalf("unexpected bad health: %+v", badHealth)
	}

	loginHealth := store.Mirrors["login.example/library/nginx:latest"]
	if loginHealth.Successes != 0 || loginHealth.Failures != 0 || loginHealth.LastStatusCode != 401 {
		t.Fatalf("unexpected login-required health: %+v", loginHealth)
	}
}

func TestRankAdjustsCandidatePriority(t *testing.T) {
	candidates := []rewrite.Candidate{
		{
			Image:    "bad.example/library/nginx:latest",
			Priority: 100,
			Mirror:   registry.Mirror{Name: "bad"},
		},
		{
			Image:    "good.example/library/nginx:latest",
			Priority: 100,
			Mirror:   registry.Mirror{Name: "good"},
		},
	}
	store := Store{Mirrors: map[string]MirrorHealth{
		"good.example/library/nginx:latest": {Successes: 3, LastOK: true, LastLatencyMS: 100},
		"bad.example/library/nginx:latest":  {Failures: 3, LastOK: false, LastLatencyMS: 6000},
	}}

	store.Rank(candidates)
	rewrite.SortCandidates(candidates)

	if candidates[0].Image != "good.example/library/nginx:latest" {
		t.Fatalf("expected good candidate first, got %s", candidates[0].Image)
	}
}
