package ref

import "testing"

func TestParseNormalizesDockerHubShortName(t *testing.T) {
	got, err := Parse("nginx:1.27")
	if err != nil {
		t.Fatal(err)
	}

	if got.String() != "docker.io/library/nginx:1.27" {
		t.Fatalf("unexpected normalized ref: %s", got.String())
	}
	if got.Familiar() != "nginx:1.27" {
		t.Fatalf("unexpected familiar ref: %s", got.Familiar())
	}
}

func TestParseDefaultsTag(t *testing.T) {
	got, err := Parse("redis")
	if err != nil {
		t.Fatal(err)
	}

	if got.String() != "docker.io/library/redis:latest" {
		t.Fatalf("unexpected normalized ref: %s", got.String())
	}
}

func TestParseKeepsExplicitRegistry(t *testing.T) {
	got, err := Parse("ghcr.io/actions/actions-runner:latest")
	if err != nil {
		t.Fatal(err)
	}

	if got.String() != "ghcr.io/actions/actions-runner:latest" {
		t.Fatalf("unexpected normalized ref: %s", got.String())
	}
}

func TestParseDigest(t *testing.T) {
	got, err := Parse("registry.k8s.io/pause@sha256:abc")
	if err != nil {
		t.Fatal(err)
	}

	if got.String() != "registry.k8s.io/pause@sha256:abc" {
		t.Fatalf("unexpected normalized ref: %s", got.String())
	}
}
