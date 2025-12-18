# Tailwind CSS Setup in WAFFLE

*Using the Tailwind standalone CLI to build CSS for Go template projects without Node.js or npm.*

---

## Overview

WAFFLE applications use Go templates distributed across multiple directories:
- **Shared templates** in `internal/app/resources/templates/`
- **Feature templates** in `internal/app/features/*/templates/`

Tailwind CSS needs to scan all these locations to find the CSS classes you use. This guide shows how to set up the [Tailwind standalone CLI](https://tailwindcss.com/blog/standalone-cli) to work with WAFFLE's template structure.

### Why the Standalone CLI?

| Benefit | Description |
|---------|-------------|
| **No Node.js required** | Single executable, no npm/node dependencies |
| **Simple deployment** | Download once, run anywhere |
| **Full functionality** | Same features as the npm version |
| **Go-friendly** | Fits naturally into Go projects |

---

## Installation

### Download the Standalone CLI

Download the appropriate executable for your platform from the [Tailwind CSS releases page](https://github.com/tailwindlabs/tailwindcss/releases/latest).

#### macOS (Apple Silicon)

```bash
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-arm64
chmod +x tailwindcss-macos-arm64
mv tailwindcss-macos-arm64 tailwindcss
```

#### macOS (Intel)

```bash
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-x64
chmod +x tailwindcss-macos-x64
mv tailwindcss-macos-x64 tailwindcss
```

#### Linux (x64)

```bash
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
chmod +x tailwindcss-linux-x64
mv tailwindcss-linux-x64 tailwindcss
```

#### Linux (ARM64)

```bash
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-arm64
chmod +x tailwindcss-linux-arm64
mv tailwindcss-linux-arm64 tailwindcss
```

#### Windows

Download `tailwindcss-windows-x64.exe` from the releases page and rename to `tailwindcss.exe`.

### Recommended Location

Place the executable in your project root or a `tools/` directory:

```
myapp/
├── tailwindcss           # Standalone CLI executable
├── tailwind.config.js    # Configuration
├── internal/
│   └── app/
│       ├── resources/
│       │   └── templates/
│       └── features/
│           └── */templates/
└── ...
```

**Tip**: Add `tailwindcss` to your `.gitignore` since it's a platform-specific binary. Team members download their own copy.

---

## Project Structure for WAFFLE

### Directory Layout

```
myapp/
├── tailwindcss              # Standalone CLI (gitignored)
├── tailwind.config.js       # Tailwind configuration
├── internal/
│   └── app/
│       ├── resources/
│       │   ├── static/
│       │   │   └── css/
│       │   │       ├── input.css    # Tailwind input (source)
│       │   │       └── output.css   # Generated CSS (embedded)
│       │   └── templates/
│       │       ├── layout.gohtml
│       │       └── menu.gohtml
│       └── features/
│           ├── users/
│           │   └── templates/
│           │       ├── users_list.gohtml
│           │       └── users_form.gohtml
│           ├── products/
│           │   └── templates/
│           │       └── products_list.gohtml
│           └── dashboard/
│               └── templates/
│                   └── dashboard.gohtml
└── ...
```

---

## Configuration

### Create tailwind.config.js

Create `tailwind.config.js` in your project root:

```javascript
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    // Shared templates in resources
    './internal/app/resources/templates/**/*.gohtml',

    // Feature templates (all features)
    './internal/app/features/**/templates/**/*.gohtml',

    // Any Go files that might contain template strings
    './internal/app/**/*.go',
  ],
  theme: {
    extend: {
      // Custom colors, fonts, spacing, etc.
      colors: {
        'brand': {
          50: '#f0f9ff',
          100: '#e0f2fe',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
        },
      },
    },
  },
  plugins: [
    // Add plugins here if needed
    // require('@tailwindcss/forms'),
    // require('@tailwindcss/typography'),
  ],
}
```

### Content Path Patterns Explained

| Pattern | Matches |
|---------|---------|
| `./internal/app/resources/templates/**/*.gohtml` | All `.gohtml` files in resources/templates and subdirectories |
| `./internal/app/features/**/templates/**/*.gohtml` | All `.gohtml` files in any feature's templates directory |
| `./internal/app/**/*.go` | Go files (catches inline template strings) |

**Note**: Paths are relative to your project root (where you run the `tailwindcss` command), not relative to `tailwind.config.js`.

---

## Input CSS File

### Create input.css

Create `internal/app/resources/static/css/input.css`:

```css
/* Tailwind base, components, and utilities */
@import "tailwindcss";

/* Custom styles below */

/* Example: Custom component classes */
@layer components {
  .btn {
    @apply px-4 py-2 rounded font-semibold transition-colors;
  }

  .btn-primary {
    @apply bg-blue-600 text-white hover:bg-blue-700;
  }

  .btn-secondary {
    @apply bg-gray-200 text-gray-800 hover:bg-gray-300;
  }

  .btn-danger {
    @apply bg-red-600 text-white hover:bg-red-700;
  }

  .card {
    @apply bg-white rounded-lg shadow-md p-6;
  }

  .form-input {
    @apply w-full px-3 py-2 border border-gray-300 rounded-md
           focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent;
  }

  .form-label {
    @apply block text-sm font-medium text-gray-700 mb-1;
  }
}

/* Example: Custom utility classes */
@layer utilities {
  .text-balance {
    text-wrap: balance;
  }
}
```

### For Tailwind v3 (if using older version)

If you're using Tailwind v3 instead of v4, use this format:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

/* Custom styles below */
```

---

## Building CSS

### Development (Watch Mode)

Run from your project root:

```bash
./tailwindcss -i ./internal/app/resources/static/css/input.css \
              -o ./internal/app/resources/static/css/output.css \
              --watch
```

This watches your template files and rebuilds `output.css` whenever you:
- Add or remove Tailwind classes in templates
- Modify `input.css`
- Change `tailwind.config.js`

### Production Build

Build minified CSS for production:

```bash
./tailwindcss -i ./internal/app/resources/static/css/input.css \
              -o ./internal/app/resources/static/css/output.css \
              --minify
```

### Using a Makefile

Add to your `Makefile`:

```makefile
# Tailwind CSS
.PHONY: css css-watch css-prod

CSS_INPUT  = ./internal/app/resources/static/css/input.css
CSS_OUTPUT = ./internal/app/resources/static/css/output.css

css:
	./tailwindcss -i $(CSS_INPUT) -o $(CSS_OUTPUT)

css-watch:
	./tailwindcss -i $(CSS_INPUT) -o $(CSS_OUTPUT) --watch

css-prod:
	./tailwindcss -i $(CSS_INPUT) -o $(CSS_OUTPUT) --minify

# Build everything
build: css-prod
	go build -o bin/myapp ./cmd/myapp

# Development
dev: css-watch &
	go run ./cmd/myapp
```

### Using a Shell Script

Create `scripts/build-css.sh`:

```bash
#!/bin/bash

INPUT="./internal/app/resources/static/css/input.css"
OUTPUT="./internal/app/resources/static/css/output.css"

case "$1" in
  watch)
    echo "Watching for changes..."
    ./tailwindcss -i "$INPUT" -o "$OUTPUT" --watch
    ;;
  prod|production)
    echo "Building production CSS..."
    ./tailwindcss -i "$INPUT" -o "$OUTPUT" --minify
    ;;
  *)
    echo "Building CSS..."
    ./tailwindcss -i "$INPUT" -o "$OUTPUT"
    ;;
esac
```

---

## Including in Templates

### Update layout.gohtml

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }}</title>

    <!-- Tailwind CSS (generated) -->
    <link rel="stylesheet" href="/static/css/output.css">

    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    {{ template "menu" . }}

    <main id="content" class="container mx-auto px-4 py-8">
        {{ template "content" . }}
    </main>

    <footer class="container mx-auto px-4 py-6 text-center text-gray-500 text-sm">
        <p>&copy; 2024 My Application</p>
    </footer>
</body>
</html>
```

### Example Feature Template

`internal/app/features/users/templates/users_list.gohtml`:

```html
{{ define "users_list" }}
{{ template "layout.gohtml" . }}
{{ end }}

{{ define "content" }}
<div class="flex justify-between items-center mb-6">
    <h1 class="text-2xl font-bold text-gray-900">Users</h1>
    <a href="/users/new"
       hx-get="/users/new"
       hx-target="#content"
       hx-push-url="true"
       class="btn btn-primary">
        Add User
    </a>
</div>

<div class="card">
    {{ template "users_table" . }}
</div>
{{ end }}

{{ define "users_table" }}
<div id="users-table" class="overflow-x-auto">
    <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
            <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Name
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Email
                </th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Role
                </th>
                <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                </th>
            </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
            {{ range .Users }}
            <tr class="hover:bg-gray-50">
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="font-medium text-gray-900">{{ .Name }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-gray-500">
                    {{ .Email }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full
                                 {{ if eq .Role "admin" }}bg-purple-100 text-purple-800
                                 {{ else }}bg-green-100 text-green-800{{ end }}">
                        {{ .Role }}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <a href="/users/{{ .ID }}/edit"
                       hx-get="/users/{{ .ID }}/edit"
                       hx-target="#content"
                       class="text-blue-600 hover:text-blue-900 mr-4">
                        Edit
                    </a>
                    <button hx-delete="/users/{{ .ID }}"
                            hx-target="#users-table"
                            hx-swap="outerHTML"
                            hx-confirm="Delete {{ .Name }}?"
                            class="text-red-600 hover:text-red-900">
                        Delete
                    </button>
                </td>
            </tr>
            {{ else }}
            <tr>
                <td colspan="4" class="px-6 py-8 text-center text-gray-500">
                    No users found
                </td>
            </tr>
            {{ end }}
        </tbody>
    </table>
</div>
{{ end }}
```

---

## Embedding the Generated CSS

The generated `output.css` is embedded into your binary via `go:embed`:

### resources/resources.go

```go
package resources

import (
    "embed"
    "io/fs"
    "sync"

    "github.com/yourusername/yourapp/waffle/templates"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var registerOnce sync.Once

func LoadSharedTemplates() {
    registerOnce.Do(func() {
        templates.Register(templates.Set{
            Name:     "shared",
            FS:       templateFS,
            Patterns: []string{"templates/*.gohtml"},
        })
    })
}

// StaticFS returns the embedded static files.
func StaticFS() fs.FS {
    sub, _ := fs.Sub(staticFS, "static")
    return sub
}
```

**Important**: Run `./tailwindcss ... --minify` before `go build` to ensure the latest CSS is embedded.

---

## Development Workflow

### Recommended Process

1. **Start CSS watcher** in one terminal:
   ```bash
   ./tailwindcss -i ./internal/app/resources/static/css/input.css \
                 -o ./internal/app/resources/static/css/output.css \
                 --watch
   ```

2. **Run your Go app** in another terminal:
   ```bash
   go run ./cmd/myapp
   ```

3. **Edit templates** — CSS rebuilds automatically when you add/remove Tailwind classes

4. **Build for production**:
   ```bash
   ./tailwindcss -i ./internal/app/resources/static/css/input.css \
                 -o ./internal/app/resources/static/css/output.css \
                 --minify
   go build -o bin/myapp ./cmd/myapp
   ```

### VS Code Task (Optional)

Add to `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Tailwind Watch",
      "type": "shell",
      "command": "./tailwindcss",
      "args": [
        "-i", "./internal/app/resources/static/css/input.css",
        "-o", "./internal/app/resources/static/css/output.css",
        "--watch"
      ],
      "isBackground": true,
      "problemMatcher": []
    }
  ]
}
```

---

## Troubleshooting

### Classes not appearing in output

**Cause**: Tailwind didn't find the classes in the content paths.

**Solutions**:
1. Verify your `content` paths in `tailwind.config.js` match your template locations
2. Check glob patterns are correct (`**` for directories, `*` for files)
3. Run build and check terminal output for scanned files
4. Ensure classes aren't dynamically constructed (Tailwind needs complete class names)

```html
<!-- ✅ Good: Complete class names -->
<div class="{{ if .Active }}bg-blue-500{{ else }}bg-gray-500{{ end }}">

<!-- ❌ Bad: Dynamic class construction -->
<div class="bg-{{ .Color }}-500">
```

### "Cannot find module" errors

**Cause**: Config file references npm plugins you don't have.

**Solution**: Remove or comment out plugin requires:

```javascript
plugins: [
  // These require npm - remove if using standalone CLI
  // require('@tailwindcss/forms'),
  // require('@tailwindcss/typography'),
],
```

### Output file not updating

**Cause**: Watch mode not detecting changes.

**Solutions**:
1. Check if the file is actually being saved
2. Verify the template file is within a `content` path
3. Restart the watch process
4. Check file permissions on output directory

### Large output file size

**Cause**: Not using `--minify` or content paths too broad.

**Solutions**:
1. Always use `--minify` for production
2. Make content paths more specific
3. Avoid patterns like `'./**/*'` that scan everything

---

## Updating Tailwind

To update to a newer version:

1. Download the latest release for your platform
2. Replace the existing `tailwindcss` executable
3. Test your build

```bash
# Example: Update on macOS ARM
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-arm64
chmod +x tailwindcss-macos-arm64
mv tailwindcss-macos-arm64 tailwindcss
```

---

## Adding Tailwind Plugins

The standalone CLI supports JavaScript config files, so you can use plugins that don't require npm:

### First-Party Plugins via CDN

For forms, typography, and other official plugins, you can include them via CDN in your layout:

```html
<!-- In layout.gohtml -->
<script src="https://cdn.tailwindcss.com?plugins=forms,typography"></script>
```

**Note**: This is for development/prototyping. For production, use the compiled CSS approach.

### Custom Plugin in Config

```javascript
// tailwind.config.js
module.exports = {
  content: [/* ... */],
  theme: {/* ... */},
  plugins: [
    // Inline plugin (no npm required)
    function({ addComponents }) {
      addComponents({
        '.card-hover': {
          '@apply transition-transform hover:scale-105': {},
        },
      })
    },
  ],
}
```

---

## See Also

- [Templates and Views](./templates-and-views.md)
- [Static File Serving](./static-files.md)
- [HTMX Integration](./htmx-integration.md)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
- [Tailwind Standalone CLI Announcement](https://tailwindcss.com/blog/standalone-cli)

---

[← Back to Frontend Documentation](./README.md)
