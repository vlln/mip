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

func TestLoadUsesOnlyUserConfigWhenPresent(t *testing.T) {
	path := writeConfig(t, `
registries:
  docker.io:
    mirrors:
      - name: company-cache
        host: registry.example.com/docker.io
        mode: prefix
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := FindProfile(Profiles(cfg), "ghcr.io"); ok {
		t.Fatal("official ghcr.io profile should not be merged into user config")
	}

	profile, ok := FindProfile(Profiles(cfg), "docker.io")
	if !ok || len(profile.Mirrors) != 1 {
		t.Fatalf("unexpected docker.io profile: %+v", profile)
	}
}

func TestProfilesPreferMirror(t *testing.T) {
	cfg := Config{
		Prefer: []string{"company-cache"},
		Registries: map[string]RegistryOverride{
			"docker.io": {
				Mirrors: []registry.Mirror{
					{Name: "company-cache", Host: "registry.example.com/docker.io", Mode: registry.Prefix, Priority: 100},
				},
			},
		},
	}

	profile, _ := FindProfile(Profiles(cfg), "docker.io")
	if got := profile.Mirrors[0].Priority; got != 1100 {
		t.Fatalf("preferred mirror priority = %d, want 1100", got)
	}
}

func TestProfilesExcludeMirror(t *testing.T) {
	cfg := Default()
	defaultProfile, ok := FindProfile(Profiles(cfg), "docker.io")
	if !ok || len(defaultProfile.Mirrors) == 0 {
		t.Fatal("missing docker.io default mirrors")
	}
	excludedHost := defaultProfile.Mirrors[0].Host
	cfg.Exclude = []string{excludedHost}

	profile, ok := FindProfile(Profiles(cfg), "docker.io")
	if !ok {
		t.Fatal("missing docker.io profile")
	}
	for _, mirror := range profile.Mirrors {
		if mirror.Host == excludedHost {
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
