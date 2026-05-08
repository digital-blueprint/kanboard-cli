# kanboard-cli

A command-line client for [Kanboard](https://kanboard.org/) — manage projects,
tasks, and comments directly from your terminal or scripts.

## Features

- **Projects** — list, create, delete
- **Tasks** — list, get, create, delete, move (column/position), move to another project, open, close
- **Comments** — list, add, delete
- **Secure credential storage** — API token is stored in the OS keyring (GNOME Keyring / libsecret on Linux, Keychain on macOS, Credential Manager on Windows); only the username is written to disk
- **JSON output** — every command accepts `--json` for machine-readable output, suitable for agents and scripting
- **Version info** — build-time version, commit, and date injection

## Requirements

| Platform | Keyring backend |
|----------|----------------|
| Linux    | libsecret / GNOME Keyring (or KWallet via D-Bus) |
| macOS    | macOS Keychain Services |
| Windows  | Windows Credential Manager |

A running [Kanboard](https://github.com/kanboard/kanboard) instance with API access enabled.

## Installation

### From source (Go)

```sh
git clone <repo-url> kanboard-cli
cd kanboard-cli
just build          # produces ./kanboard-cli
```

### With Nix

```sh
nix build .#default          # result/bin/kanboard-cli
nix run .#default -- --help  # run without installing
```

### Pre-built binaries

Download the appropriate archive for your platform from the
[Releases](../../releases) page, extract, and place `kanboard-cli` somewhere
on your `$PATH`.

## Configuration

### Server URL

Set the `KANBOARD_URL` environment variable to the base URL of your Kanboard
instance:

```sh
export KANBOARD_URL=https://kanboard.example.com
```

Add it to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) to make it permanent.

### Authentication

```sh
kanboard-cli auth login
```

You will be prompted for:

- **Username** — use `jsonrpc` for the application API (token from
  *Settings › API*), or your own username for the user API (requires a
  personal access token from your profile page).
- **API token** — entered via a hidden prompt (not echoed).

The username is stored in `$XDG_CONFIG_HOME/kanboard-cli/config.json`.
The token is stored **only** in the OS keyring — never in a plain-text file.

#### Environment variable override (CI/CD)

```sh
export KANBOARD_URL=https://kanboard.example.com
export KANBOARD_USERNAME=jsonrpc
export KANBOARD_TOKEN=<token>
kanboard-cli project list
```

Setting `KANBOARD_TOKEN` bypasses the keyring entirely.

## Usage

```
kanboard-cli [--json] <command> [subcommand] [flags]
```

The `--json` flag is available on every command and outputs the result as
pretty-printed JSON instead of a human-readable table.

### Auth

```sh
kanboard-cli auth login           # store credentials in OS keyring
kanboard-cli auth status          # show server URL and masked token
kanboard-cli auth logout          # remove stored credentials
```

### Projects

```sh
kanboard-cli project list
kanboard-cli project create "My Project" --description "Optional description"
kanboard-cli project delete <project-id>
```

### Tasks

```sh
# List active tasks in a project
kanboard-cli task list --project-id <id>

# Include closed tasks
kanboard-cli task list --project-id <id> --all

# Show full task details
kanboard-cli task get <task-id>

# Create a task
kanboard-cli task create "Fix login bug" \
  --project-id <id> \
  --column-id <id> \
  --description "Steps to reproduce…" \
  --color red \
  --due "2024-12-31 09:00"

# Move within the same project board
kanboard-cli task move <task-id> \
  --project-id <id> \
  --column-id <target-column-id> \
  --position 1

# Move to a different project
kanboard-cli task move-project <task-id> <target-project-id>

# Close / re-open
kanboard-cli task close <task-id>
kanboard-cli task open  <task-id>

# Delete
kanboard-cli task delete <task-id>
```

### Comments

```sh
kanboard-cli comment list <task-id>
kanboard-cli comment add  <task-id> "This looks good!"
kanboard-cli comment delete <comment-id>
```

### Version

```sh
kanboard-cli version
kanboard-cli --json version
```

## JSON output

Pass `--json` to any command to get structured JSON output, useful for piping
into `jq` or calling from scripts and agents:

```sh
kanboard-cli --json task list --project-id 1 | jq '.[].title'
kanboard-cli --json task get 42 | jq '{id, title, status: (if .is_active == "1" then "open" else "closed" end)}'
```

Mutating commands return a small confirmation object, e.g.:

```json
{ "task_id": 42, "deleted": true }
```

## Development

### Dev shell (devenv)

```sh
devenv shell   # or: nix develop
```

The shell provides Go, gopls, golangci-lint, goimports, and (on Linux)
libsecret/pkg-config.

### Justfile recipes

```sh
just           # list all recipes
just build     # build with version/commit/date ldflags
just run <args>
just test
just test-race
just lint
just fmt
just clean
just vendor    # go mod tidy + go mod vendor
just nix-build
just nix-run <args>
```

### Project structure

```
kanboard-cli/
├── main.go
├── go.mod / go.sum
├── vendor/
├── devenv.nix          devenv dev shell
├── flake.nix           nix build + devShell
├── justfile            task runner
├── .goreleaser.yaml    release configuration
└── internal/
    ├── api/
    │   ├── client.go   JSON-RPC HTTP client (Basic Auth)
    │   ├── flextime.go FlexibleTime — handles numeric/string timestamps
    │   └── methods.go  typed wrappers for all API procedures
    ├── config/
    │   └── config.go   OS keyring + config file management
    ├── cmd/
    │   ├── root.go     root command + --json flag + helpers
    │   ├── auth.go
    │   ├── project.go
    │   ├── task.go
    │   ├── comment.go
    │   └── version.go
    └── version/
        └── version.go  build-time version variables
```

## Releasing

Push to the `release` branch to trigger the GitHub Actions release workflow.
GoReleaser will cross-compile for Linux, macOS, and Windows, create a GitHub
Release, and upload archives with checksums.

Tag the commit with `vX.Y.Z` before pushing to produce a properly versioned
release:

```sh
git tag v1.0.0
git push origin v1.0.0:release
```

## License

Copyright (C) 2024 TU Graz

This program is free software: you can redistribute it and/or modify it under
the terms of the **GNU General Public License version 3** (or any later
version) as published by the Free Software Foundation.

See [LICENSE](LICENSE) or <https://www.gnu.org/licenses/gpl-3.0.html>.
