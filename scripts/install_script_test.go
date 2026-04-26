package scripts

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallScriptResolvesLatestReleaseAsset(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("install script fixture is linux/amd64-specific")
	}

	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	fixture := newInstallFixture(t, tmp)

	cmd := exec.Command("sh", "./install.sh")
	cmd.Env = append(os.Environ(),
		"PATH="+fixture.fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_RELEASE_DIR="+fixture.releaseDir,
		"MIP_REPO=example/mip",
		"MIP_BINDIR="+binDir,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install script failed: %s: %v", output, err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "mip")); err != nil {
		t.Fatalf("installed mip missing: %v", err)
	}
}

func TestInstallScriptFallsBackToUserLocalBinWhenDefaultIsUnwritable(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("install script fixture is linux/amd64-specific")
	}
	if os.Geteuid() == 0 {
		t.Skip("unwritable directory check requires a non-root user")
	}

	tmp := t.TempDir()
	fixture := newInstallFixture(t, tmp)
	defaultBin := filepath.Join(tmp, "root-bin")
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(defaultBin, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(defaultBin, 0o755)
	})
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sh", "./install.sh")
	cmd.Env = append(os.Environ(),
		"PATH="+fixture.fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_RELEASE_DIR="+fixture.releaseDir,
		"MIP_REPO=example/mip",
		"MIP_DEFAULT_BINDIR="+defaultBin,
		"HOME="+home,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install script failed: %s: %v", output, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "bin", "mip")); err != nil {
		t.Fatalf("installed mip missing in fallback dir: %v", err)
	}
}

type installFixture struct {
	releaseDir string
	fakeBin    string
}

func newInstallFixture(t *testing.T, tmp string) installFixture {
	t.Helper()

	releaseDir := filepath.Join(tmp, "release")
	fakeBin := filepath.Join(tmp, "fake-bin")
	work := filepath.Join(tmp, "work", "mip_1.2.3_linux_amd64")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}

	mipPath := filepath.Join(work, "mip")
	if err := os.WriteFile(mipPath, []byte("#!/bin/sh\necho mip 1.2.3\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(releaseDir, "mip_1.2.3_linux_amd64.tar.gz")
	tar := exec.Command("tar", "-C", filepath.Join(tmp, "work"), "-czf", archive, "mip_1.2.3_linux_amd64")
	if output, err := tar.CombinedOutput(); err != nil {
		t.Fatalf("create archive: %s: %v", output, err)
	}

	archiveData, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archiveData)
	checksums := fmt.Sprintf("%x  mip_1.2.3_linux_amd64.tar.gz\n", sum)
	if err := os.WriteFile(filepath.Join(releaseDir, "checksums.txt"), []byte(checksums), 0o644); err != nil {
		t.Fatal(err)
	}

	curlPath := filepath.Join(fakeBin, "curl")
	curlScript := `#!/bin/sh
set -eu
for arg do url="$arg"; done
case "$url" in
  */releases/latest)
    printf '{"tag_name":"v1.2.3"}'
    ;;
  */mip_1.2.3_linux_amd64.tar.gz)
    cat "$FAKE_RELEASE_DIR/mip_1.2.3_linux_amd64.tar.gz"
    ;;
  */checksums.txt)
    cat "$FAKE_RELEASE_DIR/checksums.txt"
    ;;
  *)
    echo "unexpected URL: $url" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(curlPath, []byte(curlScript), 0o755); err != nil {
		t.Fatal(err)
	}

	return installFixture{
		releaseDir: releaseDir,
		fakeBin:    fakeBin,
	}
}
