# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `task assign <task-id> [task-id...]` to assign one or more tasks to the
  authenticated user, with `--user-id` for assigning tasks to a specific user.

## [0.3.0] - 2026-05-08

### Added
- `task list --status open|closed|all` to select tasks by Kanboard status.
- `task list --tag <tag>` to filter tasks by an exact tag name.
- `task list --column <column-id-or-title>` to filter tasks by board column ID
  or exact column title.
- JSON output for tag-filtered task lists now includes the matching task tags.

## [0.2.0] - 2026-05-08

### Added
- `auth login` now prompts for the Kanboard server URL and stores it in the
  user config file alongside the username.
- `auth login --url` for non-interactive server URL configuration.

### Changed
- Commands now use the stored server URL by default; `KANBOARD_URL` remains an
  environment variable override.

## [0.1.0] - 2026-05-08

### Added
- Initial implementation of `kanboard-cli`
- **Auth** — `auth login`, `auth status`, `auth logout`; API token stored
  securely in the OS keyring (libsecret on Linux, Keychain on macOS,
  Credential Manager on Windows); username stored in
  `$XDG_CONFIG_HOME/kanboard-cli/config.json` (mode 0600)
- **Projects** — `project list`, `project create`, `project delete`
- **Tasks** — `task list`, `task get`, `task create`, `task delete`,
  `task move`, `task move-project`, `task close`, `task open`
- **Comments** — `comment list`, `comment add`, `comment delete`
- **Version** — `version` command with build-time injection of version,
  commit hash, and build date via `-ldflags`
- `--json` persistent flag on the root command: every subcommand outputs
  pretty-printed JSON instead of a human-readable table when set; suitable
  for agents, scripts, and piping into `jq`
- `FlexibleTime` type to handle Kanboard API date fields that are returned
  as either a bare JSON number (Unix timestamp) or a formatted string —
  fixes unmarshal errors on `date_due` and related fields
- `devenv.nix` development shell with Go, gopls, golangci-lint, goimports,
  just, and (Linux) libsecret/pkg-config/dbus
- `flake.nix` Nix package with `buildGoModule`, `installShellCompletion`
  for bash/zsh/fish, and libsecret build inputs on Linux
- `justfile` with `build`, `run`, `test`, `test-race`, `lint`, `fmt`,
  `clean`, `vendor`, `nix-build`, `nix-run`, `version` recipes
- `.goreleaser.yaml` for cross-compiled releases (Linux amd64/arm64,
  macOS amd64/arm64, Windows amd64) with checksums and changelog
- GitHub Actions CI workflow (build + test + vet on every push/PR)
- GitHub Actions release workflow (GoReleaser on push to `release` branch)
- `README.md` with installation, configuration, usage, and development docs
- `LICENSE` — GNU General Public License v3.0

[Unreleased]: https://github.com/tu-graz/kanboard-cli/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/tu-graz/kanboard-cli/releases/tag/v0.3.0
[0.2.0]: https://github.com/tu-graz/kanboard-cli/releases/tag/v0.2.0
[0.1.0]: https://github.com/tu-graz/kanboard-cli/releases/tag/v0.1.0
