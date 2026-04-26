package main

import (
	"testing"

	"github.com/vlln/mip/internal/probe"
)

func TestHasAnyDigest(t *testing.T) {
	repoDigests := []string{
		"example.com/library/nginx@sha256:abc",
		"mirror.example/library/redis@sha256:def",
	}

	if !hasAnyDigest(repoDigests, []string{"sha256:missing", "sha256:abc"}) {
		t.Fatal("expected sha256:abc to match")
	}
	if hasAnyDigest(repoDigests, []string{"sha256:missing"}) {
		t.Fatal("did not expect missing digest to match")
	}
}

func TestVerificationDigestsPrefersIndex(t *testing.T) {
	got := verificationDigests(probe.Result{
		Digest:      "sha256:child",
		IndexDigest: "sha256:index",
	})

	if len(got) != 2 || got[0] != "sha256:index" || got[1] != "sha256:child" {
		t.Fatalf("unexpected digests: %#v", got)
	}
}
