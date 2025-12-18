# Terminal Guide for macOS

This guide covers the basics of using the Terminal on macOS for WAFFLE development. If you're new to the command line, this will help you get started.

---

## Opening Terminal

There are several ways to open Terminal on macOS:

### Spotlight Search (Fastest)
1. Press **Cmd + Space** to open Spotlight
2. Type `Terminal`
3. Press **Enter**

### Applications Folder
1. Open **Finder**
2. Go to **Applications** > **Utilities**
3. Double-click **Terminal**

### Launchpad
1. Click the **Launchpad** icon in the Dock
2. Type `Terminal` in the search field
3. Click the **Terminal** icon

---

## Understanding the Terminal Window

When you open Terminal, you'll see a prompt that looks something like:

```
username@MacBook ~ %
```

This shows:
- `username` — Your macOS username
- `MacBook` — Your computer's name
- `~` — Your current directory (~ means your home folder)
- `%` — The prompt symbol (indicates zsh shell)

If you see `$` instead of `%`, you're using bash shell.

---

## Essential Commands

### Navigation

| Command | Description | Example |
|---------|-------------|---------|
| `pwd` | Print working directory (where you are) | `pwd` |
| `ls` | List files and folders | `ls` |
| `ls -la` | List all files with details | `ls -la` |
| `cd <folder>` | Change directory | `cd Documents` |
| `cd ..` | Go up one directory | `cd ..` |
| `cd ~` | Go to home directory | `cd ~` |
| `cd -` | Go to previous directory | `cd -` |

### File Operations

| Command | Description | Example |
|---------|-------------|---------|
| `mkdir <name>` | Create a directory | `mkdir myproject` |
| `touch <file>` | Create an empty file | `touch notes.txt` |
| `cp <src> <dest>` | Copy a file | `cp file.txt backup.txt` |
| `mv <src> <dest>` | Move or rename a file | `mv old.txt new.txt` |
| `rm <file>` | Delete a file | `rm unwanted.txt` |
| `rm -r <folder>` | Delete a folder and contents | `rm -r oldfolder` |
| `cat <file>` | Display file contents | `cat readme.md` |

### Getting Help

| Command | Description | Example |
|---------|-------------|---------|
| `man <command>` | Show manual for a command | `man ls` |
| `<command> --help` | Show help for a command | `go --help` |

Press **Q** to exit the manual viewer.

---

## Keyboard Shortcuts

These shortcuts work in Terminal and make command-line work faster:

| Shortcut | Action |
|----------|--------|
| **Ctrl + C** | Cancel current command / stop running process |
| **Ctrl + L** | Clear the screen (same as `clear` command) |
| **Ctrl + A** | Move cursor to beginning of line |
| **Ctrl + E** | Move cursor to end of line |
| **Ctrl + U** | Delete from cursor to beginning of line |
| **Ctrl + K** | Delete from cursor to end of line |
| **Ctrl + W** | Delete word before cursor |
| **Ctrl + R** | Search command history |
| **Tab** | Auto-complete file/folder names |
| **Up Arrow** | Previous command from history |
| **Down Arrow** | Next command from history |
| **Cmd + K** | Clear terminal buffer (macOS specific) |
| **Cmd + T** | Open new terminal tab |
| **Cmd + N** | Open new terminal window |

---

## Tab Completion

Tab completion saves typing and prevents errors. Start typing a command, file, or folder name, then press **Tab**:

```bash
cd Docu<Tab>
```

This expands to:

```bash
cd Documents/
```

If there are multiple matches, press **Tab** twice to see all options.

---

## Working with WAFFLE Projects

### Creating a New Project

```bash
# Navigate to where you want to create the project
cd ~/Documents

# Create a new WAFFLE project
makewaffle new myservice --module github.com/you/myservice

# Enter the project directory
cd myservice

# Download dependencies
go mod tidy

# Run the application
go run ./cmd/myservice
```

### Common WAFFLE Development Commands

```bash
# Run your application
go run ./cmd/myservice

# Build a binary
go build -o myservice ./cmd/myservice

# Run tests
go test ./...

# Format code
go fmt ./...

# Check for issues
go vet ./...

# Update dependencies
go mod tidy
```

---

## Running Background Processes

### Run in Background

Add `&` at the end of a command to run it in the background:

```bash
go run ./cmd/myservice &
```

### Stop a Running Process

Press **Ctrl + C** to stop a process running in the foreground.

For background processes:

```bash
# List running jobs
jobs

# Bring to foreground
fg

# Then press Ctrl + C to stop
```

---

## Environment Variables

### View Environment Variables

```bash
# View all environment variables
env

# View a specific variable
echo $PATH
echo $GOPATH
```

### Set Environment Variables

```bash
# Set for current session only
export WAFFLE_HTTP_PORT=9090

# Use in a command
WAFFLE_ENV=prod go run ./cmd/myservice
```

For permanent changes, see [Setting Your PATH](./set-path.md).

---

## File Paths

macOS uses forward slashes (`/`) for paths:

| Path | Description |
|------|-------------|
| `/` | Root of the filesystem |
| `~` | Your home folder (`/Users/username`) |
| `.` | Current directory |
| `..` | Parent directory |
| `~/Documents` | Documents folder in your home |
| `./myfile.txt` | File in current directory |

### Examples

```bash
# Absolute path (starts from root)
cd /Users/dale/Documents/myproject

# Relative path (from current location)
cd ../otherproject

# Home-relative path
cd ~/go/bin
```

---

## Editing Files

### Using nano (Beginner-Friendly)

```bash
nano filename.txt
```

- **Ctrl + O** — Save (write Out)
- **Ctrl + X** — Exit
- **Ctrl + K** — Cut line
- **Ctrl + U** — Paste line
- **Ctrl + W** — Search

### Using vim (Advanced)

```bash
vim filename.txt
```

- Press **i** to enter insert mode (type text)
- Press **Esc** to exit insert mode
- Type `:w` and **Enter** to save
- Type `:q` and **Enter** to quit
- Type `:wq` and **Enter** to save and quit
- Type `:q!` and **Enter** to quit without saving

### Opening in VSCode

```bash
# Open current directory in VSCode
code .

# Open a specific file
code myfile.go
```

---

## Viewing Output

### Scrolling

- **Scroll up**: Swipe up on trackpad, or **Shift + Page Up**
- **Scroll down**: Swipe down on trackpad, or **Shift + Page Down**

### Piping and Filtering

```bash
# Pipe output to another command
go test ./... | grep FAIL

# View output page by page
cat longfile.txt | less

# Save output to a file
go build ./... 2>&1 | tee build.log
```

### Redirecting Output

```bash
# Redirect output to a file (overwrite)
go run ./cmd/myservice > output.log

# Redirect output to a file (append)
go run ./cmd/myservice >> output.log

# Redirect errors to a file
go run ./cmd/myservice 2> errors.log

# Redirect both output and errors
go run ./cmd/myservice > all.log 2>&1
```

---

## Command History

```bash
# View recent commands
history

# Search history interactively
# Press Ctrl + R, then type to search

# Run previous command
!!

# Run command by history number
!42
```

---

## Multiple Commands

```bash
# Run commands sequentially (always runs all)
command1 ; command2 ; command3

# Run next command only if previous succeeds
command1 && command2 && command3

# Run next command only if previous fails
command1 || command2
```

Example:

```bash
# Build and run only if build succeeds
go build -o myservice ./cmd/myservice && ./myservice
```

---

## Checking What's Running

```bash
# Show your running processes
ps

# Show all processes
ps aux

# Find a specific process
ps aux | grep myservice

# Show what's using a port
lsof -i :8080
```

---

## Troubleshooting

### Command Not Found

If you see `command not found`:

1. Check spelling
2. Verify the program is installed
3. Check your PATH (see [Setting Your PATH](./set-path.md))

```bash
# Check if a command exists
which go
which makewaffle

# Check your PATH
echo $PATH
```

### Permission Denied

If you see `Permission denied`:

```bash
# Make a file executable
chmod +x myfile.sh

# Run with elevated privileges (use carefully)
sudo command
```

### Process Won't Stop

If **Ctrl + C** doesn't work:

```bash
# Find the process ID
ps aux | grep myservice

# Force kill by process ID
kill -9 <pid>
```

---

## Tips for WAFFLE Development

1. **Use Tab completion** — Type partial names and press Tab
2. **Use command history** — Press Up Arrow to repeat commands
3. **Open VSCode from Terminal** — Run `code .` to ensure correct PATH
4. **Keep Terminal open** — Watch server logs while developing
5. **Use multiple tabs** — One for running the server, one for commands

---

## See Also

- [Setting Your PATH](./set-path.md) — Configure Go and WAFFLE commands
- [makewaffle CLI Guide](./makewaffle.md) — Scaffold new projects
- [How to Write Your First WAFFLE Service](./first-service.md) — Step-by-step tutorial
