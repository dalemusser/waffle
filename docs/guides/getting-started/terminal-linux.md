# Terminal Guide for Linux

This guide covers the basics of using the terminal on Linux for WAFFLE development. While this guide focuses on Ubuntu, most commands work across all Linux distributions.

---

## Opening the Terminal

### Ubuntu / GNOME

**Keyboard shortcut (fastest):**
- Press **Ctrl + Alt + T**

**From Activities:**
1. Press the **Super** key (Windows key)
2. Type `Terminal`
3. Click the **Terminal** icon

**From Applications menu:**
1. Click **Activities** or the application grid
2. Find **Terminal** in the list

### Other Desktop Environments

| Desktop | Method |
|---------|--------|
| KDE Plasma | Ctrl + Alt + T, or search for "Konsole" |
| XFCE | Ctrl + Alt + T, or search for "Terminal" |
| MATE | Ctrl + Alt + T, or search for "MATE Terminal" |
| i3/Sway | Mod + Enter (typically) |

---

## Understanding the Terminal Window

When you open the terminal, you'll see a prompt like:

```
username@hostname:~$
```

This shows:
- `username` — Your Linux username
- `hostname` — Your computer's name
- `~` — Current directory (~ means home folder)
- `$` — Regular user prompt (# means root)

---

## Essential Commands

### Navigation

| Command | Description | Example |
|---------|-------------|---------|
| `pwd` | Print working directory | `pwd` |
| `ls` | List files and folders | `ls` |
| `ls -la` | List all files with details | `ls -la` |
| `ls -lh` | List with human-readable sizes | `ls -lh` |
| `cd <folder>` | Change directory | `cd Documents` |
| `cd ..` | Go up one directory | `cd ..` |
| `cd ~` | Go to home directory | `cd ~` |
| `cd -` | Go to previous directory | `cd -` |
| `cd /` | Go to root directory | `cd /` |

### File Operations

| Command | Description | Example |
|---------|-------------|---------|
| `mkdir <name>` | Create a directory | `mkdir myproject` |
| `mkdir -p a/b/c` | Create nested directories | `mkdir -p src/app/handlers` |
| `touch <file>` | Create an empty file | `touch notes.txt` |
| `cp <src> <dest>` | Copy a file | `cp file.txt backup.txt` |
| `cp -r <src> <dest>` | Copy a directory | `cp -r project project-backup` |
| `mv <src> <dest>` | Move or rename | `mv old.txt new.txt` |
| `rm <file>` | Delete a file | `rm unwanted.txt` |
| `rm -r <folder>` | Delete folder and contents | `rm -r oldfolder` |
| `rm -rf <folder>` | Force delete (careful!) | `rm -rf oldfolder` |
| `cat <file>` | Display file contents | `cat readme.md` |
| `less <file>` | View file with scrolling | `less longfile.txt` |
| `head <file>` | Show first 10 lines | `head log.txt` |
| `tail <file>` | Show last 10 lines | `tail log.txt` |
| `tail -f <file>` | Follow file (live updates) | `tail -f server.log` |

### Getting Help

| Command | Description | Example |
|---------|-------------|---------|
| `man <command>` | Show manual page | `man ls` |
| `<command> --help` | Show help | `go --help` |
| `which <command>` | Show command location | `which go` |
| `type <command>` | Show command type | `type ls` |

Press **Q** to exit man pages or less.

---

## Keyboard Shortcuts

These shortcuts work in most Linux terminals:

| Shortcut | Action |
|----------|--------|
| **Ctrl + C** | Cancel current command / stop process |
| **Ctrl + Z** | Suspend current process (use `fg` to resume) |
| **Ctrl + D** | Exit terminal / end input |
| **Ctrl + L** | Clear screen (same as `clear`) |
| **Ctrl + A** | Move cursor to beginning of line |
| **Ctrl + E** | Move cursor to end of line |
| **Ctrl + U** | Delete from cursor to beginning of line |
| **Ctrl + K** | Delete from cursor to end of line |
| **Ctrl + W** | Delete word before cursor |
| **Ctrl + Y** | Paste deleted text (yank) |
| **Ctrl + R** | Search command history |
| **Ctrl + G** | Cancel history search |
| **Tab** | Auto-complete |
| **Tab Tab** | Show all completions |
| **Up Arrow** | Previous command |
| **Down Arrow** | Next command |
| **Ctrl + Shift + C** | Copy selected text |
| **Ctrl + Shift + V** | Paste |
| **Ctrl + Shift + T** | New terminal tab |
| **Ctrl + Shift + N** | New terminal window |
| **Alt + 1-9** | Switch to tab N |

---

## Tab Completion

Tab completion saves typing and prevents errors. Start typing and press **Tab**:

```bash
cd Docu<Tab>
```

Expands to:

```bash
cd Documents/
```

Press **Tab** twice to see all possible completions if there are multiple matches.

Tab completion works for:
- File and directory names
- Command names
- Command options (in bash with bash-completion)
- Git branches, remotes, etc.

---

## Working with WAFFLE Projects

### Creating a New Project

```bash
# Navigate to where you want to create the project
cd ~/projects

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

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Format code
go fmt ./...

# Check for issues
go vet ./...

# Update dependencies
go mod tidy
```

---

## Running Processes

### Foreground vs Background

```bash
# Run in foreground (blocks terminal)
go run ./cmd/myservice

# Run in background (add &)
go run ./cmd/myservice &

# Run and ignore hangup signal
nohup go run ./cmd/myservice &
```

### Managing Background Processes

```bash
# List background jobs
jobs

# Bring job to foreground
fg %1

# Send job to background
bg %1

# Suspend foreground process
# Press Ctrl + Z

# Resume suspended process in background
bg
```

### Stop a Running Process

Press **Ctrl + C** to stop a foreground process.

For background processes:

```bash
# Find process ID
ps aux | grep myservice

# Kill by PID
kill <pid>

# Force kill
kill -9 <pid>

# Kill by name
pkill myservice

# Kill all matching
killall myservice
```

---

## Environment Variables

### View Environment Variables

```bash
# View all environment variables
env

# View sorted
env | sort

# View a specific variable
echo $PATH
echo $GOPATH
echo $HOME

# View with printenv
printenv PATH
```

### Set Environment Variables

```bash
# Set for current session only
export WAFFLE_HTTP_PORT=9090

# Use for a single command
WAFFLE_ENV=prod go run ./cmd/myservice

# Unset a variable
unset WAFFLE_HTTP_PORT
```

### Set Permanently

Add to your shell configuration file:

**For bash (~/.bashrc):**
```bash
export WAFFLE_HTTP_PORT=9090
```

**For zsh (~/.zshrc):**
```bash
export WAFFLE_HTTP_PORT=9090
```

Then reload:
```bash
source ~/.bashrc  # or source ~/.zshrc
```

For PATH configuration, see [Setting Your PATH](./set-path.md).

---

## File Paths

Linux uses forward slashes (`/`) for paths:

| Path | Description |
|------|-------------|
| `/` | Root of the filesystem |
| `~` | Your home folder (`/home/username`) |
| `.` | Current directory |
| `..` | Parent directory |
| `~/Documents` | Documents in your home |
| `./myfile.txt` | File in current directory |
| `/etc` | System configuration |
| `/var/log` | System logs |
| `/tmp` | Temporary files |

### Examples

```bash
# Absolute path (starts from root)
cd /home/dale/projects/myservice

# Relative path (from current location)
cd ../otherproject

# Home-relative path
cd ~/go/bin

# Current directory
./myservice
```

---

## File Permissions

Linux has a permission system for files and directories.

### View Permissions

```bash
ls -la
```

Output example:
```
-rwxr-xr-x 1 dale dale 12345 Dec 15 10:00 myservice
drwxr-xr-x 2 dale dale  4096 Dec 15 10:00 src
```

The first column shows permissions:
- First character: type (`-` = file, `d` = directory)
- Characters 2-4: owner permissions (rwx)
- Characters 5-7: group permissions (rwx)
- Characters 8-10: others permissions (rwx)

Where: `r` = read, `w` = write, `x` = execute

### Change Permissions

```bash
# Make a file executable
chmod +x myservice

# Set specific permissions (owner: rwx, group: rx, others: rx)
chmod 755 myservice

# Make readable/writable by owner only
chmod 600 secrets.txt
```

### Change Ownership

```bash
# Change owner
sudo chown dale myfile.txt

# Change owner and group
sudo chown dale:dale myfile.txt

# Recursive
sudo chown -R dale:dale myproject/
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
- **Ctrl + G** — Help

### Using vim (Advanced)

```bash
vim filename.txt
```

Vim has modes:
- **Normal mode** — Navigate and commands (default)
- **Insert mode** — Type text (press `i` to enter)
- **Command mode** — Save, quit (press `:`)

Basic vim:
- Press **i** to enter insert mode
- Press **Esc** to return to normal mode
- Type `:w` and Enter to save
- Type `:q` and Enter to quit
- Type `:wq` and Enter to save and quit
- Type `:q!` and Enter to quit without saving

### Opening in VSCode

```bash
# Open current directory
code .

# Open a specific file
code myfile.go
```

---

## Viewing and Filtering Output

### Piping

The pipe (`|`) sends output from one command to another:

```bash
# Filter output
go test ./... | grep FAIL

# Count lines
ls -la | wc -l

# Sort output
env | sort

# View page by page
cat longfile.txt | less
```

### Redirecting Output

```bash
# Redirect stdout to a file (overwrite)
go run ./cmd/myservice > output.log

# Redirect stdout to a file (append)
go run ./cmd/myservice >> output.log

# Redirect stderr to a file
go run ./cmd/myservice 2> errors.log

# Redirect both stdout and stderr
go run ./cmd/myservice > all.log 2>&1

# Redirect to /dev/null (discard)
go run ./cmd/myservice > /dev/null 2>&1
```

### Useful Filters

```bash
# Search for pattern
grep "error" logfile.txt

# Search recursively in files
grep -r "TODO" ./src

# Search case-insensitive
grep -i "error" logfile.txt

# Show lines with context
grep -C 3 "error" logfile.txt

# Filter columns
cat file.txt | awk '{print $1}'

# Replace text
cat file.txt | sed 's/old/new/g'
```

---

## Command History

```bash
# View history
history

# View last N commands
history 20

# Search history interactively
# Press Ctrl + R, type search term

# Run previous command
!!

# Run command by number
!42

# Run last command starting with 'go'
!go

# Clear history
history -c
```

---

## Multiple Commands

```bash
# Run sequentially (always runs all)
command1 ; command2 ; command3

# Run next only if previous succeeds (AND)
command1 && command2 && command3

# Run next only if previous fails (OR)
command1 || command2

# Group commands
(cd /tmp && ls)  # Runs in subshell, doesn't change current dir
```

Examples:

```bash
# Build and run only if build succeeds
go build -o myservice ./cmd/myservice && ./myservice

# Run tests, show message on failure
go test ./... || echo "Tests failed!"

# Update, tidy, and test
go get -u ./... && go mod tidy && go test ./...
```

---

## Checking System Status

### Running Processes

```bash
# Show your processes
ps

# Show all processes
ps aux

# Show process tree
pstree

# Interactive process viewer
top

# Better interactive viewer (if installed)
htop
```

### Network

```bash
# Show what's using a port
sudo lsof -i :8080

# Show listening ports
sudo ss -tlnp

# Or with netstat
sudo netstat -tlnp

# Test if a port is open
nc -zv localhost 8080
```

### Disk and Memory

```bash
# Disk usage
df -h

# Directory size
du -sh myproject/

# Memory usage
free -h
```

---

## Package Management (Ubuntu/Debian)

```bash
# Update package list
sudo apt update

# Upgrade installed packages
sudo apt upgrade

# Install a package
sudo apt install package-name

# Remove a package
sudo apt remove package-name

# Search for packages
apt search keyword
```

---

## Sudo (Superuser)

Some commands require administrator privileges:

```bash
# Run single command as root
sudo apt update

# Open root shell (use carefully)
sudo -i

# Edit protected file
sudo nano /etc/hosts
```

---

## Troubleshooting

### Command Not Found

If you see `command not found`:

```bash
# Check if installed
which go
which makewaffle

# Check PATH
echo $PATH

# Rehash commands (if recently installed)
hash -r
```

See [Setting Your PATH](./set-path.md) if commands aren't found.

### Permission Denied

```bash
# Check file permissions
ls -la filename

# Make executable
chmod +x filename

# Run with sudo if needed
sudo ./command
```

### Process Won't Stop

```bash
# Try Ctrl + C first

# Find the process
ps aux | grep processname

# Kill gracefully
kill <pid>

# Force kill
kill -9 <pid>

# Kill by name
pkill -9 processname
```

### Port Already in Use

```bash
# Find what's using the port
sudo lsof -i :8080

# Kill the process
kill <pid>

# Or use a different port
WAFFLE_HTTP_PORT=9090 go run ./cmd/myservice
```

---

## SSH (Remote Servers)

```bash
# Connect to remote server
ssh username@hostname

# Connect with specific port
ssh -p 2222 username@hostname

# Copy file to remote
scp file.txt username@hostname:/path/

# Copy file from remote
scp username@hostname:/path/file.txt ./

# Copy directory
scp -r myproject/ username@hostname:/path/
```

---

## Tips for WAFFLE Development

1. **Use Tab completion** — Saves typing and prevents errors
2. **Use Ctrl + R** — Search command history quickly
3. **Use tmux or screen** — Keep sessions alive after disconnect
4. **Use `&&`** — Chain commands that depend on each other
5. **Use `tail -f`** — Watch log files in real-time
6. **Open VSCode from terminal** — Run `code .` for correct PATH
7. **Use aliases** — Add shortcuts to ~/.bashrc:
   ```bash
   alias wr='go run ./cmd/myservice'
   alias wt='go test ./...'
   alias wb='go build -o myservice ./cmd/myservice'
   ```

---

## Shell Configuration

### Bash vs Zsh

Most Linux distributions use **bash** by default. Some users prefer **zsh**.

Check your shell:
```bash
echo $SHELL
```

Switch to zsh:
```bash
# Install zsh
sudo apt install zsh

# Change default shell
chsh -s $(which zsh)
```

### Configuration Files

| Shell | Interactive Login | Interactive Non-Login |
|-------|------------------|----------------------|
| bash | ~/.bash_profile, ~/.profile | ~/.bashrc |
| zsh | ~/.zprofile | ~/.zshrc |

---

## See Also

- [Setting Your PATH](./set-path.md) — Configure Go and WAFFLE commands
- [makewaffle CLI Guide](./makewaffle.md) — Scaffold new projects
- [How to Write Your First WAFFLE Service](./first-service.md) — Step-by-step tutorial
