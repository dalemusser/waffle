package wafflegen

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed scaffold
var scaffoldFS embed.FS

// Manifest represents the structure defined in manifest.yaml.
type Manifest struct {
	Directories []DirectoryEntry `yaml:"directories"`
	Files       []FileEntry      `yaml:"files"`
}

// DirectoryEntry represents a directory to create.
type DirectoryEntry struct {
	Path   string `yaml:"path"`
	Readme string `yaml:"readme,omitempty"`
}

// FileEntry represents a file to create from a template.
type FileEntry struct {
	Path     string `yaml:"path"`
	Template string `yaml:"template"`
}

// TemplateData is passed to all templates during execution.
type TemplateData struct {
	AppName       string
	Module        string
	GoVersion     string
	WaffleVersion string
}

// Run is the shared entrypoint used by both wafflectl and makewaffle.
//
// binName is the CLI name to show in help/usage text (e.g. "wafflectl" or "makewaffle").
// args are the command-line arguments excluding the binary name (i.e. os.Args[1:]).
//
// It returns a process exit code; callers should os.Exit(Run(...)).
func Run(binName string, args []string) int {
	// No arguments: show top-level usage.
	if len(args) < 1 {
		usage(binName)
		return 1
	}

	// Global help flags: makewaffle --help, makewaffle -h, makewaffle help
	switch args[0] {
	case "-h", "--help", "help":
		usage(binName)
		return 0
	case "new":
		return newCmd(binName, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", args[0])
		usage(binName)
		return 1
	}
}

func usage(binName string) {
	fmt.Printf("WAFFLE CLI (%s)\n", binName)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s new <appname> --module <module-path>\n", binName)
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --module         Go module path for the new app (required)")
	fmt.Println("  --waffle-version Version of waffle to require (optional)")
	fmt.Println("  --go-version     Go language version (default: 1.21)")
	fmt.Println("  --template       Template to use: full (default: full)")
	fmt.Println("  --force          Scaffold into existing directory")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Printf("  %s new myapp --module github.com/you/myapp\n", binName)
}

func newCmd(binName string, args []string) int {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	module := fs.String("module", "", "Go module path for the new app (e.g. github.com/you/myapp)")
	waffleVersion := fs.String("waffle-version", "", "Version of github.com/dalemusser/waffle to require in go.mod (e.g. v0.1.0)")
	goVersion := fs.String("go-version", "1.21", "Go language version to declare in go.mod (e.g. 1.21)")
	templateName := fs.String("template", "full", "Template to use: full")
	force := fs.Bool("force", false, "Scaffold into an existing app directory if it already exists")
	fs.Usage = func() {
		fmt.Printf("Usage: %s new <appname> --module <module-path>\n", binName)
		fs.PrintDefaults()
	}

	// Split args into flag arguments and positional arguments so flags
	// can appear before or after the app name.
	var flagArgs []string
	var posArgs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flagArgs = append(flagArgs, a)
			// If this flag takes a value and the next arg is not another flag,
			// include it as part of the flag args.
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			posArgs = append(posArgs, a)
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		log.Println("parse flags:", err)
		return 1
	}

	if len(posArgs) < 1 {
		fmt.Println("error: appname is required")
		fmt.Println()
		fs.Usage()
		return 1
	}

	appName := posArgs[0]
	if err := validateAppName(appName); err != nil {
		fmt.Println("error:", err.Error())
		fmt.Println()
		fs.Usage()
		return 1
	}
	if *module == "" {
		fmt.Println("error: --module is required")
		fmt.Println()
		fs.Usage()
		return 1
	}

	// Validate template name
	if *templateName != "full" {
		fmt.Printf("error: unknown template %q (available: full)\n", *templateName)
		return 1
	}

	if err := scaffoldApp(appName, *module, *waffleVersion, *goVersion, *templateName, *force); err != nil {
		log.Printf("scaffold failed: %v\n", err)
		return 1
	}
	return 0
}

func scaffoldApp(appName, module, waffleVersion, goVersion, templateName string, force bool) error {
	short := appBaseName(appName)

	fmt.Printf("Creating WAFFLE app %q with module %q\n", appName, module)

	// Load manifest
	manifest, err := loadManifest()
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// Create root directory (or honor --force when it already exists).
	if err := os.Mkdir(appName, 0o755); err != nil {
		// If the directory already exists and --force was not set, fail.
		if !os.IsExist(err) || !force {
			return fmt.Errorf("mkdir %s: %w", appName, err)
		}
		// If it exists and force is true, continue and scaffold into it.
	}

	// Prepare template data
	data := TemplateData{
		AppName:       short,
		Module:        module,
		GoVersion:     goVersion,
		WaffleVersion: waffleVersion,
	}

	// Helper to join paths under app root.
	join := func(parts ...string) string {
		return filepath.Join(append([]string{appName}, parts...)...)
	}

	// Create directories
	for _, dir := range manifest.Directories {
		// Expand template variables in path
		dirPath, err := expandPath(dir.Path, data)
		if err != nil {
			return fmt.Errorf("expand directory path %q: %w", dir.Path, err)
		}

		if err := os.MkdirAll(join(dirPath), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dirPath, err)
		}

		// Create README if specified
		if dir.Readme != "" {
			readmeContent, err := executeTemplate(dir.Readme, data)
			if err != nil {
				return fmt.Errorf("execute readme template %q: %w", dir.Readme, err)
			}
			readmePath := join(dirPath, "README.md")
			if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", readmePath, err)
			}
		}
	}

	// Create files from templates
	for _, file := range manifest.Files {
		// Expand template variables in path
		filePath, err := expandPath(file.Path, data)
		if err != nil {
			return fmt.Errorf("expand file path %q: %w", file.Path, err)
		}

		// Execute the template
		content, err := executeTemplate(file.Template, data)
		if err != nil {
			return fmt.Errorf("execute template %q: %w", file.Template, err)
		}

		// Ensure parent directory exists
		fullPath := join(filePath)
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", parentDir, err)
		}

		// Write the file
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", fullPath, err)
		}
	}

	fmt.Println("Done!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", appName)
	fmt.Println("  go mod tidy")
	fmt.Printf("  go run ./cmd/%s\n", short)
	fmt.Println("  go to http://localhost:8080 in web browser")
	fmt.Println()
	return nil
}

func loadManifest() (*Manifest, error) {
	data, err := scaffoldFS.ReadFile("scaffold/manifest.yaml")
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &manifest, nil
}

func expandPath(path string, data TemplateData) (string, error) {
	tmpl, err := template.New("path").Parse(path)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func executeTemplate(templatePath string, data TemplateData) (string, error) {
	// Read template from embedded FS
	fullPath := "scaffold/templates/" + templatePath
	content, err := fs.ReadFile(scaffoldFS, fullPath)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", fullPath, err)
	}

	// Parse and execute template
	tmpl, err := template.New(templatePath).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

func appBaseName(appName string) string {
	// Extract the last path element from the provided app name.
	// This is used for things like the cmd/<name> directory and Hooks.Name.
	s := appName
	if i := strings.LastIndex(appName, "/"); i >= 0 {
		s = appName[i+1:]
	}
	return s
}

func validateAppName(name string) error {
	// Consider only the last path element as the actual app name
	s := name
	if i := strings.LastIndex(name, "/"); i >= 0 {
		s = name[i+1:]
	}

	if s == "" {
		return fmt.Errorf("app name cannot be empty")
	}

	for i, r := range s {
		// allow letters, digits, and underscore
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			(r == '_')) {
			return fmt.Errorf("app name %q contains invalid character %q; only letters, digits, and underscore are allowed", name, r)
		}
		// do not allow starting with a digit
		if i == 0 && (r >= '0' && r <= '9') {
			return fmt.Errorf("app name %q cannot start with a digit", name)
		}
	}

	return nil
}
