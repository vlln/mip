package rewrite

import (
	"sort"
	"strings"

	"github.com/vlln/mip/internal/ref"
	"github.com/vlln/mip/internal/registry"
)

type Candidate struct {
	Original string               `json:"original"`
	Image    string               `json:"image"`
	Registry string               `json:"registry"`
	Mirror   registry.Mirror      `json:"mirror"`
	Mode     registry.RewriteMode `json:"mode"`
	Priority int                  `json:"priority"`
}

func Candidates(image ref.Reference, profile registry.Profile) []Candidate {
	candidates := make([]Candidate, 0, len(profile.Mirrors))
	for _, mirror := range profile.Mirrors {
		rewritten, ok := rewrite(image, profile.Name, mirror)
		if !ok {
			continue
		}
		candidates = append(candidates, Candidate{
			Original: image.String(),
			Image:    rewritten,
			Registry: profile.Name,
			Mirror:   mirror,
			Mode:     mirror.Mode,
			Priority: mirror.Priority,
		})
	}

	SortCandidates(candidates)

	return candidates
}

func SortCandidates(candidates []Candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Priority > candidates[j].Priority
	})
}

func rewrite(image ref.Reference, canonicalRegistry string, mirror registry.Mirror) (string, bool) {
	switch mirror.Mode {
	case registry.HostReplace:
		return withHost(mirror.Host, image.Repository, image.Tag, image.Digest), true
	case registry.Prefix:
		repo := strings.TrimSuffix(mirror.Host, "/") + "/" + canonicalRegistry + "/" + image.Repository
		if strings.HasSuffix(mirror.Host, canonicalRegistry) {
			repo = strings.TrimSuffix(mirror.Host, "/") + "/" + image.Repository
		}
		return withTagOrDigest(repo, image.Tag, image.Digest), true
	default:
		return "", false
	}
}

func withHost(host, repository, tag, digest string) string {
	return withTagOrDigest(strings.TrimSuffix(host, "/")+"/"+repository, tag, digest)
}

func withTagOrDigest(base, tag, digest string) string {
	if digest != "" {
		return base + "@" + digest
	}
	if tag != "" {
		return base + ":" + tag
	}
	return base
}
