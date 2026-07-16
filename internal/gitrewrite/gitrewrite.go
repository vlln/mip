package gitrewrite

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vlln/mip/internal/gitmirror"
	"github.com/vlln/mip/internal/giturl"
)

// Candidate is a rewritten URL candidate for a mirror.
type Candidate struct {
	Original string              `json:"original"`
	URL      string              `json:"url"`
	Mirror   gitmirror.Mirror    `json:"mirror"`
	Mode     gitmirror.RewriteMode `json:"mode"`
	Priority int                 `json:"priority"`
}

// Candidates generates all mirror URL candidates for a given GitHub URL.
func Candidates(u *giturl.GitHubURL, profile gitmirror.Profile) []Candidate {
	candidates := make([]Candidate, 0, len(profile.Mirrors))
	for _, mirror := range profile.Mirrors {
		rewritten, ok := rewrite(u, mirror)
		if !ok {
			continue
		}
		candidates = append(candidates, Candidate{
			Original: u.Original,
			URL:      rewritten,
			Mirror:   mirror,
			Mode:     mirror.Mode,
			Priority: mirror.Priority,
		})
	}

	SortCandidates(candidates)
	return candidates
}

// SortCandidates sorts by priority descending.
func SortCandidates(candidates []Candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Priority > candidates[j].Priority
	})
}

func rewrite(u *giturl.GitHubURL, mirror gitmirror.Mirror) (string, bool) {
	host := strings.TrimSuffix(mirror.Host, "/")

	switch mirror.Mode {
	case gitmirror.HostReplace:
		return replaceHost(u.Original, u.Host, host), true
	case gitmirror.Prefix:
		return prefixURL(u.Original, host), true
	case gitmirror.PathTransform:
		return pathTransform(u.Original, u.Host, u.Path, host), true
	default:
		return "", false
	}
}

func replaceHost(original, sourceHost, mirrorHost string) string {
	// Replace the host in the URL
	result := original

	// https://github.com/... → https://mirror.example.com/...
	result = strings.Replace(result, "https://"+sourceHost, "https://"+mirrorHost, 1)
	result = strings.Replace(result, "http://"+sourceHost, "http://"+mirrorHost, 1)

	return result
}

func prefixURL(original, mirrorHost string) string {
	return fmt.Sprintf("https://%s/%s", mirrorHost, original)
}

func pathTransform(original, sourceHost, path, mirrorHost string) string {
	// https://github.com/user/repo → https://gitclone.com/github.com/user/repo
	return fmt.Sprintf("https://%s/%s/%s", mirrorHost, sourceHost, path)
}