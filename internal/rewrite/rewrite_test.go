package rewrite

import (
	"testing"

	appconfig "github.com/vlln/mip/internal/config"
	"github.com/vlln/mip/internal/ref"
)

func TestCandidatesForDockerHub(t *testing.T) {
	image, err := ref.Parse("nginx:1.27")
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := appconfig.FindProfile(appconfig.Profiles(appconfig.Default()), image.Registry)
	if !ok {
		t.Fatal("missing docker.io profile")
	}

	got := Candidates(image, profile)
	if len(got) != len(profile.Mirrors) {
		t.Fatalf("expected one candidate per mirror, got %d candidates for %d mirrors", len(got), len(profile.Mirrors))
	}
	if got[0].Image != profile.Mirrors[0].Host+"/library/nginx:1.27" {
		t.Fatalf("unexpected host replacement candidate: %s", got[0].Image)
	}
	if got[1].Image != profile.Mirrors[1].Host+"/library/nginx:1.27" {
		t.Fatalf("unexpected prefix candidate: %s", got[1].Image)
	}
	if !hasCandidate(got, "docker.1panel.live/library/nginx:1.27") {
		t.Fatal("missing curated Docker Hub mirror candidate")
	}
}

func TestCandidatesForGHCR(t *testing.T) {
	image, err := ref.Parse("ghcr.io/actions/actions-runner:latest")
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := appconfig.FindProfile(appconfig.Profiles(appconfig.Default()), image.Registry)
	if !ok {
		t.Fatal("missing ghcr.io profile")
	}

	got := Candidates(image, profile)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].Image != profile.Mirrors[0].Host+"/actions/actions-runner:latest" {
		t.Fatalf("unexpected host replacement candidate: %s", got[0].Image)
	}
	if got[1].Image != profile.Mirrors[1].Host+"/actions/actions-runner:latest" {
		t.Fatalf("unexpected prefix candidate: %s", got[1].Image)
	}
}

func hasCandidate(candidates []Candidate, image string) bool {
	for _, candidate := range candidates {
		if candidate.Image == image {
			return true
		}
	}
	return false
}
