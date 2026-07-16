package giturl

import (
	"fmt"
	"net/url"
	"strings"
)

// Kind classifies what a GitHub URL is used for.
type Kind string

const (
	KindClone   Kind = "clone"
	KindRelease Kind = "release"
	KindRaw     Kind = "raw"
	KindArchive Kind = "archive"
	KindBlob    Kind = "blob"
	KindGist    Kind = "gist"
	KindUnknown Kind = "unknown"
)

// GitHubHosts returns all GitHub-controlled hosts that should be treated as
// aliases of github.com for mirror lookups.
func GitHubHosts() []string {
	return []string{
		"github.com",
		"raw.githubusercontent.com",
		"gist.githubusercontent.com",
		"api.github.com",
		"avatars.githubusercontent.com",
		"desktop.githubusercontent.com",
		"codeload.github.com",
	}
}

// CanonicalHost maps a GitHub host to its canonical form (github.com).
// Subdomains like raw.githubusercontent.com, api.github.com, etc. are all
// served by GitHub and share the same mirror configuration.
func CanonicalHost(host string) string {
	host = strings.ToLower(host)
	for _, h := range GitHubHosts() {
		if host == h {
			return "github.com"
		}
	}
	// Also handle *.github.com patterns
	if strings.HasSuffix(host, ".github.com") {
		return "github.com"
	}
	return host
}

// GitHubURL is a parsed GitHub resource URL.
type GitHubURL struct {
	Original string // original input string
	Host     string // e.g. github.com
	Owner    string // e.g. user or org
	Repo     string // e.g. myrepo (without .git suffix)
	Ref      string // tag, branch, or commit hash
	Path     string // full path after host (e.g. user/repo/releases/download/v1.0/file.tar.gz)
	Kind     Kind
}

// Parse takes a GitHub URL string and returns a parsed GitHubURL.
// It accepts:
//   - clone URLs: https://github.com/user/repo.git, https://github.com/user/repo
//   - release download URLs: https://github.com/user/repo/releases/download/v1.0/file.tar.gz
//   - raw URLs: https://raw.githubusercontent.com/user/repo/main/file.txt
//   - archive URLs: https://github.com/user/repo/archive/refs/tags/v1.0.tar.gz
func Parse(input string) (*GitHubURL, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty URL")
	}

	// Handle git@github.com:user/repo.git style SSH URLs
	if strings.HasPrefix(input, "git@") {
		return parseGitSSH(input)
	}

	// Handle raw.githubusercontent.com URLs
	if strings.Contains(input, "raw.githubusercontent.com") {
		return parseRaw(input)
	}

	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", input, err)
	}

	if u.Host == "" {
		// Try to handle git@github.com:user/repo.git style SSH URLs
		if strings.HasPrefix(input, "git@") {
			return parseGitSSH(input)
		}
		return nil, fmt.Errorf("invalid URL %q: missing host", input)
	}

	host := strings.ToLower(u.Host)
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, "/")

	// Remove .git suffix for clone URLs
	repoPath := strings.TrimSuffix(path, ".git")

	parts := strings.Split(repoPath, "/")

	kind := classifyPath(host, parts)
	result := &GitHubURL{
		Original: input,
		Host:     host,
		Path:     path,
		Kind:     kind,
	}

	switch kind {
	case KindClone, KindArchive, KindBlob:
		if len(parts) >= 2 {
			result.Owner = parts[0]
			result.Repo = parts[1]
		}
		if kind == KindBlob && len(parts) >= 4 {
			result.Ref = parts[3]
		}

	case KindRelease:
		// Standard path: owner/repo/releases/download/tag/...
		if len(parts) >= 6 && parts[2] == "releases" && parts[3] == "download" {
			result.Owner = parts[0]
			result.Repo = parts[1]
			result.Ref = parts[4]
		}

	case KindRaw, KindGist:
		if len(parts) >= 3 {
			result.Owner = parts[0]
			result.Repo = parts[1]
			result.Ref = parts[2]
		} else if len(parts) >= 2 && kind == KindGist {
			result.Owner = parts[0]
			result.Repo = parts[1]
		}
	}

	return result, nil
}

func parseRaw(input string) (*GitHubURL, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("invalid raw URL %q: %w", input, err)
	}

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")

	result := &GitHubURL{
		Original: input,
		Host:     strings.ToLower(u.Host),
		Path:     path,
		Kind:     KindRaw,
	}

	if len(parts) >= 3 {
		result.Owner = parts[0]
		result.Repo = parts[1]
		result.Ref = parts[2]
	}

	return result, nil
}

func parseGitSSH(input string) (*GitHubURL, error) {
	// git@github.com:user/repo.git
	rest := strings.TrimPrefix(input, "git@")
	before, after, ok := strings.Cut(rest, ":")
	if !ok {
		return nil, fmt.Errorf("invalid SSH URL %q", input)
	}

	path := strings.TrimSuffix(after, ".git")
	parts := strings.Split(path, "/")

	result := &GitHubURL{
		Original: input,
		Host:     strings.ToLower(before),
		Path:     path,
		Kind:     KindClone,
	}

	if len(parts) >= 2 {
		result.Owner = parts[0]
		result.Repo = parts[1]
	}

	return result, nil
}

func classifyPath(host string, parts []string) Kind {
	if len(parts) < 2 {
		return KindUnknown
	}

	// gist.githubusercontent.com/...
	if strings.Contains(host, "gist.githubusercontent.com") {
		return KindGist
	}

	// raw.githubusercontent.com/...
	if strings.Contains(host, "raw.githubusercontent.com") {
		return KindRaw
	}

	// releases/download/...
	if len(parts) >= 4 && parts[2] == "releases" && parts[3] == "download" {
		return KindRelease
	}

	// archive/...
	if len(parts) >= 3 && parts[2] == "archive" {
		return KindArchive
	}

	// blob/... (web UI file view, e.g. github.com/user/repo/blob/main/file.txt)
	if len(parts) >= 4 && parts[2] == "blob" {
		return KindBlob
	}

	// Default: clone URL (user/repo or user/repo.git)
	return KindClone
}

// CloneURL returns the canonical HTTPS clone URL for this repository.
func (g *GitHubURL) CloneURL() string {
	if g.Owner == "" || g.Repo == "" {
		return g.Original
	}
	return fmt.Sprintf("https://%s/%s/%s.git", g.Host, g.Owner, g.Repo)
}

// RepoPath returns the owner/repo portion of the URL.
func (g *GitHubURL) RepoPath() string {
	if g.Owner == "" || g.Repo == "" {
		return ""
	}
	return g.Owner + "/" + g.Repo
}

// String returns the original URL string.
func (g *GitHubURL) String() string {
	return g.Original
}