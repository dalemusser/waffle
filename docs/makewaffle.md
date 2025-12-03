# 🧇 WAFFLE CLI — `makewaffle`
### Complete Guide to Scaffolding WAFFLE Applications

`makewaffle` is the official WAFFLE CLI for generating new services.  
It creates the full application skeleton:

- `go.mod`
- `cmd/<appname>/main.go`
- `internal/app/...`
- `internal/domain/models/...`
- `/policy`, `/store`, `/features`

This document explains **all the ways you can run `makewaffle`**, the meaning of each form, and which scenarios each one is best suited for.

---

# 📦 1. Installation

Install the CLI:

```bash
go install github.com/dalemusser/waffle/cmd/makewaffle@latest
```

Make sure your Go bin directory is on your PATH:

```bash
export PATH="$HOME/go/bin:$PATH"
```

If you want the CLI with an industry-standard name instead:

```bash
go install github.com/dalemusser/waffle/cmd/wafflectl@latest
```

Both CLIs behave the same way — only the command name differs.

---

# 🚀 2. Basic Usage

The standard command form is:

```bash
makewaffle new <appname> --module <module-path>
```

Example:

```bash
makewaffle new hello --module github.com/you/hello
```

Behavior:

- Creates a **new directory** named `<appname>`.
- Scaffolds the full WAFFLE project inside it.

Resulting structure:

```
hello/
  go.mod
  cmd/hello/main.go
  internal/app/...
  internal/domain/models/...
  internal/app/features/...
  internal/app/policy/...
  internal/app/store/...
```

---

# 📁 3. Directory Behavior (Important!)

This is where the CLI becomes powerful — and where several modes exist.

## 3.1 Default: `<appname>` creates a subdirectory

From any directory:

```bash
makewaffle new hello --module github.com/you/hello
```

Creates:

```
./hello/
```

The *name of the current directory does not matter.*  
Only whether `./hello` already exists.

---

## 3.2 If `./<appname>` already exists → error (unless `--force` is used)

```bash
makewaffle new hello --module github.com/you/hello
```

If `./hello` exists:

```
mkdir hello: file exists
```

Unless you explicitly tell WAFFLE to continue (see `--force` below).

---

## 3.3 Scaffolding **in the current directory** using `.`

This is the most natural approach for **GitHub-first workflows**.

Example workflow:

```bash
git clone git@github.com:you/hello.git
cd hello
makewaffle new . --module github.com/you/hello
```

Behavior:

- Does **not** create a subdirectory.
- Writes all scaffold files **into the current directory**.
- `cmd/<name>/` directory is derived from the module path, not `"."`.

This is exactly what you want when starting inside a repo you created on GitHub.

---

## 3.4 Forcing scaffold into an existing directory (`--force`)

If you want WAFFLE to scaffold into a directory that already exists:

```bash
makewaffle new hello --module github.com/you/hello --force
```

Without `--force`:

- WAFFLE refuses to scaffold into existing directories.

With `--force`:

- WAFFLE writes the scaffold **into that existing directory**.

### ⚠️ Safety note

`--force` may overwrite scaffolded files if they already exist.  
It will **not** delete unrelated existing files.

---

# 🧠 4. Common Usage Scenarios

## Scenario A — Create a new WAFFLE project from scratch

```bash
makewaffle new api --module github.com/you/api
cd api
go mod tidy
go run ./cmd/api
```

Best used when starting a new service with no pre-existing repo.

---

## Scenario B — GitHub-first workflow (RECOMMENDED)

Your workflow:

1. Create repo on GitHub  
2. Add README, LICENSE, .gitignore (Go)  
3. Clone locally:

```bash
git clone git@github.com:you/hello.git
cd hello
makewaffle new . --module github.com/you/hello
go mod tidy
go run ./cmd/hello
```

Perfect for devs who start projects through GitHub’s UI.

---

## Scenario C — Scaffolding into a partially existing directory

If your repo already contains files like:

```
README.md
LICENSE
.gitignore
```

You can still scaffold WAFFLE inside it:

```bash
makewaffle new . --module github.com/you/hello --force
```

or:

```bash
makewaffle new hello --module github.com/you/hello --force
```

Use `--force` only when you intend to proceed even if the directory isn’t empty.

---

## Scenario D — Monorepo usage

Inside a monorepo structure:

```
monorepo/
  services/
```

You can create nested WAFFLE services:

```bash
cd monorepo/services
makewaffle new auth --module github.com/org/monorepo/services/auth
```

Result:

```
monorepo/services/auth/
```

---

# ⚙️ 5. Flags Reference

### `--module <path>` (required)

Defines the Go module path.  
Determines import paths and the name of `cmd/<appname>` in dot-mode.

---

### `--force` (optional)

Allows scaffolding into an existing directory.

Without `--force`, WAFFLE errors when the target exists.

---

### `--waffle-version <vX.Y.Z>` (optional)

Pins the WAFFLE version in the generated `go.mod`.

Example:

```bash
makewaffle new hello --module github.com/you/hello --waffle-version v0.1.4
```

---

### `--go-version <1.xx>` (optional)

Overrides the Go version used in the generated `go.mod`.

Default is currently `1.21`.

---

# 🔍 6. Examples

### Create new project in new directory:

```bash
makewaffle new hello --module github.com/you/hello
```

### Scaffold into current directory:

```bash
makewaffle new . --module github.com/you/hello
```

### Scaffold into existing directory (force):

```bash
makewaffle new hello --module github.com/you/hello --force
```

### Pin version of WAFFLE:

```bash
makewaffle new tool --module github.com/you/tool --waffle-version v0.1.4
```

---

# 🧇 7. Best Practices

- Use `makewaffle new .` when starting from a GitHub-created empty repo.
- Use `--force` carefully; it may overwrite scaffolded files.
- Always run:

  ```bash
  go mod tidy
  ```

  after scaffolding.

- Use pinned WAFFLE versions for production services.
- Test your new service immediately:

  ```bash
  go run ./cmd/<appname>
  ```

---

# 🎉 8. Summary

`makewaffle` now supports:

- Scaffolding into new directories  
- Scaffolding “here” using `.`  
- Forced scaffolding into existing directories  
- Version pinning  
- Custom Go versions  
- GitHub-first workflows  
- Monorepos  
- Conventional scratch project creation  

This flexibility makes WAFFLE’s CLI suitable for **beginners**, **professionals**, and **complex real-world environments**.

