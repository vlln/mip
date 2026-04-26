package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/vlln/mip/internal/engine"
	"github.com/vlln/mip/internal/registry"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Engine                string                      `json:"engine" yaml:"engine"`
	Timeout               time.Duration               `json:"timeout" yaml:"-"`
	PullTimeout           time.Duration               `json:"pull_timeout" yaml:"-"`
	ParallelProbe         int                         `json:"parallel_probe" yaml:"parallel_probe"`
	Retries               int                         `json:"retries" yaml:"retries"`
	DisableBuiltinMirrors bool                        `json:"disable_builtin_mirrors" yaml:"disable_builtin_mirrors"`
	DisabledMirrors       []string                    `json:"disabled_mirrors" yaml:"disabled_mirrors"`
	Prefer                []string                    `json:"prefer" yaml:"prefer"`
	Exclude               []string                    `json:"exclude" yaml:"exclude"`
	Registries            map[string]RegistryOverride `json:"registries" yaml:"registries"`
	LoadedFrom            string                      `json:"loaded_from,omitempty" yaml:"-"`
}

type RegistryOverride struct {
	Aliases          []string          `json:"aliases,omitempty" yaml:"aliases"`
	DefaultNamespace string            `json:"default_namespace,omitempty" yaml:"default_namespace"`
	Mirrors          []registry.Mirror `json:"mirrors,omitempty" yaml:"mirrors"`
}

type fileConfig struct {
	Engine                string                      `yaml:"engine"`
	Timeout               string                      `yaml:"timeout"`
	PullTimeout           string                      `yaml:"pull_timeout"`
	ParallelProbe         int                         `yaml:"parallel_probe"`
	Retries               int                         `yaml:"retries"`
	DisableBuiltinMirrors bool                        `yaml:"disable_builtin_mirrors"`
	DisabledMirrors       []string                    `yaml:"disabled_mirrors"`
	Prefer                []string                    `yaml:"prefer"`
	Exclude               []string                    `yaml:"exclude"`
	Registries            map[string]RegistryOverride `yaml:"registries"`
}

func Default() Config {
	return Config{
		Engine:        "docker",
		Timeout:       10 * time.Second,
		PullTimeout:   10 * time.Minute,
		ParallelProbe: 6,
		Retries:       3,
		Registries:    map[string]RegistryOverride{},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	resolved, ok, err := resolvePath(path)
	if err != nil {
		return Config{}, err
	}
	if !ok {
		return cfg, nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return Config{}, err
	}

	var file fileConfig
	if err := yaml.Unmarshal(data, &file); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", resolved, err)
	}

	if file.Engine != "" {
		cfg.Engine = file.Engine
	}
	if file.Timeout != "" {
		cfg.Timeout, err = time.ParseDuration(file.Timeout)
		if err != nil {
			return Config{}, fmt.Errorf("parse timeout: %w", err)
		}
	}
	if file.PullTimeout != "" {
		cfg.PullTimeout, err = time.ParseDuration(file.PullTimeout)
		if err != nil {
			return Config{}, fmt.Errorf("parse pull_timeout: %w", err)
		}
	}
	if file.ParallelProbe != 0 {
		cfg.ParallelProbe = file.ParallelProbe
	}
	if file.Retries != 0 {
		cfg.Retries = file.Retries
	}
	cfg.DisableBuiltinMirrors = file.DisableBuiltinMirrors
	cfg.DisabledMirrors = file.DisabledMirrors
	cfg.Prefer = file.Prefer
	cfg.Exclude = file.Exclude
	if file.Registries != nil {
		cfg.Registries = file.Registries
	}
	cfg.LoadedFrom = resolved

	if !engine.IsSupported(cfg.Engine) {
		return Config{}, fmt.Errorf("unsupported engine %q; supported engines: %v", cfg.Engine, engine.Names())
	}
	if cfg.ParallelProbe < 1 {
		return Config{}, errors.New("parallel_probe must be at least 1")
	}
	if cfg.Retries < 1 {
		return Config{}, errors.New("retries must be at least 1")
	}
	return cfg, nil
}

func Paths() []string {
	paths := []string{}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "mip", "config.yaml"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths, filepath.Join(home, ".config", "mip", "config.yaml"))
	}
	return paths
}

func Profiles(cfg Config) []registry.Profile {
	profiles := []registry.Profile{}
	if !cfg.DisableBuiltinMirrors {
		profiles = registry.Builtins()
	}

	for name, override := range cfg.Registries {
		index := slices.IndexFunc(profiles, func(profile registry.Profile) bool {
			return profile.Name == name
		})
		if index < 0 {
			profiles = append(profiles, registry.Profile{Name: name})
			index = len(profiles) - 1
		}

		if len(override.Aliases) > 0 {
			profiles[index].Aliases = override.Aliases
		}
		if override.DefaultNamespace != "" {
			profiles[index].DefaultNamespace = override.DefaultNamespace
		}
		for _, mirror := range override.Mirrors {
			if mirror.Name == "" {
				mirror.Name = mirror.Host
			}
			if mirror.Priority == 0 {
				mirror.Priority = 100
			}
			mirror.EnabledByDefault = true
			profiles[index].Mirrors = append(profiles[index].Mirrors, mirror)
		}
	}

	for i := range profiles {
		profiles[i].Mirrors = filterMirrors(profiles[i].Mirrors, cfg.DisabledMirrors, cfg.Exclude)
		applyPreference(profiles[i].Mirrors, cfg.Prefer)
	}
	return profiles
}

func FindProfile(profiles []registry.Profile, registryName string) (registry.Profile, bool) {
	for _, profile := range profiles {
		if profile.Name == registryName {
			return profile, true
		}
		for _, alias := range profile.Aliases {
			if alias == registryName {
				return profile, true
			}
		}
	}
	return registry.Profile{}, false
}

func resolvePath(path string) (string, bool, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", false, err
		}
		return path, true, nil
	}

	for _, candidate := range Paths() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
	}
	return "", false, nil
}

func filterMirrors(mirrors []registry.Mirror, disabled []string, excluded []string) []registry.Mirror {
	filtered := make([]registry.Mirror, 0, len(mirrors))
	for _, mirror := range mirrors {
		if slices.Contains(disabled, mirror.Name) || slices.Contains(excluded, mirror.Name) || slices.Contains(excluded, mirror.Host) {
			continue
		}
		filtered = append(filtered, mirror)
	}
	return filtered
}

func applyPreference(mirrors []registry.Mirror, prefer []string) {
	for i := range mirrors {
		for _, preferred := range prefer {
			if mirrors[i].Name == preferred || mirrors[i].Host == preferred {
				mirrors[i].Priority += 1000
			}
		}
	}
}
