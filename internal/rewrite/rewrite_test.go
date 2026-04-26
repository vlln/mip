package rewrite

import (
	"testing"

	"github.com/vlln/mip/internal/ref"
	"github.com/vlln/mip/internal/registry"
)

func TestCandidatesForDockerHub(t *testing.T) {
	image, err := ref.Parse("nginx:1.27")
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := registry.FindProfile(image.Registry)
	if !ok {
		t.Fatal("missing docker.io profile")
	}

	got := Candidates(image, profile)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].Image != "docker.m.daocloud.io/library/nginx:1.27" {
		t.Fatalf("unexpected host replacement candidate: %s", got[0].Image)
	}
	if got[1].Image != "m.daocloud.io/docker.io/library/nginx:1.27" {
		t.Fatalf("unexpected prefix candidate: %s", got[1].Image)
	}
}

func TestCandidatesForGHCR(t *testing.T) {
	image, err := ref.Parse("ghcr.io/actions/actions-runner:latest")
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := registry.FindProfile(image.Registry)
	if !ok {
		t.Fatal("missing ghcr.io profile")
	}

	got := Candidates(image, profile)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].Image != "ghcr.m.daocloud.io/actions/actions-runner:latest" {
		t.Fatalf("unexpected host replacement candidate: %s", got[0].Image)
	}
	if got[1].Image != "m.daocloud.io/ghcr.io/actions/actions-runner:latest" {
		t.Fatalf("unexpected prefix candidate: %s", got[1].Image)
	}
}
