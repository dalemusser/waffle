# Why Go

*The advantages of Go as a foundation, even for SPA developers.*

---

## The Argument

Even if you plan to use React, Vue, or any SPA framework on the frontend, WAFFLE with Go is a better backend foundation than Node.js.

This isn't tribal — "Go good, Node bad." It's practical: Go's characteristics make certain classes of problems disappear.

---

## Go's Advantages

### 1. Compiled to a Single Binary

```bash
go build -o myapp .
scp myapp server:/opt/apps/
./myapp
```

That's deployment. One file. No `node_modules`. No runtime installation. No version conflicts.

| Approach | Deployment Complexity |
|----------|----------------------|
| **Go** | Copy binary, run |
| **Node.js** | Copy code, install Node, `npm install`, hope nothing breaks |
| **Python** | Virtual environments, pip, system dependencies |
| **Ruby** | Bundler, Ruby version management, native extensions |

For long-lived systems, deployment simplicity compounds.

### 2. Cross-Compilation

```bash
GOOS=linux GOARCH=amd64 go build -o myapp-linux .
GOOS=windows GOARCH=amd64 go build -o myapp.exe .
GOOS=darwin GOARCH=arm64 go build -o myapp-mac .
```

Build for any platform from any platform. No CI gymnastics.

### 3. Embedded File System

```go
//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*.html
var templates embed.FS
```

Templates, static files, migrations — embedded in the binary. No external file dependencies. One artifact contains everything.

### 4. Fast Startup

Go applications start in milliseconds. This matters for:
- Container orchestration (fast scaling)
- Serverless (cold starts)
- Development (fast restart cycles)
- Windows services (quick recovery)

### 5. Predictable Performance

Go is compiled, garbage collected, and has a simple runtime. Performance characteristics are predictable:
- No JIT warmup
- No callback hell affecting timing
- Goroutines are cheap (thousands are fine)
- Memory usage is straightforward

### 6. Strong Standard Library

Go's standard library handles:
- HTTP server and client
- JSON encoding/decoding
- TLS and cryptography
- File I/O
- Templates
- Testing

You don't need packages for basics.

### 7. Static Typing

Catches errors at compile time, not runtime:
- Typos in field names
- Nil pointer possibilities
- Interface mismatches
- Type mismatches

The compiler is a safety net.

### 8. Explicit Error Handling

```go
result, err := doThing()
if err != nil {
    return fmt.Errorf("doing thing: %w", err)
}
```

Verbose? Yes. Clear about what can fail? Also yes.

You can't forget to handle an error — the compiler reminds you.

### 9. Backward Compatibility Promise

Go 1.0 code from 2012 still compiles and runs. The Go team takes backward compatibility seriously.

Compare to the Node/JavaScript ecosystem where frameworks, patterns, and best practices change constantly.

### 10. Simple Concurrency

```go
go processInBackground()
```

Goroutines and channels make concurrent programming straightforward. Handle many connections, background tasks, and parallel operations without callback spaghetti.

---

## For SPA Developers Specifically

If you're building a React/Vue/Svelte frontend, you might think "why not use Node.js for the backend too? Same language everywhere."

Consider what you actually gain:

| "Advantage" | Reality |
|-------------|---------|
| Same language | You still context-switch between frontend and backend concerns |
| Shared code | How much code actually makes sense to share? |
| One ecosystem | npm for backend means npm problems for backend |

Consider what you give up:

| Go Advantage | Node.js Reality |
|--------------|-----------------|
| Type safety | TypeScript helps but has escape hatches |
| Single binary | Deployment is more complex |
| Performance | Node.js is slower for compute |
| Concurrency | Async/await is simpler than callbacks, but goroutines are simpler still |
| Stability | The ecosystem churn is exhausting |

### The Hybrid Model Works

React frontend + Go backend is a proven, production-grade pattern:
- Clear separation of concerns
- Teams can specialize
- Deploy frontend and backend independently
- Use the right tool for each job

WAFFLE serves this model well. It doesn't care that you're building a React app — it just provides excellent API endpoints, authentication, and infrastructure.

---

## What Go Is Not Great At

Being honest:

| Area | Go's Limitation |
|------|-----------------|
| **Generics** | Added in Go 1.18, still maturing |
| **GUI development** | Not Go's strength |
| **Machine learning** | Python dominates |
| **Quick scripting** | More verbose than Python/Ruby |
| **Package versioning** | Historically messy, modules helped |

For web backends, none of these matter much.

---

## Learning Go

If you're coming from JavaScript:

| JavaScript | Go Equivalent |
|------------|---------------|
| `const/let` | `var` or `:=` |
| Arrow functions | Regular functions |
| Promises/async | Goroutines/channels |
| `try/catch` | Multiple return values |
| Classes | Structs + methods |
| `interface` | Implicit interface implementation |
| npm | go modules |

Go is simpler than JavaScript in many ways. Fewer features means fewer ways to do things, which means more readable code.

The [Tour of Go](https://go.dev/tour/) takes a few hours and covers most of the language.

---

## Summary

Go provides:
- **Simplicity** — Fewer ways to do things, easier to read
- **Performance** — Fast compilation, fast execution
- **Reliability** — Strong typing, explicit errors
- **Deployment** — Single binary, cross-compilation
- **Longevity** — Backward compatibility, stable ecosystem

Even if you love JavaScript and plan to use React, your backend being in Go means one less category of problems to worry about.

WAFFLE is built on Go because Go is the right foundation for systems that need to last.

---

## See Also

- [WAFFLE as Substrate](./waffle-as-substrate.md) — WAFFLE's foundational concept
- [Systems That Last](./longevity.md) — Durability in software
- [Tour of Go](https://go.dev/tour/) — Learn Go

---

[← Back to Philosophy](./README.md)
