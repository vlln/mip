package registry

type RewriteMode string

const (
	HostReplace RewriteMode = "host-replace"
	Prefix      RewriteMode = "prefix"
)

type Mirror struct {
	Name             string      `json:"name"`
	Host             string      `json:"host"`
	Mode             RewriteMode `json:"mode"`
	Priority         int         `json:"priority"`
	Source           string      `json:"source,omitempty"`
	ReviewedAt       string      `json:"reviewed_at,omitempty"`
	EnabledByDefault bool        `json:"enabled_by_default"`
}

type Profile struct {
	Name             string   `json:"name"`
	Aliases          []string `json:"aliases,omitempty"`
	DefaultNamespace string   `json:"default_namespace,omitempty"`
	Mirrors          []Mirror `json:"mirrors"`
}

func Builtins() []Profile {
	source := "https://github.com/DaoCloud/public-image-mirror"
	reviewed := "2026-04-26"

	return []Profile{
		{
			Name:             "docker.io",
			Aliases:          []string{"index.docker.io", "registry-1.docker.io"},
			DefaultNamespace: "library",
			Mirrors: []Mirror{
				{Name: "daocloud-docker-host", Host: "docker.m.daocloud.io", Mode: HostReplace, Priority: 90, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
				{Name: "daocloud-docker-prefix", Host: "m.daocloud.io/docker.io", Mode: Prefix, Priority: 80, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
			},
		},
		{
			Name: "ghcr.io",
			Mirrors: []Mirror{
				{Name: "daocloud-ghcr-host", Host: "ghcr.m.daocloud.io", Mode: HostReplace, Priority: 90, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
				{Name: "daocloud-ghcr-prefix", Host: "m.daocloud.io/ghcr.io", Mode: Prefix, Priority: 80, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
			},
		},
		{
			Name: "quay.io",
			Mirrors: []Mirror{
				{Name: "daocloud-quay-host", Host: "quay.m.daocloud.io", Mode: HostReplace, Priority: 90, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
				{Name: "daocloud-quay-prefix", Host: "m.daocloud.io/quay.io", Mode: Prefix, Priority: 80, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
			},
		},
		{
			Name: "mcr.microsoft.com",
			Mirrors: []Mirror{
				{Name: "daocloud-mcr-host", Host: "mcr.m.daocloud.io", Mode: HostReplace, Priority: 90, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
				{Name: "daocloud-mcr-prefix", Host: "m.daocloud.io/mcr.microsoft.com", Mode: Prefix, Priority: 80, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
			},
		},
		{
			Name:    "registry.k8s.io",
			Aliases: []string{"k8s.gcr.io"},
			Mirrors: []Mirror{
				{Name: "daocloud-k8s-host", Host: "k8s.m.daocloud.io", Mode: HostReplace, Priority: 90, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
				{Name: "daocloud-k8s-prefix", Host: "m.daocloud.io/registry.k8s.io", Mode: Prefix, Priority: 80, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
			},
		},
		{
			Name: "gcr.io",
			Mirrors: []Mirror{
				{Name: "daocloud-gcr-host", Host: "gcr.m.daocloud.io", Mode: HostReplace, Priority: 90, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
				{Name: "daocloud-gcr-prefix", Host: "m.daocloud.io/gcr.io", Mode: Prefix, Priority: 80, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
			},
		},
		{
			Name: "docker.elastic.co",
			Mirrors: []Mirror{
				{Name: "daocloud-elastic-host", Host: "elastic.m.daocloud.io", Mode: HostReplace, Priority: 90, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
				{Name: "daocloud-elastic-prefix", Host: "m.daocloud.io/docker.elastic.co", Mode: Prefix, Priority: 80, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
			},
		},
		{
			Name: "nvcr.io",
			Mirrors: []Mirror{
				{Name: "daocloud-nvcr-host", Host: "nvcr.m.daocloud.io", Mode: HostReplace, Priority: 90, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
				{Name: "daocloud-nvcr-prefix", Host: "m.daocloud.io/nvcr.io", Mode: Prefix, Priority: 80, Source: source, ReviewedAt: reviewed, EnabledByDefault: true},
			},
		},
	}
}

func FindProfile(registryName string) (Profile, bool) {
	for _, profile := range Builtins() {
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
