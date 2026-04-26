package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"time"
)

type Engine interface {
	Name() string
	Available(ctx context.Context) error
	Pull(ctx context.Context, image string, options PullOptions) error
	Tag(ctx context.Context, source string, target string) error
	Remove(ctx context.Context, image string) error
	RepoDigests(ctx context.Context, image string) ([]string, error)
}

type PullOptions struct {
	Platform string
	Timeout  time.Duration
	Stdout   io.Writer
	Stderr   io.Writer
}

type CLI struct {
	name              string
	binary            string
	availabilityArgs  []string
	platformFlagStyle string
}

func Names() []string {
	return []string{"docker", "podman", "nerdctl"}
}

func New(name string) (Engine, error) {
	switch name {
	case "", "docker":
		return CLI{
			name:              "docker",
			binary:            "docker",
			availabilityArgs:  []string{"version", "--format", "{{.Server.Version}}"},
			platformFlagStyle: "long",
		}, nil
	case "podman":
		return CLI{
			name:              "podman",
			binary:            "podman",
			availabilityArgs:  []string{"version", "--format", "{{.Server.Version}}"},
			platformFlagStyle: "long",
		}, nil
	case "nerdctl":
		return CLI{
			name:              "nerdctl",
			binary:            "nerdctl",
			availabilityArgs:  []string{"version"},
			platformFlagStyle: "long",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported engine %q; supported engines: %v", name, Names())
	}
}

func (c CLI) Name() string {
	return c.name
}

func (c CLI) Available(ctx context.Context) error {
	if c.binary == "" {
		return errors.New("engine binary is empty")
	}
	if _, err := exec.LookPath(c.binary); err != nil {
		return fmt.Errorf("%s not found in PATH", c.binary)
	}
	cmd := exec.CommandContext(ctx, c.binary, c.availabilityArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s engine unavailable: %s: %w", c.name, string(output), err)
	}
	return nil
}

func (c CLI) Pull(ctx context.Context, image string, options PullOptions) error {
	args := []string{"pull"}
	if options.Platform != "" {
		args = append(args, c.platformFlag(options.Platform)...)
	}
	args = append(args, image)
	return c.run(ctx, options.Timeout, options.Stdout, options.Stderr, args...)
}

func (c CLI) Tag(ctx context.Context, source string, target string) error {
	return c.run(ctx, 0, nil, nil, "tag", source, target)
}

func (c CLI) Remove(ctx context.Context, image string) error {
	return c.run(ctx, 0, nil, nil, "rmi", image)
}

func (c CLI) RepoDigests(ctx context.Context, image string) ([]string, error) {
	output, err := c.output(ctx, 0, "image", "inspect", "--format", "{{json .RepoDigests}}", image)
	if err != nil {
		return nil, err
	}

	var digests []string
	if err := json.Unmarshal(output, &digests); err != nil {
		return nil, fmt.Errorf("parse %s image inspect RepoDigests: %w", c.name, err)
	}
	return digests, nil
}

func (c CLI) platformFlag(platform string) []string {
	switch c.platformFlagStyle {
	case "long":
		return []string{"--platform", platform}
	default:
		return []string{"--platform", platform}
	}
}

func (c CLI) run(ctx context.Context, timeout time.Duration, stdout io.Writer, stderr io.Writer, args ...string) error {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, c.binary, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if stdout == nil && stderr == nil {
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s %v failed: %s: %w", c.name, args, string(output), err)
		}
		return nil
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v failed: %w", c.name, args, err)
	}
	return nil
}

func (c CLI) output(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, c.binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %v failed: %s: %w", c.name, args, string(output), err)
	}
	return output, nil
}

func IsSupported(name string) bool {
	return slices.Contains(Names(), name)
}
