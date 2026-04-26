package ref

import (
	"errors"
	"fmt"
	"strings"
)

const (
	DefaultRegistry  = "docker.io"
	DefaultNamespace = "library"
	DefaultTag       = "latest"
)

type Reference struct {
	Registry   string
	Repository string
	Tag        string
	Digest     string
}

func Parse(input string) (Reference, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Reference{}, errors.New("empty image reference")
	}
	if strings.ContainsAny(input, " \t\n\r") {
		return Reference{}, fmt.Errorf("invalid image reference %q: whitespace is not allowed", input)
	}

	namePart, digest := splitDigest(input)
	if namePart == "" {
		return Reference{}, fmt.Errorf("invalid image reference %q: missing repository", input)
	}

	namePart, tag := splitTag(namePart)
	if tag == "" && digest == "" {
		tag = DefaultTag
	}

	registry, repository := splitRegistry(namePart)
	if repository == "" {
		return Reference{}, fmt.Errorf("invalid image reference %q: missing repository", input)
	}

	if registry == DefaultRegistry && !strings.Contains(repository, "/") {
		repository = DefaultNamespace + "/" + repository
	}

	return Reference{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
	}, nil
}

func (r Reference) String() string {
	base := r.Registry + "/" + r.Repository
	if r.Digest != "" {
		return base + "@" + r.Digest
	}
	if r.Tag != "" {
		return base + ":" + r.Tag
	}
	return base
}

func (r Reference) Familiar() string {
	if r.Registry == DefaultRegistry {
		repo := r.Repository
		if strings.HasPrefix(repo, DefaultNamespace+"/") {
			repo = strings.TrimPrefix(repo, DefaultNamespace+"/")
		}
		if r.Digest != "" {
			return repo + "@" + r.Digest
		}
		return repo + ":" + r.Tag
	}
	return r.String()
}

func splitDigest(input string) (namePart string, digest string) {
	before, after, ok := strings.Cut(input, "@")
	if !ok {
		return input, ""
	}
	return before, after
}

func splitTag(input string) (namePart string, tag string) {
	lastSlash := strings.LastIndex(input, "/")
	lastColon := strings.LastIndex(input, ":")
	if lastColon > lastSlash {
		return input[:lastColon], input[lastColon+1:]
	}
	return input, ""
}

func splitRegistry(input string) (registry string, repository string) {
	first, rest, ok := strings.Cut(input, "/")
	if !ok {
		return DefaultRegistry, input
	}
	if strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost" {
		return first, rest
	}
	return DefaultRegistry, input
}
