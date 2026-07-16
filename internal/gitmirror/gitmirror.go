package gitmirror

// RewriteMode describes how to transform a source URL into a mirror URL.
type RewriteMode string

const (
	// HostReplace swaps the host portion of the URL.
	// e.g. https://github.com/user/repo → https://mirror.example.com/user/repo
	HostReplace RewriteMode = "host-replace"

	// Prefix prepends the mirror host as a path prefix to the original URL.
	// e.g. https://github.com/user/repo → https://ghproxy.com/https://github.com/user/repo
	Prefix RewriteMode = "prefix"

	// PathTransform rewrites the host into a path segment under the mirror host.
	// e.g. https://github.com/user/repo → https://gitclone.com/github.com/user/repo
	PathTransform RewriteMode = "path-transform"
)

// Mirror represents a single mirror endpoint for a GitHub-like service.
type Mirror struct {
	Name     string      `json:"name" yaml:"name"`
	Host     string      `json:"host" yaml:"host"`
	Mode     RewriteMode `json:"mode" yaml:"mode"`
	Priority int         `json:"priority" yaml:"priority"`
}

// Profile groups mirrors for a specific source host (e.g. github.com).
type Profile struct {
	Name    string   `json:"name" yaml:"name"`
	Aliases []string `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Mirrors []Mirror `json:"mirrors" yaml:"mirrors"`
}

// FindProfile locates a profile by name or alias.
func FindProfile(profiles []Profile, name string) (Profile, bool) {
	for _, profile := range profiles {
		if profile.Name == name {
			return profile, true
		}
		for _, alias := range profile.Aliases {
			if alias == name {
				return profile, true
			}
		}
	}
	return Profile{}, false
}