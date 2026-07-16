package gitops

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Git provides git operations through the CLI.
type Git struct {
	binary string
}

// New creates a Git instance.
func New() (*Git, error) {
	binary := "git"
	if _, err := exec.LookPath(binary); err != nil {
		return nil, fmt.Errorf("git not found in PATH")
	}
	return &Git{binary: binary}, nil
}

// CloneOptions controls clone behavior.
type CloneOptions struct {
	MirrorURL string
	TargetDir string
	Timeout   time.Duration
}

// Clone clones from a mirror URL, then sets the origin remote back to the original URL.
func (g *Git) Clone(ctx context.Context, originalURL string, opts CloneOptions) error {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Clone from mirror
	args := []string{"clone", opts.MirrorURL}
	if opts.TargetDir != "" {
		args = append(args, opts.TargetDir)
	}

	cmd := exec.CommandContext(ctx, g.binary, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone from mirror failed: %s: %w", string(output), err)
	}

	// Set origin remote back to original URL
	dir := opts.TargetDir
	if dir == "" {
		dir = repoDirFromURL(originalURL)
	}

	cmd = exec.CommandContext(ctx, g.binary, "-C", dir, "remote", "set-url", "origin", originalURL)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git remote set-url failed: %s: %w", string(output), err)
	}

	return nil
}

// Available checks if git is available and working.
func (g *Git) Available(ctx context.Context) error {
	if _, err := exec.LookPath(g.binary); err != nil {
		return fmt.Errorf("git not found in PATH")
	}
	cmd := exec.CommandContext(ctx, g.binary, "version")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git unavailable: %s: %w", string(output), err)
	}
	return nil
}

// Install sets up git insteadOf config to transparently route through a mirror.
func (g *Git) Install(ctx context.Context, originalHost, mirrorURL string) error {
	insteadOf := fmt.Sprintf("https://%s/", originalHost)
	args := []string{"config", "--global", fmt.Sprintf("url.%s.insteadOf", mirrorURL), insteadOf}

	cmd := exec.CommandContext(ctx, g.binary, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config insteadOf failed: %s: %w", string(output), err)
	}
	return nil
}

// Uninstall removes git insteadOf config for a specific mirror.
func (g *Git) Uninstall(ctx context.Context, mirrorURL string) error {
	key := fmt.Sprintf("url.%s.insteadOf", mirrorURL)
	args := []string{"config", "--global", "--unset", key}

	cmd := exec.CommandContext(ctx, g.binary, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config --unset failed: %s: %w", string(output), err)
	}
	return nil
}

// UninstallAll removes all insteadOf entries for a given host.
func (g *Git) UninstallAll(ctx context.Context, originalHost string) error {
	args := []string{"config", "--global", "--get-regexp", `^url\..*\.insteadOf`}
	cmd := exec.CommandContext(ctx, g.binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// No entries to remove is not an error
		return nil
	}

	hostSuffix := fmt.Sprintf("https://%s/", originalHost)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		if parts[1] == hostSuffix {
			key := strings.TrimSuffix(parts[0], ".insteadOf")
			_ = g.uninstallByKey(ctx, key)
		}
	}
	return nil
}

func (g *Git) uninstallByKey(ctx context.Context, key string) error {
	cmd := exec.CommandContext(ctx, g.binary, "config", "--global", "--unset", key)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git config --unset %s: %s: %w", key, string(output), err)
	}
	return nil
}

// ListManagedMirrors lists all mirrors that have been configured via install.
func (g *Git) ListManagedMirrors(ctx context.Context) ([]string, error) {
	args := []string{"config", "--global", "--get-regexp", `^url\..*\.insteadOf`}
	cmd := exec.CommandContext(ctx, g.binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// No entries
		return nil, nil
	}

	var mirrors []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	seen := map[string]bool{}
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ".insteadOf")
		key = strings.TrimPrefix(key, "url.")
		if !seen[key] {
			seen[key] = true
			mirrors = append(mirrors, key)
		}
	}
	return mirrors, nil
}

func repoDirFromURL(url string) string {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}