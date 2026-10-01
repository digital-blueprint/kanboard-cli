# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `task update <task-id> [task-id...]` (alias `task edit`) to change any task
  field on one or more tasks: `--title`, description (`-d`, `-F` file/stdin,
  `-e` editor, `--append-description`), `--color`, `--assignee`, `--category`,
  `--priority`, `--complexity`, `--reference`, `--due`, `--start`,
  `--estimate`, `--spent`, recurrence (`--recurrence`, `--recurrence-trigger`,
  `--recurrence-every`, `--recurrence-base`), tags (`--tag`, `--untag`,
  `--set-tags`, `--clear-tags`), board placement (`--project`, `--column`,
  `--swimlane`, `--position`), and `--status open|closed`. Only given flags
  are changed, unchanged values are skipped, `none` clears a value, and
  `--dry-run` previews the changes. Projects, columns, swimlanes, categories,
  and users can be given by ID or name; assignees also accept `me` and
  usernames.
- `task create` accepts the same field flags as `task update`, plus
  `--swimlane`. `--project`/`-p` and `--column`/`-c` now accept names
  (`--project-id` and `--column-id` still work).
- `comment edit <comment-id> [content]` (alias `update`) with `--file`,
  `--append`, piped stdin, or `$VISUAL`/`$EDITOR`.
- `project update <project>` (alias `edit`) to change name, description,
  identifier, email, owner, start/end date, priority range, status
  (active/inactive), and public access, with `--dry-run`.

### Changed

- `task get` shows assignee, category, swimlane, priority, complexity, start
  date, time tracking, recurrence, tags, and the task URL, with names instead
  of IDs where possible. `--json` output includes these fields and `tags`.
- Dates given without a time (`--due 2026-10-15`) are sent as midnight;
  previously Kanboard filled in the current time of day.

### Fixed

- Reading the tags of a task without tags no longer fails (Kanboard returns
  an empty JSON array instead of an object), which affected
  `task list --tag`.
- Looking up a project by a name that does not exist now reports
  "project not found" instead of a JSON decoding error.
- `task create` now reports an error when Kanboard rejects the task instead of
  a JSON decoding error.

## [0.5.0] - 2026-09-30

### Added

- `subtask` command group: `list`, `get`, `add` (one or more titles, with
  `--user-id`, `--time-estimated`, `--time-spent`, `--status`), `update`,
  `done`, and `delete`.
- `subtask from-checklist <task-id>` (aliases `split`, `sync`) to create
  subtasks from the Markdown checkbox list in a task description. Checked
  items become done subtasks. A link to the task page, with the subtask ID as
  link text, is appended to each checklist line, so re-running the command
  only converts newly added items. Links pointing to the subtask edit form
  (which opens as a broken page) are rewritten to the new format. Unlinked items matching an existing subtask title are linked to that
  subtask. Supports `--dry-run`, `--skip-checked`, `--allow-duplicates`,
  `--user-id`, `--no-links`, and `--remove-from-description`.

### Changed

- `task get` now lists the task's subtasks (and includes them as `subtasks`
  in `--json` output).
- `task get` now returns an error when the task does not exist instead of
  printing empty fields.

## [0.4.4] - 2026-05-15

### Fixed

- Addressed `golangci-lint` issues reported by pre-commit hooks, by improving error handling.

## [0.4.0] - 2026-05-11

### Added

- `task assign <task-id> [task-id...]` to assign one or more tasks to the
  authenticated user, with `--user-id` for assigning tasks to a specific user.
- `task move-board <task-id> [task-id...] --project <id-or-name>` to move one
  or more tasks to another project board, with `--swimlane` and `--column`
  resolving IDs or exact names.

### Changed

- `task close` now accepts one or more task IDs in a single command.

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

[Unreleased]: https://github.com/tu-graz/kanboard-cli/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/tu-graz/kanboard-cli/compare/v0.4.4...v0.5.0
[0.4.4]: https://github.com/tu-graz/kanboard-cli/compare/v0.4.0...v0.4.4
[0.4.0]: https://github.com/tu-graz/kanboard-cli/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/tu-graz/kanboard-cli/releases/tag/v0.3.0
[0.2.0]: https://github.com/tu-graz/kanboard-cli/releases/tag/v0.2.0
[0.1.0]: https://github.com/tu-graz/kanboard-cli/releases/tag/v0.1.0
