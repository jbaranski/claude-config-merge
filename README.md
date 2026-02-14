# claude-config-merge

Syncs Claude configuration from a master config directory into your local `~/.claude/` setup. Merges `settings.json`, copies agents and skills, and manages backups.

## Config

Create `~/.claude-config-merge.json`:

```json
{
  "configDir": "/path/to/your/claude/configs"
}
```

`configDir` mirrors the structure of `~`. Expected layout:

```
<configDir>/
└── .claude/
    ├── settings.json   ← master settings (merged into ~/.claude/settings.json)
    ├── agents/         ← synced to ~/.claude/agents/
    └── skills/         ← synced to ~/.claude/skills/
```

## Setup

```sh
make deps    # install goimports and golangci-lint
make all     # format, lint, test, build
```

## Install

```sh
make install
```

Builds the binary and copies it to `~/.local/bin/claude-config-merge`. Ensure `~/.local/bin` is on your `PATH`:

```sh
# add to ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
```

## Usage

```
claude-config-merge [-config FILE] <command> [-f]
```

Run with no arguments (or `-h`) to print help:

```sh
claude-config-merge
claude-config-merge -h
```

### Commands

| Command         | Description                                                             |
|-----------------|-------------------------------------------------------------------------|
| `settings`      | Merge master `settings.json` into `~/.claude/settings.json`            |
| `agents`        | Copy agent files from `configDir/.claude/agents` to `~/.claude/agents` |
| `skills`        | Copy skill files from `configDir/.claude/skills` to `~/.claude/skills` |
| `all`           | Run `settings`, `agents`, and `skills` in sequence                     |
| `cleanup-bak`   | Delete `settings.json.*.bak` backup files from `~/.claude/`            |
| `help`          | Print help                                                              |

### Flags

| Flag             | Applies to                  | Effect                                                              |
|------------------|-----------------------------|---------------------------------------------------------------------|
| `-f`             | `settings`, `agents`, `skills`, `all` | Force overwrite. For `settings`: master wins on conflict. For `agents`/`skills`: overwrite existing files. |
| `-config FILE`   | all commands                | Use a custom config file instead of `~/.claude-config-merge.json`  |

### Examples

```sh
claude-config-merge settings                          # merge settings, keep local on conflict
claude-config-merge settings -f                       # merge settings, master wins on conflict
claude-config-merge agents                            # copy new agents, skip existing
claude-config-merge skills -f                         # copy skills, overwrite existing
claude-config-merge all                               # sync everything
claude-config-merge all -f                            # sync everything, force overwrite
claude-config-merge cleanup-bak                       # delete .bak files from ~/.claude/
claude-config-merge -config ~/my-config.json all      # use custom config file
```

## Make Commands

| Command               | Description                              |
|-----------------------|------------------------------------------|
| `make deps`           | Install dev dependencies (run first)     |
| `make all`            | Format, lint, test, and build            |
| `make build`          | Build the binary                         |
| `make install`        | Build and install to `~/.local/bin`      |
| `make test`           | Run tests with race detector             |
| `make coverage`       | Run tests and generate coverage report   |
| `make coverage-check` | Fail if coverage is below 80%            |
| `make lint`           | Run golangci-lint                        |
| `make fmt`            | Format code with gofmt and goimports     |
| `make check`          | Format, lint, and coverage-check         |
| `make tidy`           | Tidy and verify go modules               |
| `make clean`          | Remove build artifacts                   |

## Debugger (VS Code)

Requires the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go).

1. Open the project in VS Code
2. Click in the gutter next to any line to set a breakpoint
3. Press `F5` (or **Run > Start Debugging**) — uses the `claude-config-merge` launch config
4. Execution pauses at your breakpoint; use `F10` step over, `F11` step into, `F5` continue

The launch config is at `.vscode/launch.json` and passes your `~/.claude-config-merge.json` automatically. To use a different config file, edit the `args` in `launch.json`.
