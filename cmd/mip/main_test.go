package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vlln/mip/internal/engine"
	"github.com/vlln/mip/internal/probe"
	"github.com/vlln/mip/internal/ref"
	"github.com/vlln/mip/internal/registry"
	"github.com/vlln/mip/internal/state"
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

func TestBuildProbeCandidatesAddsSourceFallback(t *testing.T) {
	image, err := ref.Parse("nginx:1.27")
	if err != nil {
		t.Fatal(err)
	}

	got := buildProbeCandidates(registry.Builtins(), state.Store{}, image)
	if len(got) != 3 {
		t.Fatalf("candidate count = %d, want 3", len(got))
	}
	last := got[len(got)-1]
	if last.Image != "docker.io/library/nginx:1.27" {
		t.Fatalf("source image = %q", last.Image)
	}
	if last.Mirror.Name != "source" {
		t.Fatalf("source mirror = %q", last.Mirror.Name)
	}
}

func TestSortProbeResultsKeepsSourceAfterMirrors(t *testing.T) {
	results := []probe.Result{
		{Image: "docker.io/library/nginx:1.27", Mirror: "source", OK: true, LatencyMS: 10},
		{Image: "mirror.example/library/nginx:1.27", Mirror: "mirror", OK: true, LatencyMS: 100},
	}

	sortProbeResults(results)

	if results[0].Mirror != "mirror" {
		t.Fatalf("first mirror = %q, want mirror", results[0].Mirror)
	}
	if results[1].Mirror != "source" {
		t.Fatalf("last mirror = %q, want source", results[1].Mirror)
	}
}

func TestPullWithFallbackUsesNextCandidate(t *testing.T) {
	runner := &fakeEngine{
		pullErrors: map[string][]error{
			"mirror.example/library/nginx:1.27": {errors.New("pull failed")},
		},
		repoDigests: map[string][]string{
			"source.example/library/nginx:1.27": {"source.example/library/nginx@sha256:ok"},
		},
	}
	candidates := []probe.Result{
		{Image: "mirror.example/library/nginx:1.27", OK: true, Digest: "sha256:bad"},
		{Image: "source.example/library/nginx:1.27", OK: true, Digest: "sha256:ok"},
	}

	outcome, code := pullWithFallback(context.Background(), runner, "source.example/library/nginx:1.27", candidates, pullOptions{retries: 1})
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	if outcome.Selected.Image != "source.example/library/nginx:1.27" {
		t.Fatalf("selected = %q", outcome.Selected.Image)
	}
	if len(outcome.Attempts) != 2 {
		t.Fatalf("attempts = %d, want 2", len(outcome.Attempts))
	}
}

func TestPullWithFallbackRetriesCandidate(t *testing.T) {
	oldSleep := retrySleep
	retrySleep = func(time.Duration) {}
	t.Cleanup(func() { retrySleep = oldSleep })

	runner := &fakeEngine{
		pullErrors: map[string][]error{
			"mirror.example/library/nginx:1.27": {errors.New("temporary failure")},
		},
		repoDigests: map[string][]string{
			"mirror.example/library/nginx:1.27": {"mirror.example/library/nginx@sha256:ok"},
		},
	}
	candidates := []probe.Result{
		{Image: "mirror.example/library/nginx:1.27", OK: true, Digest: "sha256:ok"},
	}

	outcome, code := pullWithFallback(context.Background(), runner, "docker.io/library/nginx:1.27", candidates, pullOptions{retries: 2})
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	if len(outcome.Attempts) != 2 {
		t.Fatalf("attempts = %d, want 2", len(outcome.Attempts))
	}
	if !outcome.Retagged {
		t.Fatal("expected retag")
	}
}

type fakeEngine struct {
	pullErrors  map[string][]error
	repoDigests map[string][]string
	pulled      []string
	tagged      [][2]string
	removed     []string
}

func (f *fakeEngine) Name() string {
	return "fake"
}

func (f *fakeEngine) Available(context.Context) error {
	return nil
}

func (f *fakeEngine) Pull(_ context.Context, image string, _ engine.PullOptions) error {
	f.pulled = append(f.pulled, image)
	if len(f.pullErrors[image]) == 0 {
		return nil
	}
	err := f.pullErrors[image][0]
	f.pullErrors[image] = f.pullErrors[image][1:]
	return err
}

func (f *fakeEngine) Tag(_ context.Context, source string, target string) error {
	f.tagged = append(f.tagged, [2]string{source, target})
	return nil
}

func (f *fakeEngine) Remove(_ context.Context, image string) error {
	f.removed = append(f.removed, image)
	return nil
}

func (f *fakeEngine) RepoDigests(_ context.Context, image string) ([]string, error) {
	return f.repoDigests[image], nil
}
