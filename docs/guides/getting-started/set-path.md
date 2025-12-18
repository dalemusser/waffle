# Setting Your PATH for Go and WAFFLE

When you install Go tools — including the **makewaffle** and **wafflectl** commands — they are placed into your Go bin directory, usually:

```
~/go/bin
```

To run these commands from any terminal without typing the full path (e.g., `~/go/bin/makewaffle`), you must add `~/go/bin` to your **PATH** environment variable.

This guide shows how to set your PATH correctly on macOS, Linux, and Windows, across all common shells.

---

# 📝 Editing Config Files (Using nano)

For macOS and Linux, the simplest way to edit your shell configuration files is with **nano**, a beginner‑friendly terminal editor.

Example — editing your shell profile:

```bash
nano ~/.zshrc      # for zsh
nano ~/.bashrc     # for bash
nano ~/.bash_profile
```

**nano shortcuts:**
- Save changes: **Ctrl‑O**
- Exit nano: **Ctrl‑X**
- Cancel an action: **Ctrl‑C**

You can use nano whenever this guide says “edit your PATH in …”.

---

# 🧇 1. Confirm Your Go Bin Directory

Run:

```bash
go env GOPATH
```

You'll see something like:

```
/Users/<username>/go
```

Your Go binaries live in:

```
$GOPATH/bin
```

Which is typically:

```
~/go/bin
```

---

# 🍎 2. macOS

macOS ships with **zsh** as the default shell (newer versions) but still supports **bash**, **fish**, and others.

## macOS + zsh (default)
On macOS, Terminal and iTerm usually launch **login shells**, which read `~/.zprofile` rather than `~/.zshrc`. You may edit either file, but `~/.zprofile` is recommended for PATH configuration.

Edit your PATH in:

```
~/.zprofile
```

Add this line:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Apply the change:

```bash
source ~/.zprofile
```

(If you prefer to keep PATH settings in `~/.zshrc`, you can place the same export there.)

## macOS + bash
If you're using bash instead of zsh, edit:

```
~/.bash_profile
```

Or if that file does not exist, use:

```
~/.bashrc
```

Add:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Reload:

```bash
source ~/.bash_profile
```

## macOS + fish shell
Edit the fish config file:

```bash
nano ~/.config/fish/config.fish
```

Add:

```fish
set -gx PATH $HOME/go/bin $PATH
```

Reload fish:

```fish
source ~/.config/fish/config.fish
```

---

# 🖥️ 2.5 VSCode Terminal Behavior (Important!)

VSCode’s integrated terminal does **not** behave the same as the macOS Terminal or Linux terminal. This can affect whether your PATH settings (especially for `~/go/bin`) take effect.

## VSCode launched from Dock / Applications
When you open VSCode from the macOS Dock, Spotlight, or Applications folder:

- VSCode inherits **macOS GUI environment variables**, not your shell’s login environment.
- The integrated terminal starts a **non‑login interactive shell**.
- This means **`~/.zprofile` is *not* read**.
- Only `~/.zshrc` (for zsh) or `~/.bashrc` (for bash) is read.

So if your PATH is set only in `~/.zprofile`, VSCode may *not* see it.

## VSCode launched via `code .` (recommended)
If you open VSCode from Terminal using:

```bash
code .
```

Then:
- VSCode inherits **your Terminal environment**, including the PATH created by `~/.zprofile`.
- The VSCode integrated terminal then loads `~/.zshrc` or `~/.bashrc` normally.

This is the most reliable way to ensure VSCode sees the same PATH as your system terminal.

## Best Practice
To avoid PATH differences between Terminal and VSCode:

1. Add your PATH to **`~/.zprofile`** (macOS login shell)
2. ALSO add the line below to `~/.zshrc` so VSCode picks it up:

```bash
source ~/.zprofile
```

This ensures **both Terminal and VSCode** have the same PATH.

---

# 🐧 3. Linux

Works similarly across Ubuntu, Debian, Fedora, Arch, etc.

## bash (most common)
Add to:

```
~/.bashrc
```

```bash
export PATH="$HOME/go/bin:$PATH"
```

Reload:

```bash
source ~/.bashrc
```

## zsh
Add to:

```
~/.zshrc
```

```bash
export PATH="$HOME/go/bin:$PATH"
```

Reload:

```bash
source ~/.zshrc
```

## fish
Add to:

```bash
~/.config/fish/config.fish
```

```fish
set -gx PATH $HOME/go/bin $PATH
```

Reload:

```fish
source ~/.config/fish/config.fish
```

## system-wide (advanced)
If you want Go tools available to **all users**, create a file:

```
/etc/profile.d/go.sh
```

with:

```bash
export PATH="/usr/local/go/bin:$PATH"
```

Note: This assumes Go is installed system-wide in `/usr/local/go`. For user-specific Go installations, each user must configure their own PATH.

---

# 🪟 4. Windows

Windows Go binaries are installed into your **GOPATH** directory, commonly:

```
C:\Users\<User>\go\bin
```

## Windows PowerShell
Add to your PowerShell profile:

```powershell
notepad $PROFILE
```

Add this line:

```powershell
$env:PATH = "$env:USERPROFILE\go\bin;" + $env:PATH
```

Reload PowerShell:

```powershell
. $PROFILE
```

## Windows Command Prompt (cmd.exe)
Run:

```cmd
setx PATH "%USERPROFILE%\go\bin;%PATH%"
```

Close and reopen cmd.

## Windows (System Settings)
You can also add it permanently via GUI:

1. Open **System Properties**
2. Click **Environment Variables**
3. Under **User variables**, find `PATH`
4. Add:
   ```
   %USERPROFILE%\go\bin
   ```

---

# 🛠️ Troubleshooting PATH Issues in VSCode

If `makewaffle` or other Go-installed tools work in your system terminal but **do not work inside VSCode**, the cause depends on your operating system.

## macOS / Linux
VSCode launches a **non-login interactive shell**, which means it may not read `~/.zprofile` (macOS) or other login shell configuration files.

Fix:
- Ensure your PATH is set in `~/.zprofile` **and** that `~/.zshrc` contains:

```bash
source ~/.zprofile
```

- Or open VSCode from Terminal using:

```bash
code .
```

This makes VSCode inherit your correct environment.

## Windows
VSCode inherits its PATH from the **Windows environment**, not from shell startup files.

If you recently changed PATH:
- **Restart VSCode completely** so it picks up the updated PATH.
- Also restart PowerShell or cmd.exe if they were open during the change.

## Verifying PATH Inside VSCode Terminal
Run:

```bash
echo $PATH          # macOS / Linux
echo $env:PATH      # Windows PowerShell
echo %PATH%         # Windows cmd.exe
```

Look for:
- `~/go/bin` on macOS/Linux
- `%USERPROFILE%\go\bin` on Windows

If the directory is missing, re-check the steps above.

---

# 🧪 5. Test Your PATH

Try running:

```bash
makewaffle --help
```

or:

```bash
wafflectl --help
```

If the PATH is correct, these will run from any directory.

If not, try:

```bash
echo $PATH
```

and look for `~/go/bin`.

---

# 🎉 Done!

Your PATH is now correctly configured. You can run WAFFLE tools like `makewaffle`, `wafflectl`, or any Go-installed CLI without typing their full path.

---

## See Also

- [makewaffle CLI Guide](./makewaffle.md) — Scaffold new WAFFLE applications
- [How to Write Your First WAFFLE Service](./first-service.md) — Step-by-step tutorial
- [WAFFLE Quickstart Guide](./quickstart.md) — Quick overview
