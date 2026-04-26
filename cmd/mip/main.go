package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vlln/mip/internal/completion"
	appconfig "github.com/vlln/mip/internal/config"
	"github.com/vlln/mip/internal/engine"
	"github.com/vlln/mip/internal/output"
	"github.com/vlln/mip/internal/probe"
	"github.com/vlln/mip/internal/ref"
	"github.com/vlln/mip/internal/registry"
	"github.com/vlln/mip/internal/rewrite"
	"github.com/vlln/mip/internal/state"
	"github.com/vlln/mip/internal/version"
)

const (
	exitOK             = 0
	exitGeneralError   = 1
	exitInvalidRef     = 2
	exitNoUsableMirror = 3
	exitEngineError    = 4
	exitPullFailed     = 5
	exitDigestMismatch = 6
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
	case "pull":
		return runPull(args[1:])
	default:
		return runPull(append([]string{args[0]}, args[1:]...))
	}
}

func runVersion(args []string) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--json": true,
	}, nil)); err != nil {
		return exitGeneralError
	}

	info := version.Get()
	if *jsonOut {
		_ = output.JSON(os.Stdout, info)
		return exitOK
	}
	fmt.Fprintf(os.Stdout, "mip %s\n", info.Version)
	fmt.Fprintf(os.Stdout, "commit: %s\n", info.Commit)
	fmt.Fprintf(os.Stdout, "date: %s\n", info.Date)
	fmt.Fprintf(os.Stdout, "go: %s\n", info.Go)
	fmt.Fprintf(os.Stdout, "platform: %s/%s\n", info.OS, info.Arch)
	return exitOK
}

func runCompletion(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: mip completion bash|zsh|fish")
		return exitGeneralError
	}
	script, err := completion.Script(args[0])
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
	plain := fs.Bool("plain", false, "print only image references")
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
		fmt.Fprintln(os.Stderr, "usage: mip rewrite IMAGE [--all] [--plain] [--json]")
		return exitGeneralError
	}
	cfg, err := appconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}
	profiles := appconfig.Profiles(cfg)

	image, err := ref.Parse(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid image reference: %v\n", err)
		return exitInvalidRef
	}

	profile, ok := appconfig.FindProfile(profiles, image.Registry)
	if !ok {
		if *jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"image":      image.String(),
				"candidates": []rewrite.Candidate{},
			})
			return exitOK
		}
		if *plain {
			fmt.Fprintln(os.Stdout, image.String())
			return exitOK
		}
		fmt.Fprintf(os.Stderr, "no configured mirrors for registry %q\n", image.Registry)
		fmt.Fprintln(os.Stdout, image.String())
		return exitNoUsableMirror
	}

	candidates := rewrite.Candidates(image, profile)
	if !*all && len(candidates) > 1 {
		candidates = candidates[:1]
	}

	if *jsonOut {
		_ = output.JSON(os.Stdout, map[string]any{
			"image":      image.String(),
			"registry":   profile.Name,
			"candidates": candidates,
		})
		return exitOK
	}

	if *plain {
		for _, candidate := range candidates {
			fmt.Fprintln(os.Stdout, candidate.Image)
		}
		return exitOK
	}

	fmt.Fprintf(os.Stdout, "image: %s\n", image.String())
	fmt.Fprintf(os.Stdout, "registry: %s\n", profile.Name)
	for i, candidate := range candidates {
		fmt.Fprintf(os.Stdout, "candidate[%d]: %s\n", i, candidate.Image)
		fmt.Fprintf(os.Stdout, "  mirror: %s\n", candidate.Mirror.Name)
		fmt.Fprintf(os.Stdout, "  mode: %s\n", candidate.Mode)
	}
	return exitOK
}

func runProbe(args []string) int {
	fs := flag.NewFlagSet("probe", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	configPath := fs.String("config", configPathArg(args), "config file path")
	cfg, err := appconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}
	timeout := fs.Duration("timeout", cfg.Timeout, "per candidate timeout")
	concurrency := fs.Int("concurrency", cfg.ParallelProbe, "maximum concurrent probes")
	platform := fs.String("platform", "", "platform used for manifest list selection, for example linux/amd64")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--json": true,
	}, map[string]bool{
		"--config": true, "--timeout": true, "--concurrency": true, "--platform": true,
	})); err != nil {
		return exitGeneralError
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: mip probe IMAGE [--platform PLATFORM] [--timeout DURATION] [--concurrency N] [--json]")
		return exitGeneralError
	}

	image, err := ref.Parse(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid image reference: %v\n", err)
		return exitInvalidRef
	}

	profiles := appconfig.Profiles(cfg)
	store := loadState()
	_, results, code := selectCandidate(context.Background(), profiles, store, image, *timeout, *concurrency, *platform)
	saveState(store.Record(results))
	profile, _ := appconfig.FindProfile(profiles, image.Registry)

	if *jsonOut {
		payload := map[string]any{
			"image":   image.String(),
			"results": results,
		}
		if profile.Name != "" {
			payload["registry"] = profile.Name
		}
		_ = output.JSON(os.Stdout, payload)
		return exitOK
	}

	printProbeResults(results)
	return code
}

func runPull(args []string) int {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	dryRun := fs.Bool("dry-run", false, "show selected candidate without pulling")
	noRetag := fs.Bool("no-retag", false, "keep the mirrored image name locally")
	noVerifyDigest := fs.Bool("no-verify-digest", false, "skip digest verification after pull")
	configPath := fs.String("config", configPathArg(args), "config file path")
	cfg, err := appconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}
	engineName := fs.String("engine", cfg.Engine, "container engine: docker, podman, or nerdctl")
	platform := fs.String("platform", "", "platform passed to the engine pull command, for example linux/amd64")
	probeTimeout := fs.Duration("timeout", cfg.Timeout, "per candidate probe timeout")
	pullTimeout := fs.Duration("pull-timeout", cfg.PullTimeout, "engine pull timeout")
	concurrency := fs.Int("concurrency", cfg.ParallelProbe, "maximum concurrent probes")
	retries := fs.Int("retries", cfg.Retries, "pull attempts per candidate")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--json": true, "--dry-run": true, "--no-retag": true, "--no-verify-digest": true,
	}, map[string]bool{
		"--config": true, "--engine": true, "--platform": true, "--timeout": true, "--pull-timeout": true, "--concurrency": true, "--retries": true,
	})); err != nil {
		return exitGeneralError
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: mip pull IMAGE [--engine docker|podman|nerdctl] [--dry-run] [--platform PLATFORM] [--retries N] [--no-verify-digest] [--json]")
		return exitGeneralError
	}
	if *retries < 1 {
		fmt.Fprintln(os.Stderr, "retries must be at least 1")
		return exitConfigError
	}

	image, err := ref.Parse(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid image reference: %v\n", err)
		return exitInvalidRef
	}

	runner, err := engine.New(*engineName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitConfigError
	}

	profiles := appconfig.Profiles(cfg)
	store := loadState()
	selected, results, code := selectCandidate(context.Background(), profiles, store, image, *probeTimeout, *concurrency, *platform)
	saveState(store.Record(results))
	if code != exitOK {
		if *jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"image":   image.String(),
				"status":  "no_usable_mirror",
				"results": results,
			})
		} else {
			printProbeResults(results)
		}
		return code
	}

	if *dryRun {
		if *jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"image":    image.String(),
				"selected": selected.Image,
				"engine":   runner.Name(),
				"status":   "dry_run",
				"results":  results,
			})
			return exitOK
		}
		fmt.Fprintf(os.Stdout, "image: %s\n", image.String())
		fmt.Fprintf(os.Stdout, "selected: %s\n", selected.Image)
		fmt.Fprintf(os.Stdout, "engine: %s\n", runner.Name())
		fmt.Fprintln(os.Stdout, "status: dry-run")
		return exitOK
	}

	if err := runner.Available(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitEngineError
	}

	start := time.Now()
	pullResult, pullCode := pullWithFallback(context.Background(), runner, image.String(), successfulResults(results), pullOptions{
		platform:       *platform,
		pullTimeout:    *pullTimeout,
		retries:        *retries,
		noRetag:        *noRetag,
		noVerifyDigest: *noVerifyDigest,
	})
	elapsed := time.Since(start).Milliseconds()
	if pullCode != exitOK {
		if *jsonOut {
			_ = output.JSON(os.Stdout, map[string]any{
				"image":      image.String(),
				"selected":   selected.Image,
				"engine":     runner.Name(),
				"status":     "pull_failed",
				"attempts":   pullResult.Attempts,
				"elapsed_ms": elapsed,
			})
		}
		return pullCode
	}
	selected = pullResult.Selected

	if *jsonOut {
		_ = output.JSON(os.Stdout, map[string]any{
			"image":        image.String(),
			"selected":     selected.Image,
			"mirror":       selected.Mirror,
			"engine":       runner.Name(),
			"status":       "pulled",
			"retagged":     pullResult.Retagged,
			"verified":     pullResult.Verified,
			"digest":       selected.Digest,
			"index_digest": selected.IndexDigest,
			"elapsed_ms":   elapsed,
			"attempts":     pullResult.Attempts,
		})
		return exitOK
	}

	fmt.Fprintf(os.Stdout, "image: %s\n", image.String())
	fmt.Fprintf(os.Stdout, "selected: %s\n", selected.Image)
	fmt.Fprintf(os.Stdout, "engine: %s\n", runner.Name())
	fmt.Fprintln(os.Stdout, "status: pulled")
	if selected.Digest != "" {
		fmt.Fprintf(os.Stdout, "digest: %s\n", selected.Digest)
		if selected.IndexDigest != "" && selected.IndexDigest != selected.Digest {
			fmt.Fprintf(os.Stdout, "index_digest: %s\n", selected.IndexDigest)
		}
		fmt.Fprintf(os.Stdout, "verified: %t\n", pullResult.Verified)
	}
	fmt.Fprintf(os.Stdout, "elapsed: %.1fs\n", float64(elapsed)/1000)
	return exitOK
}

type pullOptions struct {
	platform       string
	pullTimeout    time.Duration
	retries        int
	noRetag        bool
	noVerifyDigest bool
}

type pullAttempt struct {
	Image   string `json:"image"`
	Attempt int    `json:"attempt"`
	Error   string `json:"error,omitempty"`
}

type pullOutcome struct {
	Selected probe.Result  `json:"selected"`
	Verified bool          `json:"verified"`
	Retagged bool          `json:"retagged"`
	Attempts []pullAttempt `json:"attempts"`
}

func pullWithFallback(ctx context.Context, runner engine.Engine, target string, candidates []probe.Result, options pullOptions) (pullOutcome, int) {
	if options.retries < 1 {
		options.retries = 1
	}

	outcome := pullOutcome{}
	digestMismatch := false
	for _, candidate := range candidates {
		for attempt := 1; attempt <= options.retries; attempt++ {
			err := pullCandidate(ctx, runner, target, candidate, options)
			record := pullAttempt{Image: candidate.Image, Attempt: attempt}
			if err != nil {
				record.Error = err.Error()
				outcome.Attempts = append(outcome.Attempts, record)
				fmt.Fprintf(os.Stderr, "pull attempt failed: image=%s attempt=%d/%d error=%v\n", candidate.Image, attempt, options.retries, err)
				if strings.Contains(err.Error(), "digest verification failed") {
					digestMismatch = true
					break
				}
				if attempt < options.retries {
					retrySleep(retryDelay(attempt))
				}
				continue
			}
			outcome.Attempts = append(outcome.Attempts, record)
			outcome.Selected = candidate
			outcome.Verified = !options.noVerifyDigest && candidate.Digest != ""
			outcome.Retagged = !options.noRetag && candidate.Image != target
			return outcome, exitOK
		}
	}
	if digestMismatch {
		return outcome, exitDigestMismatch
	}
	return outcome, exitPullFailed
}

func pullCandidate(ctx context.Context, runner engine.Engine, target string, candidate probe.Result, options pullOptions) error {
	if err := runner.Pull(ctx, candidate.Image, engine.PullOptions{
		Platform: options.platform,
		Timeout:  options.pullTimeout,
		Stdout:   os.Stderr,
		Stderr:   os.Stderr,
	}); err != nil {
		return err
	}

	if !options.noVerifyDigest && candidate.Digest != "" {
		repoDigests, err := runner.RepoDigests(ctx, candidate.Image)
		if err != nil {
			return err
		}
		expectedDigests := verificationDigests(candidate)
		if !hasAnyDigest(repoDigests, expectedDigests) {
			return fmt.Errorf("digest verification failed: expected one of %v, repo digests: %v", expectedDigests, repoDigests)
		}
	}

	if !options.noRetag && candidate.Image != target {
		if err := runner.Tag(ctx, candidate.Image, target); err != nil {
			return err
		}
		_ = runner.Remove(ctx, candidate.Image)
	}
	return nil
}

func retryDelay(attempt int) time.Duration {
	delay := time.Duration(attempt) * 500 * time.Millisecond
	if delay > 3*time.Second {
		return 3 * time.Second
	}
	return delay
}

var retrySleep = time.Sleep

func successfulResults(results []probe.Result) []probe.Result {
	successful := make([]probe.Result, 0, len(results))
	for _, result := range results {
		if result.OK {
			successful = append(successful, result)
		}
	}
	return successful
}

func selectCandidate(ctx context.Context, profiles []registry.Profile, store state.Store, image ref.Reference, timeout time.Duration, concurrency int, platform string) (probe.Result, []probe.Result, int) {
	candidates := buildProbeCandidates(profiles, store, image)
	results := probe.Candidates(ctx, candidates, probe.Options{
		Timeout:     timeout,
		Concurrency: concurrency,
		Platform:    platform,
	})
	sortProbeResults(results)
	for _, result := range results {
		if result.OK {
			return result, results, exitOK
		}
	}
	return probe.Result{}, results, exitNoUsableMirror
}

func buildProbeCandidates(profiles []registry.Profile, store state.Store, image ref.Reference) []rewrite.Candidate {
	profile, ok := appconfig.FindProfile(profiles, image.Registry)
	if !ok {
		return []rewrite.Candidate{sourceCandidate(image, image.Registry)}
	}

	candidates := rewrite.Candidates(image, profile)
	store.Rank(candidates)
	rewrite.SortCandidates(candidates)
	candidates = append(candidates, sourceCandidate(image, profile.Name))
	return candidates
}

func sourceCandidate(image ref.Reference, registryName string) rewrite.Candidate {
	return rewrite.Candidate{
		Original: image.String(),
		Image:    image.String(),
		Registry: registryName,
		Mirror: registry.Mirror{
			Name:     "source",
			Host:     image.Registry,
			Mode:     registry.HostReplace,
			Priority: -10000,
		},
		Mode:     registry.HostReplace,
		Priority: -10000,
	}
}

func runMirrors(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mip mirrors list [--registry REGISTRY] [--json]")
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

func printProbeResults(results []probe.Result) {
	for _, result := range results {
		status := "fail"
		if result.OK {
			status = "ok"
		}
		fmt.Fprintf(os.Stdout, "%s %s %dms", status, result.Image, result.LatencyMS)
		if result.StatusCode != 0 {
			fmt.Fprintf(os.Stdout, " http=%d", result.StatusCode)
		}
		if result.Mirror != "" {
			fmt.Fprintf(os.Stdout, " mirror=%s", result.Mirror)
		}
		if result.Digest != "" {
			fmt.Fprintf(os.Stdout, " digest=%s", result.Digest)
		}
		if result.Error != "" {
			fmt.Fprintf(os.Stdout, " error=%q", result.Error)
		}
		fmt.Fprintln(os.Stdout)
	}
}

func sortProbeResults(results []probe.Result) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].OK != results[j].OK {
			return results[i].OK
		}
		if isSourceResult(results[i]) != isSourceResult(results[j]) {
			return !isSourceResult(results[i])
		}
		return results[i].LatencyMS < results[j].LatencyMS
	})
}

func isSourceResult(result probe.Result) bool {
	return result.Mirror == "source"
}

func hasAnyDigest(repoDigests []string, expected []string) bool {
	for _, digest := range expected {
		for _, repoDigest := range repoDigests {
			if strings.HasSuffix(repoDigest, "@"+digest) || repoDigest == digest {
				return true
			}
		}
	}
	return false
}

func verificationDigests(result probe.Result) []string {
	digests := []string{}
	if result.IndexDigest != "" {
		digests = append(digests, result.IndexDigest)
	}
	if result.Digest != "" && result.Digest != result.IndexDigest {
		digests = append(digests, result.Digest)
	}
	return digests
}

func loadState() state.Store {
	store, err := state.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load state: %v\n", err)
		return state.Store{Mirrors: map[string]state.MirrorHealth{}}
	}
	return store
}

func saveState(store state.Store) {
	if err := store.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save state: %v\n", err)
	}
}

func runMirrorsList(args []string) int {
	fs := flag.NewFlagSet("mirrors list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	registryName := fs.String("registry", "", "filter by registry")
	jsonOut := fs.Bool("json", false, "emit JSON")
	configPath := fs.String("config", configPathArg(args), "config file path")
	if err := fs.Parse(moveFlagsFirst(args, map[string]bool{
		"--json": true,
	}, map[string]bool{
		"--registry": true, "--config": true,
	})); err != nil {
		return exitGeneralError
	}
	cfg, err := appconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}

	profiles := appconfig.Profiles(cfg)
	if *registryName != "" {
		profile, ok := appconfig.FindProfile(profiles, *registryName)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown registry %q\n", *registryName)
			return exitConfigError
		}
		profiles = []registry.Profile{profile}
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
		fmt.Fprintln(os.Stderr, "usage: mip config show [--config PATH]")
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
	cfg, err := appconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return exitConfigError
	}
	_ = output.JSON(os.Stdout, map[string]any{
		"engine":             cfg.Engine,
		"engines":            engine.Names(),
		"timeout":            cfg.Timeout.String(),
		"pull_timeout":       cfg.PullTimeout.String(),
		"parallel_probe":     cfg.ParallelProbe,
		"retries":            cfg.Retries,
		"prefer":             cfg.Prefer,
		"exclude":            cfg.Exclude,
		"registries":         cfg.Registries,
		"effective_profiles": appconfig.Profiles(cfg),
		"loaded_from":        cfg.LoadedFrom,
		"config_files":       appconfig.Paths(),
	})
	return exitOK
}

func printUsage(out *os.File) {
	usage := strings.TrimSpace(`
mip accelerates container image pulls through registry-aware mirrors.

Usage:
  mip IMAGE
  mip version [--json]
  mip completion bash|zsh|fish
  mip pull IMAGE [--engine docker|podman|nerdctl] [--dry-run] [--platform PLATFORM] [--retries N] [--no-verify-digest] [--json]
  mip rewrite IMAGE [--all] [--plain] [--json]
  mip probe IMAGE [--platform PLATFORM] [--timeout DURATION] [--concurrency N] [--json]
  mip mirrors list [--registry REGISTRY] [--json]
  mip config show

Examples:
  mip version
  mip completion bash
  mip nginx:1.27 --dry-run
  mip pull nginx:1.27 --engine podman --dry-run
  mip rewrite nginx:1.27 --all
  mip probe nginx:1.27
  mip rewrite ghcr.io/actions/actions-runner:latest --plain
  mip mirrors list --registry registry.k8s.io
`)
	fmt.Fprintln(out, usage)
}

func moveBoolFlagsFirst(args []string, boolFlags map[string]bool) []string {
	return moveFlagsFirst(args, boolFlags, nil)
}

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
