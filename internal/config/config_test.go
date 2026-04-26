package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vlln/mip/internal/registry"
)

func TestLoadConfigFile(t *testing.T) {
	path := writeConfig(t, `
engine: podman
timeout: 2s
pull_timeout: 3m
parallel_probe: 2
retries: 4
prefer:
  - company-cache
registries:
  docker.io:
    mirrors:
      - name: company-cache
        host: registry.example.com/docker.io
        mode: prefix
        priority: 100
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Engine != "podman" {
		t.Fatalf("engine = %q", cfg.Engine)
	}
	if cfg.Timeout.String() != "2s" {
		t.Fatalf("timeout = %s", cfg.Timeout)
	}
	if cfg.PullTimeout.String() != "3m0s" {
		t.Fatalf("pull_timeout = %s", cfg.PullTimeout)
	}
	if cfg.ParallelProbe != 2 {
		t.Fatalf("parallel_probe = %d", cfg.ParallelProbe)
	}
	if cfg.Retries != 4 {
		t.Fatalf("retries = %d", cfg.Retries)
	}
}

func TestProfilesMergeCustomMirrorAndPrefer(t *testing.T) {
	cfg := Default()
	cfg.Prefer = []string{"company-cache"}
	cfg.Registries = map[string]RegistryOverride{
		"docker.io": {
			Mirrors: []registry.Mirror{
				{Name: "company-cache", Host: "registry.example.com/docker.io", Mode: registry.Prefix, Priority: 100},
			},
		},
	}

	profile, ok := FindProfile(Profiles(cfg), "docker.io")
	if !ok {
		t.Fatal("missing docker.io profile")
	}

	var found registry.Mirror
	for _, mirror := range profile.Mirrors {
		if mirror.Name == "company-cache" {
			found = mirror
		}
	}
	if found.Name == "" {
		t.Fatal("custom mirror was not merged")
	}
	if found.Priority != 1100 {
		t.Fatalf("preferred mirror priority = %d, want 1100", found.Priority)
	}
}

func TestProfilesDisableBuiltins(t *testing.T) {
	cfg := Default()
	cfg.DisableBuiltinMirrors = true

	if profiles := Profiles(cfg); len(profiles) != 0 {
		t.Fatalf("profiles len = %d, want 0", len(profiles))
	}
}

func TestProfilesExcludeMirror(t *testing.T) {
	cfg := Default()
	cfg.Exclude = []string{"docker.m.daocloud.io"}

	profile, ok := FindProfile(Profiles(cfg), "docker.io")
	if !ok {
		t.Fatal("missing docker.io profile")
	}
	for _, mirror := range profile.Mirrors {
		if mirror.Host == "docker.m.daocloud.io" {
			t.Fatal("excluded mirror still present")
		}
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
