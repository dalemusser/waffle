# Systems That Last

*WAFFLE's philosophy of durability and clarity.*

---

## The Problem

Most software doesn't age well.

A project started 5 years ago is often:
- Stuck on old framework versions (upgrade path is painful or broken)
- Dependent on abandoned packages
- Built with patterns now considered anti-patterns
- Requiring specialized knowledge that's hard to hire for
- Expensive to maintain relative to its value

This isn't inevitable. It's a consequence of choices — choices about dependencies, complexity, and design philosophy.

---

## WAFFLE's Position

WAFFLE is designed for systems that need to last.

This isn't about being anti-modern or avoiding useful tools. It's about making choices with a 10-year horizon instead of a 10-month horizon.

---

## Principles of Longevity

### 1. Stable Foundations

Build on things that don't change:

| Foundation | Why It's Stable |
|------------|-----------------|
| **HTTP** | Hasn't fundamentally changed in 30 years |
| **HTML** | Browser vendors are committed to backward compatibility |
| **Go** | Explicit backward compatibility promise |
| **SQL** | Databases honor their query languages |
| **File systems** | Files remain files |

Build on things that change frequently:

| Foundation | Risk |
|------------|------|
| **npm packages** | Left-pad. Security vulnerabilities. Abandonment. |
| **JavaScript frameworks** | How many React patterns are now "legacy"? |
| **Build tools** | Grunt → Gulp → Webpack → Vite → ??? |
| **Cloud-specific APIs** | Lock-in. Deprecation. Price changes. |

WAFFLE chooses stable foundations.

### 2. Explicit Over Implicit

Code that will be maintained by someone else (including future you) should be obvious:

- **Explicit routing** — Look at routes.go, see all routes
- **Explicit dependencies** — DBDeps struct shows what a handler needs
- **Explicit data flow** — Request → Handler → Response, visible in code
- **Explicit configuration** — config.toml shows all settings

Implicit magic feels productive at first. It becomes expensive when debugging or onboarding.

### 3. Fewer Dependencies

Every dependency is:
- A potential security vulnerability
- A potential breaking change
- A potential abandoned project
- Something someone has to understand

WAFFLE's standard library approach:
- Use Go standard library when possible
- Choose dependencies with stable APIs
- Prefer fewer, well-maintained packages over many specialized ones
- Vendor or embed when stability matters more than updates

### 4. Boring Technology

"Boring" technology is technology that:
- Has known failure modes
- Has established best practices
- Has available expertise
- Has proven it works at scale
- Doesn't require explaining to new team members

Go, PostgreSQL, HTML, HTTP — these are boring. They work. They'll keep working.

### 5. Debuggability

When something goes wrong (it will), can you figure out what happened?

- **Logs** — Structured logging with request IDs
- **Metrics** — Prometheus metrics for everything important
- **Traces** — Request flow is visible
- **Network** — HTTP requests/responses are inspectable
- **State** — Server state is queryable (health endpoints, pprof)

Systems that are hard to debug become systems people are afraid to change.

---

## Practical Implications

### Choose Technologies With Exit Strategies

Before adopting something, ask:
- What if this project is abandoned?
- What if we need to switch away?
- Can we fork and maintain it ourselves?
- Is the surface area small enough to replace?

HTMX, for example, is ~14KB of JavaScript with a stable API. If it were abandoned tomorrow, maintaining a fork is feasible. Replacing it with vanilla JS fetch calls is tedious but possible.

### Invest in Documentation

Code without documentation becomes legacy code faster. WAFFLE encourages:
- README files in every significant directory
- Comments explaining "why", not just "what"
- Architecture documents that explain design decisions
- Examples that show complete patterns

### Design for the Maintainer

The next person to work on this code might:
- Not have context on why decisions were made
- Be less experienced than the original author
- Be working under time pressure
- Be you, in 2 years, having forgotten everything

Write code for them.

---

## The Cost of Complexity

Complexity has carrying costs:

| Complexity Source | Ongoing Cost |
|-------------------|--------------|
| Large dependency tree | Security updates, version conflicts, build issues |
| Framework-specific patterns | Training, hiring filter, documentation |
| Implicit behavior | Debugging time, unexpected interactions |
| Build tooling | CI/CD maintenance, local setup issues |
| Multiple languages | Context switching, tooling for each |

Every complexity you add should earn its place by providing more value than it costs over the system's lifetime — not just this quarter.

---

## WAFFLE's Choices

WAFFLE makes specific choices to support longevity:

| Choice | Reason |
|--------|--------|
| **Go** | Stable, fast compilation, single binary, backward compatible |
| **Chi router** | Minimal, stable, idiomatic Go |
| **Viper config** | Industry standard, well-maintained |
| **Zap logging** | Performant, structured, standard |
| **Standard library HTTP** | It's not going anywhere |
| **Explicit lifecycle** | Clear initialization and shutdown |
| **File-based configuration** | Inspectable, versionable, portable |

These aren't the "coolest" choices. They're the ones that will still work in 10 years.

---

## When Longevity Matters Less

To be fair, not every project needs a 10-year horizon:

- **Prototypes** — Ship fast, learn, throw away
- **Marketing sites** — Rebuild every few years anyway
- **Short-term projects** — Known end date
- **Experiments** — Learning is the goal

For these, optimize for speed and learning, not durability.

But for your core business systems, the tools people depend on daily, the infrastructure that must keep running — choose longevity.

---

## See Also

- [WAFFLE as Substrate](./waffle-as-substrate.md) — The foundational concept
- [Why HTMX + Tailwind](./why-htmx-tailwind.md) — A longevity-oriented UI choice
- [Why Go](./why-go.md) — Go's role in durable systems

---

[← Back to Philosophy](./README.md)
