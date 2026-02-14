# claude-config-merge

Merges a master `settings.json` into `~/.claude/settings.json`. New keys from master are added; conflicting keys keep their local value. A timestamped backup is created before any changes are written.

## Config

Create `~/.claude-config-merge.json`:

```json
{
  "configDir": "/path/to/your/claude/configs"
}
```

`configDir` is the base directory and mirrors the structure of `~`. The master settings file must exist at `<configDir>/.claude/settings.json`.

## Setup

```sh
make deps    # install goimports and golangci-lint
make all     # format, lint, test, build
```

## Run

```sh
make build
./claude-config-merge
```

To use a different config file:

```sh
./claude-config-merge -config /path/to/config.json
```

## Make Commands

| Command              | Description                              |
|----------------------|------------------------------------------|
| `make deps`          | Install dev dependencies (run first)     |
| `make all`           | Format, lint, test, and build            |
| `make build`         | Build the binary                         |
| `make test`          | Run tests with race detector             |
| `make coverage`      | Run tests and generate coverage report   |
| `make coverage-check`| Fail if coverage is below 80%            |
| `make lint`          | Run golangci-lint                        |
| `make fmt`           | Format code with gofmt and goimports     |
| `make check`         | Format, lint, and coverage-check         |
| `make tidy`          | Tidy and verify go modules               |
| `make clean`         | Remove build artifacts                   |

## Debugger (VS Code)

Requires the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go).

1. Open the project in VS Code
2. Click in the gutter next to any line to set a breakpoint
3. Press `F5` (or **Run > Start Debugging**) — uses the `claude-config-merge` launch config
4. Execution pauses at your breakpoint; use `F10` step over, `F11` step into, `F5` continue

The launch config is at `.vscode/launch.json` and passes your `~/.claude-config-merge.json` automatically. To use a different config file, edit the `args` in `launch.json`.

## Output

```
Backup created: /Users/you/.claude/settings.json.20260214T103045.000.bak

Conflicts (local value kept):
  CONFLICT  theme                                     master=dark  local=light

Done. Keys added: 3  |  Conflicts skipped: 1
Written to: /Users/you/.claude/settings.json
```
