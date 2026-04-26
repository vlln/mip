package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/vlln/mip/configs"
	"github.com/vlln/mip/internal/engine"
	"github.com/vlln/mip/internal/registry"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Engine        string                      `json:"engine" yaml:"engine"`
	Timeout       time.Duration               `json:"timeout" yaml:"-"`
	PullTimeout   time.Duration               `json:"pull_timeout" yaml:"-"`
	ParallelProbe int                         `json:"parallel_probe" yaml:"parallel_probe"`
	Retries       int                         `json:"retries" yaml:"retries"`
	Prefer        []string                    `json:"prefer" yaml:"prefer"`
	Exclude       []string                    `json:"exclude" yaml:"exclude"`
	Registries    map[string]RegistryOverride `json:"registries" yaml:"registries"`
	LoadedFrom    string                      `json:"loaded_from,omitempty" yaml:"-"`
}

type RegistryOverride struct {
	Aliases          []string          `json:"aliases,omitempty" yaml:"aliases"`
	DefaultNamespace string            `json:"default_namespace,omitempty" yaml:"default_namespace"`
	Mirrors          []registry.Mirror `json:"mirrors,omitempty" yaml:"mirrors"`
}

type fileConfig struct {
	Engine        string                      `yaml:"engine"`
	Timeout       string                      `yaml:"timeout"`
	PullTimeout   string                      `yaml:"pull_timeout"`
	ParallelProbe int                         `yaml:"parallel_probe"`
	Retries       int                         `yaml:"retries"`
	Prefer        []string                    `yaml:"prefer"`
	Exclude       []string                    `yaml:"exclude"`
	Registries    map[string]RegistryOverride `yaml:"registries"`
}

func Default() Config {
	cfg := defaultBase()
	if err := mergeYAML(&cfg, configs.Official, "official config"); err != nil {
		panic(err)
	}
	return cfg
}

func defaultBase() Config {
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
	resolved, ok, err := resolvePath(path)
	if err != nil {
		return Config{}, err
	}
	if !ok {
		return Default(), nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return Config{}, err
	}

	cfg := defaultBase()
	if err := mergeYAML(&cfg, data, resolved); err != nil {
		return Config{}, err
	}
	cfg.LoadedFrom = resolved

	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func parseYAML(data []byte, label string) (fileConfig, error) {
	var file fileConfig
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fileConfig{}, fmt.Errorf("parse config %s: %w", label, err)
	}
	return file, nil
}

func mergeYAML(cfg *Config, data []byte, label string) error {
	file, err := parseYAML(data, label)
	if err != nil {
		return err
	}

	if file.Engine != "" {
		cfg.Engine = file.Engine
	}
	if file.Timeout != "" {
		timeout, err := time.ParseDuration(file.Timeout)
		if err != nil {
			return fmt.Errorf("parse timeout: %w", err)
		}
		cfg.Timeout = timeout
	}
	if file.PullTimeout != "" {
		pullTimeout, err := time.ParseDuration(file.PullTimeout)
		if err != nil {
			return fmt.Errorf("parse pull_timeout: %w", err)
		}
		cfg.PullTimeout = pullTimeout
	}
	if file.ParallelProbe != 0 {
		cfg.ParallelProbe = file.ParallelProbe
	}
	if file.Retries != 0 {
		cfg.Retries = file.Retries
	}
	cfg.Prefer = file.Prefer
	cfg.Exclude = file.Exclude
	if file.Registries != nil {
		if cfg.Registries == nil {
			cfg.Registries = map[string]RegistryOverride{}
		}
		for name, override := range file.Registries {
			current := cfg.Registries[name]
			if len(override.Aliases) > 0 {
				current.Aliases = override.Aliases
			}
			if override.DefaultNamespace != "" {
				current.DefaultNamespace = override.DefaultNamespace
			}
			current.Mirrors = append(current.Mirrors, override.Mirrors...)
			cfg.Registries[name] = current
		}
	}
	return nil
}

func validate(cfg Config) error {
	if !engine.IsSupported(cfg.Engine) {
		return fmt.Errorf("unsupported engine %q; supported engines: %v", cfg.Engine, engine.Names())
	}
	if cfg.ParallelProbe < 1 {
		return errors.New("parallel_probe must be at least 1")
	}
	if cfg.Retries < 1 {
		return errors.New("retries must be at least 1")
	}
	return nil
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
		profiles[index].Mirrors = append(profiles[index].Mirrors, normalizeMirrors(name, override.Mirrors)...)
	}

	for i := range profiles {
		profiles[i].Mirrors = filterMirrors(profiles[i].Mirrors, cfg.Exclude)
		applyPreference(profiles[i].Mirrors, cfg.Prefer)
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})
	return profiles
}

func normalizeMirrors(registryName string, mirrors []registry.Mirror) []registry.Mirror {
	normalized := make([]registry.Mirror, 0, len(mirrors))
	for index, mirror := range mirrors {
		if mirror.Name == "" {
			mirror.Name = mirror.Host
		}
		if mirror.Mode == "" {
			mirror.Mode = inferMode(registryName, mirror.Host)
		}
		if mirror.Priority == 0 {
			mirror.Priority = 1000 - index
		}
		normalized = append(normalized, mirror)
	}
	return normalized
}

func inferMode(registryName string, host string) registry.RewriteMode {
	trimmed := strings.TrimSuffix(host, "/")
	if trimmed == registryName || strings.HasSuffix(trimmed, "/"+registryName) {
		return registry.Prefix
	}
	return registry.HostReplace
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

func filterMirrors(mirrors []registry.Mirror, excluded []string) []registry.Mirror {
	filtered := make([]registry.Mirror, 0, len(mirrors))
	for _, mirror := range mirrors {
		if slices.Contains(excluded, mirror.Name) || slices.Contains(excluded, mirror.Host) {
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
