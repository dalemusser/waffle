# windowsservice

Run WAFFLE applications as Windows services.

## Overview

The `windowsservice` package provides integration with the Windows Service Control Manager (SCM), allowing WAFFLE applications to run as native Windows services. It wraps the `app` package's lifecycle and responds to SCM start/stop commands with graceful shutdown.

This package uses build tags: the full implementation is only compiled on Windows. On other platforms, a stub is provided that allows code to compile but returns an error if used.

## Import

```go
import "github.com/dalemusser/waffle/windowsservice"
```

## Quick Start

```go
//go:build windows

package main

import (
    "log"

    "github.com/dalemusser/waffle/app"
    "github.com/dalemusser/waffle/windowsservice"
    "github.com/kardianos/service"
)

func main() {
    // Create the service program
    prg := &windowsservice.Program[AppConfig, DBDeps]{
        Hooks: app.Hooks[AppConfig, DBDeps]{
            LoadAppConfig: loadAppConfig,
            ConnectDB:     connectDB,
            BuildHandler:  buildHandler,
            OnShutdown:    onShutdown,
        },
    }

    // Configure the service
    svcConfig := &service.Config{
        Name:        "MyWaffleApp",
        DisplayName: "My WAFFLE Application",
        Description: "A web application built with WAFFLE",
    }

    // Create and run the service
    s, err := service.New(prg, svcConfig)
    if err != nil {
        log.Fatal(err)
    }

    if err := s.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## API

### Program

**Location:** `programwindows.go` (Windows only)

```go
type Program[C any, D any] struct {
    Hooks  app.Hooks[C, D]
    cancel func()
}
```

Generic type that wraps `app.Hooks` for use with the Windows SCM. The type parameters match those used with `app.Run`:
- `C` — Application config type
- `D` — Database/dependencies bundle type

### Start

**Location:** `programwindows.go`

```go
func (p *Program[C, D]) Start(s service.Service) error
```

Called by the SCM when the service starts. Launches `app.Run` in a goroutine and returns immediately (required by SCM protocol).

### Stop

**Location:** `programwindows.go`

```go
func (p *Program[C, D]) Stop(s service.Service) error
```

Called by the SCM when the service stops. Cancels the context passed to `app.Run`, triggering graceful shutdown.

### ErrNotWindows

**Location:** `programstub.go` (non-Windows only)

```go
var ErrNotWindows = errors.New("windowsservice: not supported on this platform")
```

Error returned on non-Windows platforms. The stub allows code to compile cross-platform, but the service functionality is only available on Windows.

## Patterns

### Dual-Mode Binary

Create a binary that runs as either a console application or Windows service:

```go
//go:build windows

package main

import (
    "flag"
    "log"
    "os"

    "github.com/dalemusser/waffle/app"
    "github.com/dalemusser/waffle/windowsservice"
    "github.com/kardianos/service"
)

func main() {
    svcFlag := flag.String("service", "", "Control the service: install, uninstall, start, stop")
    flag.Parse()

    prg := &windowsservice.Program[AppConfig, DBDeps]{
        Hooks: app.Hooks[AppConfig, DBDeps]{
            LoadAppConfig: loadAppConfig,
            ConnectDB:     connectDB,
            BuildHandler:  buildHandler,
        },
    }

    svcConfig := &service.Config{
        Name:        "MyWaffleApp",
        DisplayName: "My WAFFLE Application",
        Description: "A web application built with WAFFLE",
    }

    s, err := service.New(prg, svcConfig)
    if err != nil {
        log.Fatal(err)
    }

    // Handle service control commands
    if *svcFlag != "" {
        err := service.Control(s, *svcFlag)
        if err != nil {
            log.Printf("Service control failed: %v\n", err)
            os.Exit(1)
        }
        return
    }

    // Run the service (or interactively if not installed)
    if err := s.Run(); err != nil {
        log.Fatal(err)
    }
}
```

**Usage:**

```powershell
# Install the service
myapp.exe -service install

# Start the service
myapp.exe -service start

# Stop the service
myapp.exe -service stop

# Uninstall the service
myapp.exe -service uninstall

# Run interactively (for development)
myapp.exe
```

### Cross-Platform Main

Use build tags to have different entry points per platform:

```go
// main.go (shared logic)
package main

// Hooks and handlers defined here...
```

```go
// main_windows.go
//go:build windows

package main

import (
    "github.com/dalemusser/waffle/windowsservice"
    "github.com/kardianos/service"
)

func main() {
    prg := &windowsservice.Program[AppConfig, DBDeps]{
        Hooks: hooks,
    }

    s, _ := service.New(prg, &service.Config{
        Name: "MyApp",
    })
    s.Run()
}
```

```go
// main_other.go
//go:build !windows

package main

import "github.com/dalemusser/waffle/app"

func main() {
    app.Run(hooks)
}
```

### Service with Custom Working Directory

```go
svcConfig := &service.Config{
    Name:             "MyWaffleApp",
    DisplayName:      "My WAFFLE Application",
    Description:      "A web application built with WAFFLE",
    WorkingDirectory: "C:\\MyApp",
}
```

### Service with Dependencies

```go
svcConfig := &service.Config{
    Name:         "MyWaffleApp",
    DisplayName:  "My WAFFLE Application",
    Dependencies: []string{"MSSQLSERVER", "W3SVC"},
}
```

## Installation

### Using sc.exe

```powershell
# Create service
sc.exe create MyWaffleApp binPath= "C:\path\to\myapp.exe"

# Configure startup type
sc.exe config MyWaffleApp start= auto

# Start service
sc.exe start MyWaffleApp

# Stop service
sc.exe stop MyWaffleApp

# Delete service
sc.exe delete MyWaffleApp
```

### Using PowerShell

```powershell
# Create service
New-Service -Name "MyWaffleApp" -BinaryPathName "C:\path\to\myapp.exe" -DisplayName "My WAFFLE Application" -StartupType Automatic

# Start service
Start-Service -Name "MyWaffleApp"

# Stop service
Stop-Service -Name "MyWaffleApp"

# Remove service
Remove-Service -Name "MyWaffleApp"
```

### Using the Binary (Recommended)

```powershell
# Build with install/uninstall support
myapp.exe -service install
myapp.exe -service start
```

## Logging

Windows services cannot write to stdout/stderr in the traditional sense. Configure logging to write to files or Windows Event Log:

```go
func loadAppConfig() (AppConfig, error) {
    cfg := AppConfig{
        // Use file-based logging for services
        LogFile: "C:\\Logs\\myapp.log",
    }
    return cfg, nil
}
```

Or use the Windows Event Log via zap:

```go
import "go.uber.org/zap/zapcore"

// Configure zap to write to Windows Event Log
// (requires additional setup with eventlog package)
```

## See Also

- [app](../app/app.md) — Application lifecycle
- [server](../server/server.md) — HTTP server with graceful shutdown
- [kardianos/service](https://github.com/kardianos/service) — Cross-platform service library

