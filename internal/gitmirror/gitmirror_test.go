package gitmirror

import (
	"testing"
)

func TestFindProfileByName(t *testing.T) {
	profiles := []Profile{
		{Name: "github.com", Mirrors: []Mirror{{Name: "ghproxy", Host: "ghproxy.com"}}},
		{Name: "gitlab.com", Mirrors: []Mirror{{Name: "glmirror", Host: "mirror.gitlab.com"}}},
	}

	profile, ok := FindProfile(profiles, "github.com")
	if !ok {
		t.Fatal("expected to find github.com profile")
	}
	if profile.Name != "github.com" {
		t.Fatalf("profile name = %s, want github.com", profile.Name)
	}
}

func TestFindProfileByAlias(t *testing.T) {
	profiles := []Profile{
		{Name: "github.com", Aliases: []string{"gh.io", "github"}, Mirrors: []Mirror{{Name: "ghproxy"}}},
	}

	profile, ok := FindProfile(profiles, "gh.io")
	if !ok {
		t.Fatal("expected to find profile by alias")
	}
	if profile.Name != "github.com" {
		t.Fatalf("profile name = %s, want github.com", profile.Name)
	}
}

func TestFindProfileMissing(t *testing.T) {
	profiles := []Profile{
		{Name: "github.com"},
	}

	_, ok := FindProfile(profiles, "bitbucket.org")
	if ok {
		t.Fatal("did not expect to find bitbucket.org profile")
	}
}