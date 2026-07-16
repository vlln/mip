package giturl

import (
	"testing"
)

func TestParseCloneHTTPS(t *testing.T) {
	u, err := Parse("https://github.com/user/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != KindClone {
		t.Fatalf("kind = %s, want clone", u.Kind)
	}
	if u.Owner != "user" {
		t.Fatalf("owner = %s, want user", u.Owner)
	}
	if u.Repo != "repo" {
		t.Fatalf("repo = %s, want repo", u.Repo)
	}
	if u.Host != "github.com" {
		t.Fatalf("host = %s, want github.com", u.Host)
	}
}

func TestParseCloneHTTPSNoGit(t *testing.T) {
	u, err := Parse("https://github.com/user/repo")
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != KindClone {
		t.Fatalf("kind = %s, want clone", u.Kind)
	}
	if u.Owner != "user" || u.Repo != "repo" {
		t.Fatalf("owner/repo = %s/%s, want user/repo", u.Owner, u.Repo)
	}
}

func TestParseCloneSSH(t *testing.T) {
	u, err := Parse("git@github.com:vlln/mip.git")
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != KindClone {
		t.Fatalf("kind = %s, want clone", u.Kind)
	}
	if u.Owner != "vlln" || u.Repo != "mip" {
		t.Fatalf("owner/repo = %s/%s, want vlln/mip", u.Owner, u.Repo)
	}
}

func TestParseRelease(t *testing.T) {
	u, err := Parse("https://github.com/user/repo/releases/download/v1.0/binary.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != KindRelease {
		t.Fatalf("kind = %s, want release", u.Kind)
	}
	if u.Owner != "user" || u.Repo != "repo" {
		t.Fatalf("owner/repo = %s/%s, want user/repo", u.Owner, u.Repo)
	}
	if u.Ref != "v1.0" {
		t.Fatalf("ref = %s, want v1.0", u.Ref)
	}
}

func TestParseRaw(t *testing.T) {
	u, err := Parse("https://raw.githubusercontent.com/user/repo/main/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != KindRaw {
		t.Fatalf("kind = %s, want raw", u.Kind)
	}
	if u.Owner != "user" || u.Repo != "repo" {
		t.Fatalf("owner/repo = %s/%s, want user/repo", u.Owner, u.Repo)
	}
	if u.Ref != "main" {
		t.Fatalf("ref = %s, want main", u.Ref)
	}
}

func TestParseArchive(t *testing.T) {
	u, err := Parse("https://github.com/user/repo/archive/refs/tags/v1.0.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != KindArchive {
		t.Fatalf("kind = %s, want archive", u.Kind)
	}
	if u.Owner != "user" || u.Repo != "repo" {
		t.Fatalf("owner/repo = %s/%s, want user/repo", u.Owner, u.Repo)
	}
}

func TestParseEmpty(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestCloneURL(t *testing.T) {
	u, _ := Parse("https://github.com/user/repo")
	if got := u.CloneURL(); got != "https://github.com/user/repo.git" {
		t.Fatalf("CloneURL = %s, want https://github.com/user/repo.git", got)
	}
}

func TestRepoPath(t *testing.T) {
	u, _ := Parse("https://github.com/user/repo.git")
	if got := u.RepoPath(); got != "user/repo" {
		t.Fatalf("RepoPath = %s, want user/repo", got)
	}
}

func TestParseReleaseDeepPath(t *testing.T) {
	u, err := Parse("https://github.com/kubernetes/kubernetes/releases/download/v1.30.0/kubernetes.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != KindRelease {
		t.Fatalf("kind = %s, want release", u.Kind)
	}
	if u.Owner != "kubernetes" || u.Repo != "kubernetes" {
		t.Fatalf("owner/repo = %s/%s", u.Owner, u.Repo)
	}
	if u.Ref != "v1.30.0" {
		t.Fatalf("ref = %s, want v1.30.0", u.Ref)
	}
}

func TestParseBlob(t *testing.T) {
	u, err := Parse("https://github.com/WJQSERVER-STUDIO/ghproxy/blob/main/config/config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != KindBlob {
		t.Fatalf("kind = %s, want blob", u.Kind)
	}
	if u.Owner != "WJQSERVER-STUDIO" || u.Repo != "ghproxy" {
		t.Fatalf("owner/repo = %s/%s", u.Owner, u.Repo)
	}
	if u.Ref != "main" {
		t.Fatalf("ref = %s, want main", u.Ref)
	}
}

func TestParseGist(t *testing.T) {
	u, err := Parse("https://gist.githubusercontent.com/oopsunix/2dbf20f64984773da6740d1d1cf7c2d4/raw/github-blacklist")
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != KindGist {
		t.Fatalf("kind = %s, want gist", u.Kind)
	}
	if u.Owner != "oopsunix" {
		t.Fatalf("owner = %s, want oopsunix", u.Owner)
	}
}

func TestParseAPI(t *testing.T) {
	u, err := Parse("https://api.github.com/repos/umami-software/umami")
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "api.github.com" {
		t.Fatalf("host = %s, want api.github.com", u.Host)
	}
}

func TestParseAvatars(t *testing.T) {
	u, err := Parse("https://avatars.githubusercontent.com/u/10000?v=4")
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "avatars.githubusercontent.com" {
		t.Fatalf("host = %s", u.Host)
	}
}

func TestParseDesktop(t *testing.T) {
	u, err := Parse("https://desktop.githubusercontent.com/releases/3.6.2-beta1-f62b1c7a/GitHubDesktop-x64.zip")
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "desktop.githubusercontent.com" {
		t.Fatalf("host = %s", u.Host)
	}
}

func TestCanonicalHost(t *testing.T) {
	tests := []struct{ input, want string }{
		{"github.com", "github.com"},
		{"raw.githubusercontent.com", "github.com"},
		{"gist.githubusercontent.com", "github.com"},
		{"api.github.com", "github.com"},
		{"avatars.githubusercontent.com", "github.com"},
		{"desktop.githubusercontent.com", "github.com"},
		{"codeload.github.com", "github.com"},
		{"gitlab.com", "gitlab.com"},
	}
	for _, tt := range tests {
		got := CanonicalHost(tt.input)
		if got != tt.want {
			t.Errorf("CanonicalHost(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGitHubHosts(t *testing.T) {
	hosts := GitHubHosts()
	if len(hosts) < 7 {
		t.Fatalf("expected at least 7 hosts, got %d", len(hosts))
	}
	for _, h := range hosts {
		if CanonicalHost(h) != "github.com" {
			t.Errorf("CanonicalHost(%q) = %q, want github.com", h, CanonicalHost(h))
		}
	}
}