package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/vlln/mip/internal/probe"
	"github.com/vlln/mip/internal/rewrite"
)

type Store struct {
	Path    string                  `json:"-"`
	Mirrors map[string]MirrorHealth `json:"mirrors"`
}

type MirrorHealth struct {
	Image          string    `json:"image"`
	Mirror         string    `json:"mirror,omitempty"`
	Successes      int       `json:"successes"`
	Failures       int       `json:"failures"`
	LastOK         bool      `json:"last_ok"`
	LastStatusCode int       `json:"last_status_code,omitempty"`
	LastLatencyMS  int64     `json:"last_latency_ms"`
	LastError      string    `json:"last_error,omitempty"`
	LastDigest     string    `json:"last_digest,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func Load(path string) (Store, error) {
	resolved := path
	if resolved == "" {
		resolved = DefaultPath()
	}
	store := Store{Path: resolved, Mirrors: map[string]MirrorHealth{}}
	data, err := os.ReadFile(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, err
	}
	if len(data) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return Store{Path: resolved, Mirrors: map[string]MirrorHealth{}}, err
	}
	store.Path = resolved
	if store.Mirrors == nil {
		store.Mirrors = map[string]MirrorHealth{}
	}
	return store, nil
}

func (s Store) Save() error {
	if s.Path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, append(data, '\n'), 0o600)
}

func (s Store) Rank(candidates []rewrite.Candidate) {
	for i := range candidates {
		health, ok := s.Mirrors[candidates[i].Image]
		if !ok {
			continue
		}
		candidates[i].Priority += health.Score()
	}
}

func (s Store) Record(results []probe.Result) Store {
	if s.Mirrors == nil {
		s.Mirrors = map[string]MirrorHealth{}
	}
	now := time.Now().UTC()
	for _, result := range results {
		health := s.Mirrors[result.Image]
		health.Image = result.Image
		health.Mirror = result.Mirror
		health.LastOK = result.OK
		health.LastStatusCode = result.StatusCode
		health.LastLatencyMS = result.LatencyMS
		health.LastError = result.Error
		health.LastDigest = result.Digest
		health.UpdatedAt = now
		if result.OK {
			health.Successes++
		} else {
			health.Failures++
		}
		s.Mirrors[result.Image] = health
	}
	return s
}

func (h MirrorHealth) Score() int {
	score := 0
	score += h.Successes * 20
	score -= h.Failures * 80
	if h.LastOK {
		score += 50
	} else if !h.UpdatedAt.IsZero() {
		score -= 150
	}
	switch {
	case h.LastLatencyMS <= 0:
	case h.LastLatencyMS < 300:
		score += 40
	case h.LastLatencyMS < 1000:
		score += 20
	case h.LastLatencyMS > 5000:
		score -= 60
	case h.LastLatencyMS > 2000:
		score -= 30
	}
	return score
}

func DefaultPath() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "mip", "state.json")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".local", "state", "mip", "state.json")
	}
	return ""
}
