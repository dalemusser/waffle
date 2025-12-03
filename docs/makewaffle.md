# makewaffle CLI Guide

The `makewaffle` command scaffolds a new WAFFLE application. It supports two main scenarios:
- Creating a new WAFFLE service in a new directory
- Scaffolding into an existing directory using the `--force` flag

## Usage

```
makewaffle new <appname> --module <module-path> [--force]
```

### Arguments
- `<appname>`: The directory where the skeleton will be created. Must be a valid name (letters, digits, and underscores).
- `--module`: Required. The Go module path for your new service (e.g. `github.com/example/hello`).
- `--force`: Optional. Allows scaffolding into an existing directory.

## Creating a New Project

To create a new project in a new directory, simply run:

```
makewaffle new myservice --module github.com/example/myservice
```

This generates a new directory `myservice` containing:
- `go.mod`
- `cmd/myservice/main.go`
- `internal/app/bootstrap/...`
- `internal/app/features`
- `internal/app/store`
- `internal/app/policy`
- `internal/domain/models`

## Scaffolding into an Existing Directory

If the target directory already exists, `makewaffle` will error unless you use `--force`:

```
makewaffle new myservice --module github.com/example/myservice --force
```

This allows you to scaffold into a directory that already exists (for example, if you cloned a GitHub repository ahead of time).

**Note:** The tool may overwrite files that have the same names as generated boilerplate.

## Behavior Summary

- Running `makewaffle new myservice ...` creates (or reuses with `--force`) the directory `myservice`.
- Scaffolding always writes files _inside_ the `<appname>` directory.
- `appname` must not be `.`. The current version does not support using `.` as an appname.
