package gitrewrite

import (
	"testing"

	"github.com/vlln/mip/internal/gitmirror"
	"github.com/vlln/mip/internal/giturl"
)

func TestHostReplace(t *testing.T) {
	u, _ := giturl.Parse("https://github.com/user/repo.git")
	profile := gitmirror.Profile{
		Name: "github.com",
		Mirrors: []gitmirror.Mirror{
			{Name: "mirror.example.com", Host: "mirror.example.com", Mode: gitmirror.HostReplace, Priority: 100},
		},
	}

	candidates := Candidates(u, profile)
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	want := "https://mirror.example.com/user/repo.git"
	if candidates[0].URL != want {
		t.Fatalf("URL = %s, want %s", candidates[0].URL, want)
	}
}

func TestPrefix(t *testing.T) {
	u, _ := giturl.Parse("https://github.com/user/repo.git")
	profile := gitmirror.Profile{
		Name: "github.com",
		Mirrors: []gitmirror.Mirror{
			{Name: "ghproxy", Host: "ghproxy.com", Mode: gitmirror.Prefix, Priority: 200},
		},
	}

	candidates := Candidates(u, profile)
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	want := "https://ghproxy.com/https://github.com/user/repo.git"
	if candidates[0].URL != want {
		t.Fatalf("URL = %s, want %s", candidates[0].URL, want)
	}
}

func TestPathTransform(t *testing.T) {
	u, _ := giturl.Parse("https://github.com/user/repo.git")
	profile := gitmirror.Profile{
		Name: "github.com",
		Mirrors: []gitmirror.Mirror{
			{Name: "gitclone", Host: "gitclone.com", Mode: gitmirror.PathTransform, Priority: 150},
		},
	}

	candidates := Candidates(u, profile)
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	want := "https://gitclone.com/github.com/user/repo.git"
	if candidates[0].URL != want {
		t.Fatalf("URL = %s, want %s", candidates[0].URL, want)
	}
}

func TestCandidatesSortedByPriority(t *testing.T) {
	u, _ := giturl.Parse("https://github.com/user/repo.git")
	profile := gitmirror.Profile{
		Name: "github.com",
		Mirrors: []gitmirror.Mirror{
			{Name: "low", Host: "low.example.com", Mode: gitmirror.HostReplace, Priority: 10},
			{Name: "high", Host: "high.example.com", Mode: gitmirror.HostReplace, Priority: 100},
		},
	}

	candidates := Candidates(u, profile)
	if len(candidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(candidates))
	}
	if candidates[0].Mirror.Name != "high" {
		t.Fatalf("first candidate = %s, want high", candidates[0].Mirror.Name)
	}
}

func TestReleaseURLRewrite(t *testing.T) {
	u, _ := giturl.Parse("https://github.com/user/repo/releases/download/v1.0/file.tar.gz")
	profile := gitmirror.Profile{
		Name: "github.com",
		Mirrors: []gitmirror.Mirror{
			{Name: "ghproxy", Host: "ghproxy.com", Mode: gitmirror.Prefix, Priority: 100},
		},
	}

	candidates := Candidates(u, profile)
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	want := "https://ghproxy.com/https://github.com/user/repo/releases/download/v1.0/file.tar.gz"
	if candidates[0].URL != want {
		t.Fatalf("URL = %s, want %s", candidates[0].URL, want)
	}
}