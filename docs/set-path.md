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
Edit your PATH in:

```
~/.zshrc
```

Add this line:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Apply the change:

```bash
source ~/.zshrc
```

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
If you want Go tools available to **all users**, add:

```
/etc/profile.d/go.sh
```

with:

```bash
export PATH="/home/YOURUSER/go/bin:$PATH"
```

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

Your PATH is now correctly configured.  
You can run WAFFLE tools like `makewaffle`, `wafflectl`, or any Go-installed CLI without typing their full path.

If you'd like, we can link this guide from the Quickstart or README.
