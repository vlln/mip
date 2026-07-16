package gitconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/vlln/mip/configs"
	"github.com/vlln/mip/internal/gitmirror"
	"gopkg.in/yaml.v3"
)

// Config holds all gip configuration.
type Config struct {
	Timeout       time.Duration                `json:"timeout" yaml:"-"`
	ParallelProbe int                          `json:"parallel_probe" yaml:"parallel_probe"`
	Retries       int                          `json:"retries" yaml:"retries"`
	Prefer        []string                     `json:"prefer" yaml:"prefer"`
	Exclude       []string                     `json:"exclude" yaml:"exclude"`
	Mirrors       map[string]MirrorOverride    `json:"mirrors" yaml:"mirrors"`
	LoadedFrom    string                       `json:"loaded_from,omitempty" yaml:"-"`
}

// MirrorOverride allows users to add or override mirrors for a source host.
type MirrorOverride struct {
	Aliases []string `json:"aliases,omitempty" yaml:"aliases"`
	Mirrors []string `json:"mirrors,omitempty" yaml:"mirrors"`
}

type fileConfig struct {
	Prefer  []string                  `yaml:"prefer"`
	Exclude []string                  `yaml:"exclude"`
	Mirrors map[string]MirrorOverride `yaml:"mirrors"`
}

// Default returns the default configuration with embedded mirrors.
func Default() Config {
	cfg := defaultBase()
	if err := applyYAML(&cfg, configs.OfficialGIP, "official config"); err != nil {
		panic(err)
	}
	return cfg
}

func defaultBase() Config {
	return Config{
		Timeout:       10 * time.Second,
		ParallelProbe: 6,
		Retries:       3,
		Mirrors:       map[string]MirrorOverride{},
	}
}

// Load reads configuration from disk, falling back to defaults.
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
	if err := applyYAML(&cfg, data, resolved); err != nil {
		return Config{}, err
	}
	cfg.LoadedFrom = resolved

	return cfg, nil
}

func applyYAML(cfg *Config, data []byte, label string) error {
	file, err := parseFileConfig(data, label)
	if err != nil {
		return err
	}

	cfg.Prefer = file.Prefer
	cfg.Exclude = file.Exclude
	if file.Mirrors != nil {
		if cfg.Mirrors == nil {
			cfg.Mirrors = map[string]MirrorOverride{}
		}
		for name, override := range file.Mirrors {
			current := cfg.Mirrors[name]
			if len(override.Aliases) > 0 {
				current.Aliases = override.Aliases
			}
			current.Mirrors = append(current.Mirrors, override.Mirrors...)
			cfg.Mirrors[name] = current
		}
	}
	return nil
}

func parseFileConfig(data []byte, label string) (fileConfig, error) {
	var file fileConfig
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fileConfig{}, fmt.Errorf("parse config %s: %w", label, err)
	}
	return file, nil
}

// Paths returns the search paths for the gip config file.
func Paths() []string {
	paths := []string{}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "gip", "config.yaml"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths, filepath.Join(home, ".config", "gip", "config.yaml"))
	}
	return paths
}

// Profiles builds the list of mirror profiles from config.
func Profiles(cfg Config) []gitmirror.Profile {
	profiles := []gitmirror.Profile{}
	for name, override := range cfg.Mirrors {
		index := slices.IndexFunc(profiles, func(profile gitmirror.Profile) bool {
			return profile.Name == name
		})
		if index < 0 {
			profiles = append(profiles, gitmirror.Profile{Name: name})
			index = len(profiles) - 1
		}

		if len(override.Aliases) > 0 {
			profiles[index].Aliases = override.Aliases
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

func normalizeMirrors(hostName string, mirrors []string) []gitmirror.Mirror {
	normalized := make([]gitmirror.Mirror, 0, len(mirrors))
	for index, entry := range mirrors {
		host, mode := parseMirrorEntry(entry)
		if mode == "" {
			mode = inferMode(hostName, host)
		}
		normalized = append(normalized, gitmirror.Mirror{
			Name:     host,
			Host:     host,
			Mode:     mode,
			Priority: 1000 - index,
		})
	}
	return normalized
}

// parseMirrorEntry supports both "host" and "host:mode" formats.
// e.g. "ghproxy.com" or "ghproxy.com:prefix"
func parseMirrorEntry(entry string) (host string, mode gitmirror.RewriteMode) {
	// Check if there's a mode suffix
	for _, m := range []gitmirror.RewriteMode{
		gitmirror.HostReplace,
		gitmirror.Prefix,
		gitmirror.PathTransform,
	} {
		suffix := ":" + string(m)
		if len(entry) > len(suffix) && entry[len(entry)-len(suffix):] == suffix {
			return entry[:len(entry)-len(suffix)], m
		}
	}
	return entry, ""
}

func inferMode(hostName string, host string) gitmirror.RewriteMode {
	trimmed := trimSlash(host)

	// If the mirror host contains the original host name in its path,
	// it's likely a path-transform or prefix mode
	if trimmed != hostName && hasPathComponent(trimmed, hostName) {
		return gitmirror.Prefix
	}

	// If the mirror is a simple host (no path), default to host-replace
	if !hasPath(trimmed) {
		return gitmirror.HostReplace
	}

	// Has a path prefix — could be prefix or path-transform
	return gitmirror.Prefix
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func hasPathComponent(mirror, host string) bool {
	// Check if mirror URL contains the host as a path component
	// e.g. "ghproxy.com/https://github.com" contains "github.com"
	return mirror != host
}

func hasPath(mirror string) bool {
	for _, c := range mirror {
		if c == '/' {
			return true
		}
	}
	return false
}

func filterMirrors(mirrors []gitmirror.Mirror, excluded []string) []gitmirror.Mirror {
	filtered := make([]gitmirror.Mirror, 0, len(mirrors))
	for _, mirror := range mirrors {
		if slices.Contains(excluded, mirror.Name) || slices.Contains(excluded, mirror.Host) {
			continue
		}
		filtered = append(filtered, mirror)
	}
	return filtered
}

func applyPreference(mirrors []gitmirror.Mirror, prefer []string) {
	for i := range mirrors {
		for _, preferred := range prefer {
			if mirrors[i].Name == preferred || mirrors[i].Host == preferred {
				mirrors[i].Priority += 1000
			}
		}
	}
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
		}
	}
	return "", false, nil
}