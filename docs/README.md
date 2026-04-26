# Image Mirror CLI Docs

This directory describes a new CLI tool for accelerated container image pulls in constrained network environments.

The directories at the repository root are third-party open source tools downloaded for research. They may inform product requirements, but this project must not copy their implementation, data model, documentation wording, or mirror list representation.

## Documents

- [Product Brief](product.md): purpose, users, goals, non-goals, and design principles.
- [CLI Specification](cli.md): commands, flags, output, exit codes, and examples.
- [Architecture](architecture.md): modules, data flow, engine adapters, and persistence.
- [Mirror Model](mirrors.md): registry profiles, rewrite modes, scoring, and official config policy.
- [Roadmap](roadmap.md): MVP scope and implementation phases.
- [Release](release.md): local builds, release archives, and version metadata.

## Working Name

The current working name is `mip`, short for `mirror image pull`.

The name is provisional. The design should not bake the binary name into core packages.
