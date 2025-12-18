# Customized makewaffle Ideas

*Future directions for extending WAFFLE scaffolding with custom templates and teaching workflows.*

This page collects ideas for making **makewaffle** extensible, enabling custom template sets, teaching-focused variants, and domain-specific generators.

---

## Current State

WAFFLE's scaffolding already uses embedded template files via `embed.FS`:

```
internal/wafflegen/
├── wafflegen.go              # Scaffolding logic
└── scaffold/
    ├── manifest.yaml         # Defines directories and files to generate
    └── templates/
        ├── go.mod.tmpl
        ├── cmd/
        │   └── main.go.tmpl
        ├── bootstrap/
        │   ├── hooks.go.tmpl
        │   ├── config.go.tmpl
        │   ├── appconfig.go.tmpl
        │   ├── db.go.tmpl
        │   ├── dbdeps.go.tmpl
        │   ├── startup.go.tmpl
        │   ├── routes.go.tmpl
        │   └── shutdown.go.tmpl
        └── readme/
            ├── features.md.tmpl
            ├── resources.md.tmpl
            ├── system.md.tmpl
            ├── store.md.tmpl
            ├── policy.md.tmpl
            └── models.md.tmpl
```

The `manifest.yaml` declares:
- **Directories** to create (with optional README templates)
- **Files** to generate from templates

Templates receive `TemplateData` with: `AppName`, `Module`, `GoVersion`, `WaffleVersion`.

---

## Future Enhancements

### 1. Custom Template Overrides

Allow developers to override specific templates without forking WAFFLE:

**Possible override sources:**

- `.waffle/templates/` in the user's project
- Environment variable: `WAFFLE_TEMPLATES=/path/to/custom/templates`

If present, `makewaffle` would:
- Prefer user-supplied templates
- Fall back to embedded defaults

This enables:
- Teams sharing org-level scaffolding templates
- Teachers distributing starter templates for courses
- Individual devs maintaining personal scaffolding preferences

---

### 2. Multiple Template "Flavors"

Support named template sets for different use cases:

- **full** (current default) — Complete WAFFLE app with all directories
- **api** — JSON API skeleton only, no HTML templates
- **minimal** — Bare minimum to run `app.Run`

Usage:
```bash
makewaffle new myapp --module github.com/you/myapp --template api
```

Each flavor would have its own `manifest.yaml` and templates in:
```
internal/wafflegen/scaffold/
├── full/
│   ├── manifest.yaml
│   └── templates/
├── api/
│   ├── manifest.yaml
│   └── templates/
└── minimal/
    ├── manifest.yaml
    └── templates/
```

---

### 3. Custom makewaffle Variants

Enable building custom scaffolding CLIs that use different embedded templates:

```go
// cmd/makestrata/main.go
package main

import (
    "embed"
    "github.com/dalemusser/waffle/internal/wafflegen"
)

//go:embed templates
var customTemplates embed.FS

func main() {
    // Use custom templates instead of WAFFLE defaults
    wafflegen.RunWithTemplates("makestrata", os.Args[1:], customTemplates)
}
```

This allows:
- **Strata-flavored WAFFLE** — Preconfigured for educational WebGL game services
- **Admin-dashboard WAFFLE** — Auth, RBAC, dashboard starter
- **Teaching WAFFLE** — Extra comments, simpler structure for students

---

### 4. Teaching Platform Potential

Teachers could create different scaffolding levels for students:

- **Beginner templates** — More comments, simpler structure
- **Intermediate templates** — Some files pre-filled with examples
- **Advanced templates** — Production-ready patterns
- **Domain-specific templates** — Games, APIs, admin dashboards

A teacher's custom template repo:
```
teaching-waffle-templates/
├── manifest.yaml
└── templates/
    ├── bootstrap/
    │   └── hooks.go.tmpl    # With extra explanatory comments
    └── features/
        └── example/         # Pre-built example feature
```

---

## Why This Matters

WAFFLE is not just a web framework:

- It is a **teaching platform**
- It is a **game/education infrastructure foundation**
- It is a **reusable engineering stack**

Template-driven scaffolding enables WAFFLE to:

- Support multiple curated starter packs
- Support instructors with approachable scaffolding
- Keep WAFFLE core clean, stable, and minimal
- Let users build their own `makewaffle` variants without forking WAFFLE

---

## Implementation Notes

The current architecture already supports most of this:

1. **Templates are already files** — No code changes needed to the template format
2. **manifest.yaml is declarative** — Easy to create alternative manifests
3. **`embed.FS` is swappable** — Could accept external FS at runtime

Remaining work:
- Add template override discovery logic
- Support multiple built-in flavors
- Expose API for custom CLIs to provide their own templates
- Documentation for creating custom template sets

---

## See Also

- [makewaffle Guide](../guides/getting-started/makewaffle.md) — Current usage
- [First Service](../guides/getting-started/first-service.md) — What gets generated
