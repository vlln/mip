package registry

type RewriteMode string

const (
	HostReplace RewriteMode = "host-replace"
	Prefix      RewriteMode = "prefix"
)

type Mirror struct {
	Name     string      `json:"name" yaml:"name"`
	Host     string      `json:"host" yaml:"host"`
	Mode     RewriteMode `json:"mode" yaml:"mode"`
	Priority int         `json:"priority" yaml:"priority"`
}

type Profile struct {
	Name             string   `json:"name" yaml:"name"`
	Aliases          []string `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	DefaultNamespace string   `json:"default_namespace,omitempty" yaml:"default_namespace,omitempty"`
	Mirrors          []Mirror `json:"mirrors" yaml:"mirrors"`
}

func FindProfile(profiles []Profile, registryName string) (Profile, bool) {
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
	return Profile{}, false
}
