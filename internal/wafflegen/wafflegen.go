package wafflegen

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Run is the shared entrypoint used by both wafflectl and makewaffle.
//
// binName is the CLI name to show in help/usage text (e.g. "wafflectl" or "makewaffle").
// args are the command-line arguments excluding the binary name (i.e. os.Args[1:]).
//
// It returns a process exit code; callers should os.Exit(Run(...)).
func Run(binName string, args []string) int {
	if len(args) < 1 {
		usage(binName)
		return 1
	}

	switch args[0] {
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
	fmt.Println("Example:")
	fmt.Printf("  %s new strata_hub --module github.com/dalemusser/strata_hub\n", binName)
}

func newCmd(binName string, args []string) int {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	module := fs.String("module", "", "Go module path for the new app (e.g. github.com/you/hello_waffle)")
	waffleVersion := fs.String("waffle-version", "", "Version of github.com/dalemusser/waffle to require in go.mod (e.g. v0.1.0)")
	goVersion := fs.String("go-version", "1.21", "Go language version to declare in go.mod (e.g. 1.21)")
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

	if err := scaffoldApp(appName, *module, *waffleVersion, *goVersion, *force); err != nil {
		log.Printf("scaffold failed: %v\n", err)
		return 1
	}
	return 0
}

func scaffoldApp(appName, module, waffleVersion, goVersion string, force bool) error {
	short := appBaseName(appName)

	fmt.Printf("Creating WAFFLE app %q with module %q\n", appName, module)

	// Create root directory (or honor --force when it already exists).
	if err := os.Mkdir(appName, 0o755); err != nil {
		// If the directory already exists and --force was not set, fail.
		if !os.IsExist(err) || !force {
			return fmt.Errorf("mkdir %s: %w", appName, err)
		}
		// If it exists and force is true, continue and scaffold into it.
	}

	// Helper to join paths under app root.
	join := func(parts ...string) string {
		return filepath.Join(append([]string{appName}, parts...)...)
	}

	// go.mod
	if err := os.WriteFile(join("go.mod"), []byte(goModContent(module, waffleVersion, goVersion)), 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}

	// Directories
	dirs := []string{
		filepath.Join("cmd", short),
		"internal/app/bootstrap",
		"internal/app/features",
		"internal/app/store",
		"internal/app/policy",
		"internal/domain/models",
	}

	for _, d := range dirs {
		if err := os.MkdirAll(join(d), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	// Files
	if err := os.WriteFile(join("cmd", short, "main.go"), []byte(mainGoContent(module, short)), 0o644); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	if err := os.WriteFile(join("internal", "app", "bootstrap", "appconfig.go"), []byte(appConfigContent()), 0o644); err != nil {
		return fmt.Errorf("write appconfig.go: %w", err)
	}

	if err := os.WriteFile(join("internal", "app", "bootstrap", "dbdeps.go"), []byte(dbDepsContent()), 0o644); err != nil {
		return fmt.Errorf("write dbdeps.go: %w", err)
	}

	if err := os.WriteFile(join("internal", "app", "bootstrap", "hooks.go"), []byte(hooksContent(appName)), 0o644); err != nil {
		return fmt.Errorf("write hooks.go: %w", err)
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
			r == '_') {
			return fmt.Errorf("app name %q contains invalid character %q; only letters, digits, and underscore are allowed", name, r)
		}
		// do not allow starting with a digit
		if i == 0 && (r >= '0' && r <= '9') {
			return fmt.Errorf("app name %q cannot start with a digit", name)
		}
	}

	return nil
}

func goModContent(module, waffleVersion, goVersion string) string {
	if goVersion == "" {
		goVersion = "1.21"
	}

	// If a WAFFLE version is provided, include it as a require line.
	if waffleVersion != "" {
		return fmt.Sprintf(`module %s

go %s

require github.com/dalemusser/waffle %s
`, module, goVersion, waffleVersion)
	}

	// Default: leave WAFFLE and other deps to go get / go mod tidy.
	return fmt.Sprintf(`module %s

go %s
`, module, goVersion)
}

func mainGoContent(module, short string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"log"

	"github.com/dalemusser/waffle/app"
	"%s/internal/app/bootstrap"
)

func main() {
	if err := app.Run(context.Background(), bootstrap.Hooks); err != nil {
		log.Fatal(err)
	}
}
`, module)
}

func appConfigContent() string {
	return `package bootstrap

// AppConfig holds service-specific configuration for this WAFFLE app.
// Extend this struct as your app grows.
type AppConfig struct {
	Greeting string
}
`
}

func dbDepsContent() string {
	return `package bootstrap

// DBDeps holds database/back-end dependencies for the app.
// Extend this struct as your app evolves.
type DBDeps struct{}
`
}

func hooksContent(appName string) string {
	name := appBaseName(appName)

	const tpl = `package bootstrap

import (
    "context"
    "net/http"

    "github.com/dalemusser/waffle/app"
    "github.com/dalemusser/waffle/config"
    "github.com/go-chi/chi/v5"
    "go.uber.org/zap"
)

// LoadConfig loads WAFFLE core config and app-specific config.
func LoadConfig(logger *zap.Logger) (*config.CoreConfig, AppConfig, error) {
    coreCfg, err := config.Load(logger)
    if err != nil {
        return nil, AppConfig{}, err
    }

    appCfg := AppConfig{
        Greeting: "Hello from WAFFLE!",
    }

    return coreCfg, appCfg, nil
}

// ConnectDB connects to databases or other backends.
func ConnectDB(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
	// TODO: connect to Mongo, Postgres, Redis, etc.
	return DBDeps{}, nil
}

// EnsureSchema sets up indexes or schema as needed.
func EnsureSchema(ctx context.Context, coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) error {
	// TODO: create indexes, run migrations, etc.
	return nil
}

// BuildHandler constructs the HTTP handler for the service.
func BuildHandler(coreCfg *config.CoreConfig, appCfg AppConfig, deps DBDeps, logger *zap.Logger) (http.Handler, error) {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(appCfg.Greeting))
	})

	return r, nil
}

// Hooks wires the app into WAFFLE's lifecycle.
var Hooks = app.Hooks[AppConfig, DBDeps]{
	Name:         %q,
	LoadConfig:   LoadConfig,
	ConnectDB:    ConnectDB,
	EnsureSchema: EnsureSchema,
	BuildHandler: BuildHandler,
}
`

	return fmt.Sprintf(tpl, name)
}
