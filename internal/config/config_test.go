package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vlln/mip/internal/registry"
)

func TestLoadConfigFile(t *testing.T) {
	path := writeConfig(t, `
prefer:
  - company-cache
registries:
  docker.io:
    mirrors:
      - registry.example.com/docker.io
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Engine != "docker" {
		t.Fatalf("engine = %q", cfg.Engine)
	}
	if cfg.Prefer[0] != "company-cache" {
		t.Fatalf("prefer = %#v", cfg.Prefer)
	}
	if got := cfg.Registries["docker.io"].Mirrors[0]; got != "registry.example.com/docker.io" {
		t.Fatalf("mirror = %q", got)
	}
}

func TestLoadRejectsOldConfigFields(t *testing.T) {
	path := writeConfig(t, `
timeout: 2s
registries:
  docker.io:
    mirrors:
      - name: company-cache
        host: registry.example.com/docker.io
`)

	if _, err := Load(path); err == nil {
		t.Fatal("expected old config fields to be rejected")
	}
}

func TestLoadUsesOnlyUserConfigWhenPresent(t *testing.T) {
	path := writeConfig(t, `
registries:
  docker.io:
    mirrors:
      - registry.example.com/docker.io
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := FindProfile(Profiles(cfg), "ghcr.io"); ok {
		t.Fatal("user config should replace the official config")
	}

	profile, ok := FindProfile(Profiles(cfg), "docker.io")
	if !ok || len(profile.Mirrors) != 1 {
		t.Fatalf("unexpected docker.io profile: %+v", profile)
	}
	if profile.Mirrors[0].Name != "registry.example.com/docker.io" {
		t.Fatalf("mirror name = %q", profile.Mirrors[0].Name)
	}
	if profile.Mirrors[0].Mode != registry.Prefix {
		t.Fatalf("mirror mode = %q, want prefix", profile.Mirrors[0].Mode)
	}
}

func TestProfilesPreferMirror(t *testing.T) {
	cfg := Config{
		Prefer: []string{"company-cache"},
		Registries: map[string]RegistryOverride{
			"docker.io": {
				Mirrors: []string{"company-cache"},
			},
		},
	}

	profile, _ := FindProfile(Profiles(cfg), "docker.io")
	if got := profile.Mirrors[0].Priority; got != 2000 {
		t.Fatalf("preferred mirror priority = %d, want 2000", got)
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
