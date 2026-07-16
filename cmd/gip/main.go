package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vlln/mip/internal/gipcompletion"
	"github.com/vlln/mip/internal/gitconfig"
	"github.com/vlln/mip/internal/gitmirror"
	"github.com/vlln/mip/internal/gitops"
	"github.com/vlln/mip/internal/gitprobe"
	"github.com/vlln/mip/internal/gitrewrite"
	"github.com/vlln/mip/internal/gitstate"
	"github.com/vlln/mip/internal/giturl"
	"github.com/vlln/mip/internal/output"
	"github.com/vlln/mip/internal/version"
)

const (
	exitOK             = 0
	exitGeneralError   = 1
	exitInvalidURL     = 2
	exitNoUsableMirror = 3
	exitGitError       = 4
	exitDownloadFailed = 5
	exitConfigError    = 9
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return exitGeneralError
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return exitOK
	case "version":
		return runVersion(args[1:])
	case "rewrite":
		return runRewrite(args[1:])
	case "mirrors":
		return runMirrors(args[1:])
	case "config":
		return runConfig(args[1:])
	case "completion":
		return runCompletion(args[1:])
	case "probe":
		return runProbe(args[1:])
	case "clone":
		return runClone(args[1:])
	case "install":
		return runInstall(args[1:])
	case "uninstall":
		return runUninstall(args[1:])
	case "get":
		return runGet(args[1:])
	default:
		if looksLikeURL(args[0]) {
			return runClone(args)
		}
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		printUsage(os.Stderr)
		return exitGeneralError
	}
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "git@")
}

func runVersion(args []string) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(moveBoolFlagsFirst(args, map[string]bool{"--json": true})); err != nil {
		return exitGeneralError
	}

	info := version.Get()
	if *jsonOut {
		_ = output.JSON(os.Stdout, info)
		return exitOK
	}
	fmt.Fprintf(os.Stdout, "gip %s\n", info.Version)
	return exitOK
}

func runCompletion(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: gip completion bash|zsh|fish")
		return exitGeneralError
	}
	script, err := gipcompletion.Script(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitGeneralError
	}
	fmt.Fprint(os.Stdout, script)
	return exitOK
}

func runRewrite(args []string) int {
	fs := flag.NewFlagSet("rewrite", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	all := fs.Bool("all", false, "print all candidates")
	plain := fs.Bool("plain", false, "print only URLs")
	jsonOut := fs.Bool("json", false, "emit JSON")
	configPath := fs.String("config", configPathArg(args), "config file path")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--all": true, "--plain": true, "--json": true,
	}, map[string]bool{
		"--config": true,
	})); err != nil {
		return exitGeneralError
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gip rewrite URL [--all] [--plain] [--json]")
		return exitGeneralError
	}

	cfg, err := gitconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}

	return rewriteURL(cfg, fs.Arg(0), *all, *plain, *jsonOut)
}

func rewriteURL(cfg gitconfig.Config, input string, all, plain, jsonOut bool) int {
	u, err := giturl.Parse(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid URL: %v\n", err)
		return exitInvalidURL
	}

	profiles := gitconfig.Profiles(cfg)
	profile, ok := gitmirror.FindProfile(profiles, u.Host)
	if !ok {
		if jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"url":        u.Original,
				"candidates": []gitrewrite.Candidate{},
			})
		} else if plain {
			fmt.Fprintln(os.Stdout, u.Original)
		} else {
			fmt.Fprintf(os.Stderr, "no configured mirrors for host %q\n", u.Host)
			fmt.Fprintln(os.Stdout, u.Original)
		}
		return exitOK
	}

	candidates := gitrewrite.Candidates(u, profile)
	if !all && len(candidates) > 1 {
		candidates = candidates[:1]
	}

	if jsonOut {
		_ = output.JSON(os.Stdout, map[string]any{
			"url":        u.Original,
			"host":       profile.Name,
			"candidates": candidates,
		})
		return exitOK
	}

	if plain {
		for _, c := range candidates {
			fmt.Fprintln(os.Stdout, c.URL)
		}
		return exitOK
	}

	fmt.Fprintf(os.Stdout, "url: %s\n", u.Original)
	fmt.Fprintf(os.Stdout, "host: %s\n", profile.Name)
	for i, c := range candidates {
		fmt.Fprintf(os.Stdout, "candidate[%d]: %s\n", i, c.URL)
		fmt.Fprintf(os.Stdout, "  mirror: %s\n", c.Mirror.Name)
		fmt.Fprintf(os.Stdout, "  mode: %s\n", c.Mode)
	}
	return exitOK
}

func runProbe(args []string) int {
	fs := flag.NewFlagSet("probe", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	configPath := fs.String("config", configPathArg(args), "config file path")
	cfg, err := gitconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}
	timeout := fs.Duration("timeout", cfg.Timeout, "per candidate timeout")
	concurrency := fs.Int("concurrency", cfg.ParallelProbe, "maximum concurrent probes")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--json": true,
	}, map[string]bool{
		"--config": true, "--timeout": true, "--concurrency": true,
	})); err != nil {
		return exitGeneralError
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gip probe URL [--timeout DURATION] [--concurrency N] [--json]")
		return exitGeneralError
	}

	u, err := giturl.Parse(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid URL: %v\n", err)
		return exitInvalidURL
	}

	profiles := gitconfig.Profiles(cfg)
	store := loadState()
	_, results, code := selectCandidate(context.Background(), profiles, store, u, *timeout, *concurrency)
	saveState(store.Record(results))

	if *jsonOut {
		payload := map[string]any{
			"url":     u.Original,
			"results": results,
		}
		_ = output.JSON(os.Stdout, payload)
		return exitOK
	}

	printProbeResults(results)
	return code
}

func runClone(args []string) int {
	fs := flag.NewFlagSet("clone", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	dryRun := fs.Bool("dry-run", false, "show selected candidate without cloning")
	configPath := fs.String("config", configPathArg(args), "config file path")
	cfg, err := gitconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}
	timeout := fs.Duration("timeout", cfg.Timeout, "per candidate probe timeout")
	concurrency := fs.Int("concurrency", cfg.ParallelProbe, "maximum concurrent probes")
	dir := fs.String("dir", "", "target directory")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--json": true, "--dry-run": true,
	}, map[string]bool{
		"--config": true, "--timeout": true, "--concurrency": true, "--dir": true,
	})); err != nil {
		return exitGeneralError
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gip clone URL [--dir DIR] [--dry-run] [--json]")
		return exitGeneralError
	}

	return cloneOne(cfg, fs.Arg(0), *timeout, *concurrency, *dir, *dryRun, *jsonOut)
}

func cloneOne(cfg gitconfig.Config, input string, timeout time.Duration, concurrency int, dir string, dryRun, jsonOut bool) int {
	u, err := giturl.Parse(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid URL: %v\n", err)
		return exitInvalidURL
	}

	profiles := gitconfig.Profiles(cfg)
	store := loadState()
	selected, results, code := selectCandidate(context.Background(), profiles, store, u, timeout, concurrency)
	store = store.Record(results)
	saveState(store)

	if code != exitOK {
		if jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"url":     u.Original,
				"status":  "no_usable_mirror",
				"results": results,
			})
		} else {
			printProbeResults(results)
		}
		return code
	}

	if !jsonOut {
		fmt.Fprintf(os.Stderr, "probing %d candidates → %s (%dms)\n", len(results), selected.Mirror, selected.LatencyMS)
	}

	if dryRun {
		if jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"url":      u.Original,
				"selected": selected.URL,
				"mirror":   selected.Mirror,
				"status":   "dry_run",
				"results":  results,
			})
		} else {
			fmt.Fprintf(os.Stdout, "url: %s\n", u.Original)
			fmt.Fprintf(os.Stdout, "selected: %s\n", selected.URL)
			fmt.Fprintf(os.Stdout, "mirror: %s\n", selected.Mirror)
			fmt.Fprintln(os.Stdout, "status: dry-run")
		}
		return exitOK
	}

	g, err := gitops.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitGitError
	}

	if err := g.Available(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitGitError
	}

	start := time.Now()
	cloneErr := g.Clone(context.Background(), u.Original, gitops.CloneOptions{
		MirrorURL: selected.URL,
		TargetDir: dir,
		Timeout:   10 * time.Minute,
	})
	elapsed := time.Since(start).Milliseconds()

	if cloneErr != nil {
		if jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"url":        u.Original,
				"selected":   selected.URL,
				"mirror":     selected.Mirror,
				"status":     "clone_failed",
				"error":      cloneErr.Error(),
				"elapsed_ms": elapsed,
			})
		} else {
			fmt.Fprintf(os.Stdout, "url: %s\n", u.Original)
			fmt.Fprintf(os.Stdout, "selected: %s\n", selected.URL)
			fmt.Fprintf(os.Stderr, "clone failed: %v\n", cloneErr)
		}
		return exitGitError
	}

	if jsonOut {
		_ = output.JSON(os.Stdout, map[string]any{
			"url":        u.Original,
			"selected":   selected.URL,
			"mirror":     selected.Mirror,
			"status":     "cloned",
			"elapsed_ms": elapsed,
		})
	} else {
		fmt.Fprintf(os.Stdout, "url: %s\n", u.Original)
		fmt.Fprintf(os.Stdout, "selected: %s\n", selected.URL)
		fmt.Fprintf(os.Stdout, "mirror: %s\n", selected.Mirror)
		fmt.Fprintln(os.Stdout, "status: cloned")
		fmt.Fprintf(os.Stdout, "elapsed: %.1fs\n", float64(elapsed)/1000)
	}
	return exitOK
}

func runInstall(args []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	configPath := fs.String("config", configPathArg(args), "config file path")
	cfg, err := gitconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}
	timeout := fs.Duration("timeout", cfg.Timeout, "per candidate probe timeout")
	concurrency := fs.Int("concurrency", cfg.ParallelProbe, "maximum concurrent probes")
	host := fs.String("host", "github.com", "source host to configure")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--json": true,
	}, map[string]bool{
		"--config": true, "--timeout": true, "--concurrency": true, "--host": true,
	})); err != nil {
		return exitGeneralError
	}

	// Create a synthetic URL for probing
	testURL := fmt.Sprintf("https://%s/gip/test", *host)
	u, err := giturl.Parse(testURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid URL: %v\n", err)
		return exitInvalidURL
	}

	profiles := gitconfig.Profiles(cfg)
	store := loadState()
	selected, results, code := selectCandidate(context.Background(), profiles, store, u, *timeout, *concurrency)
	store = store.Record(results)
	saveState(store)

	if code != exitOK {
		if *jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"host":    *host,
				"status":  "no_usable_mirror",
				"results": results,
			})
		} else {
			printProbeResults(results)
		}
		return code
	}

	g, err := gitops.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitGitError
	}

	mirrorBase := buildMirrorBase(selected.URL, *host)

	if err := g.Install(context.Background(), *host, mirrorBase); err != nil {
		fmt.Fprintf(os.Stderr, "install failed: %v\n", err)
		return exitGitError
	}

	if *jsonOut {
		_ = output.JSON(os.Stdout, map[string]any{
			"host":   *host,
			"mirror": selected.Mirror,
			"url":    mirrorBase,
			"status": "installed",
		})
	} else {
		fmt.Fprintf(os.Stdout, "host: %s\n", *host)
		fmt.Fprintf(os.Stdout, "mirror: %s\n", selected.Mirror)
		fmt.Fprintf(os.Stdout, "url: %s\n", mirrorBase)
		fmt.Fprintln(os.Stdout, "status: installed")
		fmt.Fprintf(os.Stderr, "git config --global url.%s.insteadOf https://%s/\n", mirrorBase, *host)
	}
	return exitOK
}

func buildMirrorBase(mirrorURL, host string) string {
	u := mirrorURL
	u = strings.TrimSuffix(u, "/gip/test")

	if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
		u = "https://" + u
	}

	hostPrefix := host + "/"
	if idx := strings.Index(u, hostPrefix); idx > 8 {
		return u[:idx+len(hostPrefix)]
	}

	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	return u
}

func runUninstall(args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	host := fs.String("host", "github.com", "source host to remove")
	if err := fs.Parse(moveFlagsFirst(args, nil, map[string]bool{
		"--host": true,
	})); err != nil {
		return exitGeneralError
	}

	g, err := gitops.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitGitError
	}

	if err := g.UninstallAll(context.Background(), *host); err != nil {
		fmt.Fprintf(os.Stderr, "uninstall failed: %v\n", err)
		return exitGitError
	}

	fmt.Fprintf(os.Stdout, "status: uninstalled\n")
	fmt.Fprintf(os.Stdout, "host: %s\n", *host)
	return exitOK
}

func runGet(args []string) int {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	dryRun := fs.Bool("dry-run", false, "show selected candidate without downloading")
	output := fs.String("output", "", "output file path (default: basename of URL)")
	configPath := fs.String("config", configPathArg(args), "config file path")
	cfg, err := gitconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}
	timeout := fs.Duration("timeout", cfg.Timeout, "per candidate probe timeout")
	concurrency := fs.Int("concurrency", cfg.ParallelProbe, "maximum concurrent probes")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--json": true, "--dry-run": true,
	}, map[string]bool{
		"--config": true, "--output": true, "--timeout": true, "--concurrency": true,
	})); err != nil {
		return exitGeneralError
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gip get URL [--output PATH] [--dry-run] [--json]")
		return exitGeneralError
	}

	return getOne(cfg, fs.Arg(0), *timeout, *concurrency, *output, *dryRun, *jsonOut)
}

func getOne(cfg gitconfig.Config, input string, timeout time.Duration, concurrency int, outputPath string, dryRun, jsonOut bool) int {
	u, err := giturl.Parse(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid URL: %v\n", err)
		return exitInvalidURL
	}

	profiles := gitconfig.Profiles(cfg)
	store := loadState()
	selected, results, code := selectCandidate(context.Background(), profiles, store, u, timeout, concurrency)
	store = store.Record(results)
	saveState(store)

	if code != exitOK {
		if jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"url":     u.Original,
				"status":  "no_usable_mirror",
				"results": results,
			})
		} else {
			printProbeResults(results)
		}
		return code
	}

	if !jsonOut {
		fmt.Fprintf(os.Stderr, "probing %d candidates → %s (%dms)\n", len(results), selected.Mirror, selected.LatencyMS)
	}

	if dryRun {
		if jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"url":      u.Original,
				"selected": selected.URL,
				"mirror":   selected.Mirror,
				"status":   "dry_run",
				"results":  results,
			})
		} else {
			fmt.Fprintf(os.Stdout, "url: %s\n", u.Original)
			fmt.Fprintf(os.Stdout, "selected: %s\n", selected.URL)
			fmt.Fprintf(os.Stdout, "mirror: %s\n", selected.Mirror)
			fmt.Fprintln(os.Stdout, "status: dry-run")
		}
		return exitOK
	}

	destPath := outputPath
	if destPath == "" {
		destPath = basenameFromURL(u.Original)
	}

	start := time.Now()
	dlErr := downloadFile(context.Background(), selected.URL, destPath)
	elapsed := time.Since(start).Milliseconds()

	if dlErr != nil {
		if jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"url":        u.Original,
				"selected":   selected.URL,
				"mirror":     selected.Mirror,
				"status":     "download_failed",
				"error":      dlErr.Error(),
				"elapsed_ms": elapsed,
			})
		} else {
			fmt.Fprintf(os.Stderr, "download failed: %v\n", dlErr)
		}
		return exitDownloadFailed
	}

	if jsonOut {
		_ = output.JSON(os.Stdout, map[string]any{
			"url":        u.Original,
			"selected":   selected.URL,
			"mirror":     selected.Mirror,
			"dest":       destPath,
			"status":     "downloaded",
			"elapsed_ms": elapsed,
		})
	} else {
		fmt.Fprintf(os.Stdout, "url: %s\n", u.Original)
		fmt.Fprintf(os.Stdout, "selected: %s\n", selected.URL)
		fmt.Fprintf(os.Stdout, "mirror: %s\n", selected.Mirror)
		fmt.Fprintf(os.Stdout, "dest: %s\n", destPath)
		fmt.Fprintln(os.Stdout, "status: downloaded")
		fmt.Fprintf(os.Stdout, "elapsed: %.1fs\n", float64(elapsed)/1000)
	}
	return exitOK
}

func basenameFromURL(rawURL string) string {
	if idx := strings.Index(rawURL, "?"); idx > 0 {
		rawURL = rawURL[:idx]
	}
	if idx := strings.Index(rawURL, "#"); idx > 0 {
		rawURL = rawURL[:idx]
	}
	parts := strings.Split(rawURL, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}
	if len(parts) > 1 {
		return parts[len(parts)-2]
	}
	return "download"
}

func downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "gip/0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func runMirrors(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gip mirrors list [--host HOST] [--json]")
		return exitGeneralError
	}
	switch args[0] {
	case "list":
		return runMirrorsList(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown mirrors command %q\n", args[0])
		return exitGeneralError
	}
}

func runMirrorsList(args []string) int {
	fs := flag.NewFlagSet("mirrors list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	hostName := fs.String("host", "", "filter by host")
	jsonOut := fs.Bool("json", false, "emit JSON")
	configPath := fs.String("config", configPathArg(args), "config file path")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--json": true,
	}, map[string]bool{
		"--host": true, "--config": true,
	})); err != nil {
		return exitGeneralError
	}
	cfg, err := gitconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}

	profiles := gitconfig.Profiles(cfg)
	if *hostName != "" {
		profile, ok := gitmirror.FindProfile(profiles, *hostName)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown host %q\n", *hostName)
			return exitConfigError
		}
		profiles = []gitmirror.Profile{profile}
	}

	if *jsonOut {
		_ = output.JSON(os.Stdout, profiles)
		return exitOK
	}

	for _, profile := range profiles {
		fmt.Fprintf(os.Stdout, "%s\n", profile.Name)
		for _, mirror := range profile.Mirrors {
			fmt.Fprintf(os.Stdout, "  %s %s\n", mirror.Host, mirror.Mode)
		}
	}
	return exitOK
}

func runConfig(args []string) int {
	if len(args) == 0 || args[0] != "show" {
		fmt.Fprintln(os.Stderr, "usage: gip config show [--config PATH]")
		return exitGeneralError
	}
	fs := flag.NewFlagSet("config show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", configPathArg(args[1:]), "config file path")
	if err := fs.Parse(moveFlagsFirst(args[1:], nil, map[string]bool{
		"--config": true,
	})); err != nil {
		return exitGeneralError
	}
	cfg, err := gitconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}
	_ = output.JSON(os.Stdout, map[string]any{
		"timeout":            cfg.Timeout.String(),
		"parallel_probe":     cfg.ParallelProbe,
		"retries":            cfg.Retries,
		"prefer":             cfg.Prefer,
		"exclude":            cfg.Exclude,
		"mirrors":            cfg.Mirrors,
		"effective_profiles": gitconfig.Profiles(cfg),
		"loaded_from":        cfg.LoadedFrom,
		"config_files":       gitconfig.Paths(),
	})
	return exitOK
}

func selectCandidate(ctx context.Context, profiles []gitmirror.Profile, store gitstate.Store, u *giturl.GitHubURL, timeout time.Duration, concurrency int) (gitprobe.Result, []gitprobe.Result, int) {
	candidates := buildProbeCandidates(profiles, store, u)
	results := gitprobe.Candidates(ctx, candidates, u.Kind, gitprobe.Options{
		Timeout:     timeout,
		Concurrency: concurrency,
	})
	sortProbeResults(results)
	for _, result := range results {
		if result.OK {
			return result, results, exitOK
		}
	}
	for _, result := range results {
		if result.AuthRequired {
			return result, results, exitOK
		}
	}
	return gitprobe.Result{}, results, exitNoUsableMirror
}

func buildProbeCandidates(profiles []gitmirror.Profile, store gitstate.Store, u *giturl.GitHubURL) []gitrewrite.Candidate {
	profile, ok := gitmirror.FindProfile(profiles, u.Host)
	if !ok {
		return []gitrewrite.Candidate{sourceCandidate(u)}
	}

	candidates := gitrewrite.Candidates(u, profile)
	store.Rank(candidates)
	gitrewrite.SortCandidates(candidates)
	candidates = append(candidates, sourceCandidate(u))
	return candidates
}

func sourceCandidate(u *giturl.GitHubURL) gitrewrite.Candidate {
	return gitrewrite.Candidate{
		Original: u.Original,
		URL:      u.Original,
		Mirror: gitmirror.Mirror{
			Name:     "source",
			Host:     u.Host,
			Mode:     gitmirror.HostReplace,
			Priority: -10000,
		},
		Mode:     gitmirror.HostReplace,
		Priority: -10000,
	}
}

func printProbeResults(results []gitprobe.Result) {
	for _, result := range results {
		status := "fail"
		if result.OK {
			status = "ok"
		} else if result.AuthRequired {
			status = "warn"
		}
		fmt.Fprintf(os.Stdout, "%s %s %dms", status, result.URL, result.LatencyMS)
		if result.StatusCode != 0 {
			fmt.Fprintf(os.Stdout, " http=%d", result.StatusCode)
		}
		if result.Mirror != "" {
			fmt.Fprintf(os.Stdout, " mirror=%s", result.Mirror)
		}
		if result.Error != "" {
			fmt.Fprintf(os.Stdout, " error=%q", result.Error)
		}
		if result.Warning != "" {
			fmt.Fprintf(os.Stdout, " warning=%q", result.Warning)
		}
		fmt.Fprintln(os.Stdout)
	}
}

func sortProbeResults(results []gitprobe.Result) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].OK != results[j].OK {
			return results[i].OK
		}
		if results[i].AuthRequired != results[j].AuthRequired {
			return results[i].AuthRequired
		}
		if isSourceResult(results[i]) != isSourceResult(results[j]) {
			return !isSourceResult(results[i])
		}
		return results[i].LatencyMS < results[j].LatencyMS
	})
}

func isSourceResult(result gitprobe.Result) bool {
	return result.Mirror == "source"
}

func loadState() gitstate.Store {
	store, err := gitstate.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load state: %v\n", err)
		return gitstate.Store{Mirrors: map[string]gitstate.MirrorHealth{}}
	}
	return store
}

func saveState(store gitstate.Store) {
	if err := store.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save state: %v\n", err)
	}
}

func printUsage(w io.Writer) {
	usage := strings.TrimSpace(`
gip accelerates GitHub access through mirrors.

Usage:
  gip URL
  gip version [--json]
  gip completion bash|zsh|fish
  gip clone URL [--dir DIR] [--dry-run] [--json]
  gip install [--host HOST] [--json]
  gip uninstall [--host HOST]
  gip get URL [--output PATH] [--dry-run] [--json]
  gip rewrite URL [--all] [--plain] [--json]
  gip probe URL [--timeout DURATION] [--concurrency N] [--json]
  gip mirrors list [--host HOST] [--json]
  gip config show

Examples:
  gip version
  gip clone https://github.com/user/repo
  gip clone https://github.com/user/repo --dry-run
  gip install
  gip uninstall
  gip get https://github.com/user/repo/releases/download/v1.0/binary.tar.gz
  gip rewrite https://github.com/user/repo --all
  gip probe https://github.com/user/repo --timeout 8s
  gip mirrors list --host github.com
`)
	fmt.Fprintln(w, usage)
}

// moveFlagsFirst reorders args so flag-style arguments precede operands.
func moveFlagsFirst(args []string, boolFlags map[string]bool, valueFlags map[string]bool) []string {
	flags := make([]string, 0, len(args))
	operands := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if boolFlags[arg] {
			flags = append(flags, arg)
			continue
		}
		if isInlineValueFlag(arg, valueFlags) {
			flags = append(flags, arg)
			continue
		}
		if valueFlags[arg] && i+1 < len(args) {
			flags = append(flags, arg, args[i+1])
			i++
			continue
		}
		operands = append(operands, arg)
	}
	return append(flags, operands...)
}

func moveBoolFlagsFirst(args []string, boolFlags map[string]bool) []string {
	return moveFlagsFirst(args, boolFlags, nil)
}

func isInlineValueFlag(arg string, valueFlags map[string]bool) bool {
	for valueFlag := range valueFlags {
		if strings.HasPrefix(arg, valueFlag+"=") {
			return true
		}
	}
	return false
}

func configPathArg(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
		if value, ok := strings.CutPrefix(arg, "--config="); ok {
			return value
		}
	}
	return ""
}