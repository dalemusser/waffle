# Terminal Guide for Windows 11

This guide covers the basics of using the terminal on Windows 11 for WAFFLE development. Windows offers multiple terminal options — this guide focuses on **PowerShell** and **Windows Terminal**, the recommended choices for modern development.

---

## Choosing a Terminal

Windows 11 provides several command-line options:

| Terminal | Description | Recommended |
|----------|-------------|-------------|
| **Windows Terminal** | Modern terminal with tabs, themes, multiple shells | Yes |
| **PowerShell 7** | Cross-platform, modern PowerShell | Yes |
| **Windows PowerShell** | Built-in PowerShell (older version) | Acceptable |
| **Command Prompt (cmd)** | Legacy Windows command line | Not recommended |

**Recommendation:** Use **Windows Terminal** with **PowerShell 7** for the best experience.

---

## Installing Windows Terminal

Windows Terminal may already be installed on Windows 11. If not:

1. Open the **Microsoft Store**
2. Search for **Windows Terminal**
3. Click **Install**

Or install via winget:

```powershell
winget install Microsoft.WindowsTerminal
```

---

## Installing PowerShell 7

Windows PowerShell (version 5.1) comes pre-installed, but PowerShell 7 is newer and cross-platform.

### Via Microsoft Store
1. Open **Microsoft Store**
2. Search for **PowerShell**
3. Install **PowerShell** (the one from Microsoft)

### Via winget
```powershell
winget install Microsoft.PowerShell
```

After installation, you can select PowerShell 7 as a profile in Windows Terminal.

---

## Opening the Terminal

### Windows Terminal (Recommended)

**From Start Menu:**
1. Press **Windows key**
2. Type `Terminal`
3. Press **Enter**

**From context menu:**
- Right-click on the desktop or in a folder
- Select **Open in Terminal**

**Keyboard shortcut:**
- Press **Windows + X**, then **I** (for Terminal)
- Press **Windows + X**, then **A** (for Terminal as Admin)

### PowerShell Directly

1. Press **Windows key**
2. Type `PowerShell`
3. Press **Enter**

---

## Understanding the Terminal Window

When you open PowerShell, you'll see a prompt like:

```
PS C:\Users\Username>
```

This shows:
- `PS` — PowerShell indicator
- `C:\Users\Username` — Current directory
- `>` — The prompt (ready for input)

---

## Essential Commands

### Navigation

| Command | Description | Example |
|---------|-------------|---------|
| `pwd` | Print working directory | `pwd` |
| `Get-Location` | Same as pwd (PowerShell native) | `Get-Location` |
| `ls` | List files and folders | `ls` |
| `dir` | List files (alias) | `dir` |
| `Get-ChildItem` | List files (PowerShell native) | `Get-ChildItem` |
| `cd <folder>` | Change directory | `cd Documents` |
| `cd ..` | Go up one directory | `cd ..` |
| `cd ~` | Go to home directory | `cd ~` |
| `cd -` | Go to previous directory | `cd -` |

### File Operations

| Command | Description | Example |
|---------|-------------|---------|
| `mkdir <name>` | Create a directory | `mkdir myproject` |
| `New-Item <file>` | Create a file | `New-Item notes.txt` |
| `Copy-Item` or `cp` | Copy a file | `cp file.txt backup.txt` |
| `Move-Item` or `mv` | Move or rename | `mv old.txt new.txt` |
| `Remove-Item` or `rm` | Delete a file | `rm unwanted.txt` |
| `rm -r <folder>` | Delete folder and contents | `rm -r oldfolder` |
| `Get-Content` or `cat` | Display file contents | `cat readme.md` |

### Getting Help

| Command | Description | Example |
|---------|-------------|---------|
| `Get-Help <cmd>` | Show help for a command | `Get-Help Get-ChildItem` |
| `<command> -?` | Quick help | `cd -?` |
| `Get-Command` | List available commands | `Get-Command *Item*` |

---

## Keyboard Shortcuts

These shortcuts work in Windows Terminal and PowerShell:

| Shortcut | Action |
|----------|--------|
| **Ctrl + C** | Cancel current command / stop process |
| **Ctrl + L** | Clear screen |
| **Ctrl + A** | Select all text |
| **Ctrl + V** | Paste |
| **Ctrl + Shift + C** | Copy (in Windows Terminal) |
| **Ctrl + Shift + V** | Paste (in Windows Terminal) |
| **Tab** | Auto-complete |
| **Up Arrow** | Previous command |
| **Down Arrow** | Next command |
| **Ctrl + R** | Search command history |
| **Ctrl + Home** | Scroll to top |
| **Ctrl + End** | Scroll to bottom |
| **Ctrl + Shift + T** | New tab (Windows Terminal) |
| **Ctrl + Shift + W** | Close tab (Windows Terminal) |
| **Alt + Enter** | Toggle fullscreen |

---

## Tab Completion

Tab completion works in PowerShell. Start typing and press **Tab**:

```powershell
cd Docu<Tab>
```

Expands to:

```powershell
cd .\Documents\
```

Press **Tab** repeatedly to cycle through matches.

---

## Working with WAFFLE Projects

### Creating a New Project

```powershell
# Navigate to where you want to create the project
cd ~\Documents

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

```powershell
# Run your application
go run ./cmd/myservice

# Build a binary
go build -o myservice.exe ./cmd/myservice

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

### Run in Background (PowerShell)

```powershell
# Start a background job
Start-Job -ScriptBlock { go run ./cmd/myservice }

# List background jobs
Get-Job

# Get output from a job
Receive-Job -Id 1

# Stop a job
Stop-Job -Id 1

# Remove completed jobs
Remove-Job -Id 1
```

### Stop a Running Process

Press **Ctrl + C** to stop a process running in the foreground.

---

## Environment Variables

### View Environment Variables

```powershell
# View all environment variables
Get-ChildItem Env:

# View a specific variable
$env:PATH
$env:GOPATH

# Or use echo
echo $env:PATH
```

### Set Environment Variables

```powershell
# Set for current session only
$env:WAFFLE_HTTP_PORT = "9090"

# Use in a command
$env:WAFFLE_ENV = "prod"; go run ./cmd/myservice
```

### Set Permanently (User Level)

```powershell
# Add to user PATH permanently
[Environment]::SetEnvironmentVariable("PATH", "$env:PATH;C:\new\path", "User")

# Set a new variable permanently
[Environment]::SetEnvironmentVariable("WAFFLE_ENV", "prod", "User")
```

Or use the GUI:
1. Press **Windows + R**
2. Type `sysdm.cpl` and press Enter
3. Click **Advanced** tab
4. Click **Environment Variables**

For PATH configuration, see [Setting Your PATH](./set-path.md).

---

## File Paths

Windows uses backslashes (`\`) for paths, but PowerShell accepts forward slashes too:

| Path | Description |
|------|-------------|
| `C:\` | Root of C: drive |
| `~` | Your home folder (`C:\Users\Username`) |
| `.` | Current directory |
| `..` | Parent directory |
| `~\Documents` | Documents folder |
| `.\myfile.txt` | File in current directory |

### Examples

```powershell
# Absolute path
cd C:\Users\Dale\Documents\myproject

# Relative path
cd ..\otherproject

# Home-relative path
cd ~\go\bin

# Forward slashes also work
cd ~/Documents/myproject
```

---

## Editing Files

### Using Notepad

```powershell
notepad filename.txt
```

### Using VSCode (Recommended)

```powershell
# Open current directory in VSCode
code .

# Open a specific file
code myfile.go
```

### Using nano (if installed via Git Bash or WSL)

```powershell
nano filename.txt
```

---

## Viewing Output

### Scrolling

- **Scroll up**: Mouse wheel, or **Ctrl + Shift + Up**
- **Scroll down**: Mouse wheel, or **Ctrl + Shift + Down**
- **Page up/down**: **Shift + Page Up/Down**

### Piping and Filtering

```powershell
# Pipe output to another command
go test ./... | Select-String "FAIL"

# View output page by page
Get-Content longfile.txt | more

# Save output to a file
go build ./... 2>&1 | Out-File build.log
```

### Redirecting Output

```powershell
# Redirect output to a file (overwrite)
go run ./cmd/myservice > output.log

# Redirect output to a file (append)
go run ./cmd/myservice >> output.log

# Redirect errors
go run ./cmd/myservice 2> errors.log

# Redirect both output and errors
go run ./cmd/myservice *> all.log
```

---

## Command History

```powershell
# View recent commands
Get-History

# Or use alias
history

# Search history interactively
# Press Ctrl + R, then type

# Run previous command
# Press Up Arrow, then Enter

# Run command by history ID
Invoke-History 42
```

---

## Multiple Commands

```powershell
# Run commands sequentially (always runs all)
command1; command2; command3

# Run next command only if previous succeeds
command1 && command2 && command3

# Run next command only if previous fails
command1 || command2
```

Example:

```powershell
# Build and run only if build succeeds
go build -o myservice.exe ./cmd/myservice && .\myservice.exe
```

---

## Checking What's Running

```powershell
# Show running processes
Get-Process

# Find a specific process
Get-Process | Where-Object { $_.ProcessName -like "*myservice*" }

# Show what's using a port
netstat -ano | Select-String ":8080"

# Find process by port
Get-NetTCPConnection -LocalPort 8080
```

---

## Windows Terminal Features

### Multiple Tabs

- **New tab**: Ctrl + Shift + T
- **Close tab**: Ctrl + Shift + W
- **Switch tabs**: Ctrl + Tab

### Split Panes

- **Split horizontally**: Alt + Shift + -
- **Split vertically**: Alt + Shift + +
- **Navigate panes**: Alt + Arrow keys
- **Close pane**: Ctrl + Shift + W

### Profiles

Windows Terminal supports multiple profiles (PowerShell, Command Prompt, WSL, etc.):

1. Click the dropdown arrow next to the + button
2. Select **Settings**
3. Add or configure profiles

---

## WSL (Windows Subsystem for Linux)

For a Linux-like experience on Windows:

### Install WSL

```powershell
wsl --install
```

This installs Ubuntu by default. Restart your computer when prompted.

### Using WSL

```powershell
# Enter WSL
wsl

# Run a single Linux command
wsl ls -la

# Install other distributions
wsl --install -d Debian
```

WSL is useful if you prefer Linux commands or need Linux-specific tools.

---

## Troubleshooting

### Command Not Found

If you see `'command' is not recognized`:

1. Check spelling
2. Verify the program is installed
3. Check your PATH

```powershell
# Check if a command exists
Get-Command go
Get-Command makewaffle

# Check your PATH
$env:PATH -split ';'
```

### Permission Denied

If you need elevated privileges:

1. Right-click Windows Terminal
2. Select **Run as administrator**

Or in PowerShell:

```powershell
Start-Process powershell -Verb RunAs
```

### Execution Policy

If scripts won't run:

```powershell
# Check current policy
Get-ExecutionPolicy

# Allow local scripts (run as Administrator)
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Process Won't Stop

If **Ctrl + C** doesn't work:

```powershell
# Find the process
Get-Process | Where-Object { $_.ProcessName -like "*myservice*" }

# Stop by name
Stop-Process -Name "myservice"

# Stop by ID
Stop-Process -Id 12345 -Force
```

---

## Tips for WAFFLE Development

1. **Use Windows Terminal** — Multiple tabs and better experience
2. **Use PowerShell 7** — More features than Windows PowerShell
3. **Use Tab completion** — Saves typing
4. **Use command history** — Press Up Arrow
5. **Open VSCode from Terminal** — Run `code .` in your project
6. **Use split panes** — Server in one pane, commands in another
7. **Consider WSL** — If you prefer Linux commands

---

## PowerShell vs Command Prompt

| Feature | PowerShell | Command Prompt |
|---------|------------|----------------|
| Object pipeline | Yes | No |
| Tab completion | Better | Basic |
| Aliases (ls, cat, etc.) | Yes | No |
| Scripting | Powerful | Limited |
| Cross-platform | Yes (PS 7) | No |
| Go development | Recommended | Works |

**Recommendation:** Use PowerShell for WAFFLE development.

---

## See Also

- [Setting Your PATH](./set-path.md) — Configure Go and WAFFLE commands
- [makewaffle CLI Guide](./makewaffle.md) — Scaffold new projects
- [How to Write Your First WAFFLE Service](./first-service.md) — Step-by-step tutorial
- [Windows Service Examples](../deployment/windows-service.md) — Run WAFFLE as a Windows service
