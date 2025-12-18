# makewaffle CLI Guide

The `makewaffle` command scaffolds a new WAFFLE application. It generates a complete project structure with all the files needed to start building a WAFFLE-based web service.

## Installation

```bash
go install github.com/dalemusser/waffle/cmd/makewaffle@latest
```

Ensure your Go bin directory (often `~/go/bin`) is on your PATH. See [Setting Your PATH for Go and WAFFLE](./set-path.md) if you need help with this.

## Usage

```
makewaffle new <appname> --module <module-path> [options]
```

## Getting Help

`makewaffle` supports several ways to view help:

### Top-level help

```bash
makewaffle
makewaffle --help
makewaffle -h
makewaffle help
```

Output:

```
WAFFLE CLI (makewaffle)

Usage:
  makewaffle new <appname> --module <module-path>

Options:
  --module         Go module path for the new app (required)
  --waffle-version Version of waffle to require (optional)
  --go-version     Go language version (default: 1.21)
  --template       Template to use: full (default: full)
  --force          Scaffold into existing directory

Example:
  makewaffle new myapp --module github.com/you/myapp
```

### Subcommand help

```bash
makewaffle new --help
```

## Arguments and Options

| Argument/Option | Required | Description |
|---|---|---|
| `<appname>` | Yes | Directory name for the new project. Must contain only letters, digits, and underscores. Cannot start with a digit. |
| `--module` | Yes | Go module path (e.g., `github.com/you/myapp`) |
| `--waffle-version` | No | Specific WAFFLE version to require (e.g., `v0.1.18`). If omitted, `go mod tidy` will resolve the latest. |
| `--go-version` | No | Go language version for `go.mod` (default: `1.21`) |
| `--template` | No | Template to use: `full` (default: `full`). Future versions may support `minimal` and `custom`. |
| `--force` | No | Allow scaffolding into an existing directory |

## Creating a New Project

To create a new project:

```bash
makewaffle new myservice --module github.com/example/myservice
```

Output:

```
Creating WAFFLE app "myservice" with module "github.com/example/myservice"
Done!

Next steps:
  cd myservice
  go mod tidy
  go run ./cmd/myservice
  go to http://localhost:8080 in web browser
```

## Generated Project Structure

`makewaffle` generates the following structure:

```
myservice/
├── go.mod                              # Go module definition
├── cmd/
│   └── myservice/
│       └── main.go                     # Application entry point
└── internal/
    ├── app/
    │   ├── bootstrap/                  # WAFFLE lifecycle hooks
    │   │   ├── hooks.go                # Hooks struct wiring
    │   │   ├── config.go               # LoadConfig & ValidateConfig
    │   │   ├── appconfig.go            # AppConfig struct
    │   │   ├── db.go                   # ConnectDB & EnsureSchema
    │   │   ├── dbdeps.go               # DBDeps struct
    │   │   ├── startup.go              # Startup hook
    │   │   ├── routes.go               # BuildHandler (HTTP routing)
    │   │   └── shutdown.go             # Shutdown hook
    │   ├── features/                   # Feature modules
    │   │   └── README.md
    │   ├── resources/                  # Embedded files (templates, images)
    │   │   └── README.md
    │   ├── system/                     # App-specific shared utilities
    │   │   └── README.md
    │   ├── store/                      # Data persistence layer
    │   │   └── README.md
    │   └── policy/                     # Authorization logic
    │       └── README.md
    └── domain/
        └── models/                     # Domain models
            └── README.md
```

### Bootstrap Files

| File | Purpose |
|---|---|
| `hooks.go` | Wires lifecycle functions into `app.Hooks` struct |
| `config.go` | `LoadConfig` and `ValidateConfig` functions |
| `appconfig.go` | `AppConfig` struct for app-specific configuration |
| `db.go` | `ConnectDB` and `EnsureSchema` functions |
| `dbdeps.go` | `DBDeps` struct for database clients |
| `startup.go` | `Startup` hook for one-time initialization |
| `routes.go` | `BuildHandler` function for HTTP routing |
| `shutdown.go` | `Shutdown` hook for graceful cleanup |

### Directory READMEs

Each empty directory contains a README.md explaining its purpose:

| Directory | Purpose |
|---|---|
| `features/` | Self-contained feature modules with routes, handlers, and templates |
| `resources/` | Embedded files via `go:embed` (shared templates, images, JSON) |
| `system/` | App-specific shared utilities used across features |
| `store/` | Data persistence/repository layer |
| `policy/` | Authorization and business rule logic |
| `domain/models/` | Domain models and business entities |

## Scaffolding into an Existing Directory

By default, `makewaffle` fails if the target directory exists:

```bash
makewaffle new myservice --module github.com/example/myservice
# Error: mkdir myservice: file exists
```

Use `--force` to scaffold into an existing directory:

```bash
makewaffle new myservice --module github.com/example/myservice --force
```

This is useful when:
- You've cloned an empty GitHub repository
- You want to re-scaffold over an existing project

**Warning:** Files with the same names as generated files will be overwritten.

## Specifying WAFFLE Version

To pin a specific WAFFLE version in `go.mod`:

```bash
makewaffle new myservice --module github.com/example/myservice --waffle-version v0.1.18
```

This generates a `go.mod` with explicit dependencies:

```
module github.com/example/myservice

go 1.21

require (
    github.com/dalemusser/waffle v0.1.18
    github.com/go-chi/chi/v5 v5.2.3
    go.uber.org/zap v1.27.1
)
```

If `--waffle-version` is omitted, `go.mod` contains only the module declaration and Go version. Run `go mod tidy` to resolve dependencies.

## Specifying Go Version

To use a different Go version:

```bash
makewaffle new myservice --module github.com/example/myservice --go-version 1.22
```

The default is `1.21`, which is the minimum version required for WAFFLE.

## App Name Validation

The `<appname>` must:
- Contain only letters (a-z, A-Z), digits (0-9), and underscores (_)
- Not start with a digit
- Not be empty

Valid examples:
- `myservice`
- `my_service`
- `MyService`
- `service1`

Invalid examples:
- `1service` (starts with digit)
- `my-service` (contains hyphen)
- `my.service` (contains period)

## Quick Start

```bash
# Install makewaffle
go install github.com/dalemusser/waffle/cmd/makewaffle@latest

# Create a new project
makewaffle new hello --module github.com/you/hello

# Enter the project
cd hello

# Download dependencies
go mod tidy

# Run the app
go run ./cmd/hello

# Visit http://localhost:8080
```

## Alternative: wafflectl

WAFFLE also provides `wafflectl` as an alternative command name, following the `*ctl` naming convention (like `kubectl`):

```bash
go install github.com/dalemusser/waffle/cmd/wafflectl@latest
wafflectl new myservice --module github.com/example/myservice
```

Both commands are identical in functionality.

## See Also

- [How to Write Your First WAFFLE Service](./first-service.md) — Step-by-step tutorial
- [WAFFLE Quickstart Guide](./quickstart.md) — Quick overview
- [Development Guide](../development/README.md) — Detailed reference
