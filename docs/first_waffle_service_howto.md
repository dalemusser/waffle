# How to Write Your First WAFFLE Service  
*A hands-on, beginner‑friendly guide to creating, running, and understanding your first WAFFLE-powered Go web application.*

---

# 🧇 Introduction

WAFFLE — the **Web Application Framework for Flavorful Layered Engineering** — gives you a clean, structured, and deliciously modular starting point for building Go web services.

In this guide, you will:

- Scaffold a brand‑new WAFFLE app
- Run it immediately (instant success!)
- Open the project in VSCode
- Take a guided tour of each important file WAFFLE generated
- Add your first route
- Explore optional enhancements (CORS, Windows services)
- Follow links to **short example documents** if you want to see deeper patterns

This guide keeps the required steps simple and introduces complexity only when *you* decide to click into it.

---

# 🏁 1. Install the WAFFLE CLI (makewaffle)

To generate new WAFFLE services, install the **makewaffle** command:

```bash
go install github.com/dalemusser/waffle/cmd/makewaffle@latest
```

After installation, ensure that your Go bin directory  
(often `~/go/bin`) is on your system PATH so the `makewaffle` command is accessible.

> **Optional:**  
> WAFFLE also provides a CLI named **wafflectl**, following common industry naming conventions (like `kubectl`).  
> You may substitute `wafflectl` for `makewaffle` in any command.  
> This guide uses **makewaffle** for clarity and approachability.

# 🧱 2. Generate Your First WAFFLE Project

Now that you have the WAFFLE CLI installed, you can generate a full project scaffold using `makewaffle`.

Generate a new service called **hello**:

```bash
makewaffle new hello \
  --module github.com/you/hello
```

Now enter the project directory — you will stay in this directory for the rest of the steps:

```bash
cd hello
```

Once inside the project directory, prepare your module:

```bash
go mod tidy
```

You now have a fully‑structured WAFFLE application ready to run.

---

# 🧰 3. Open the Project in VSCode

If you do not already have a preferred editor, we recommend **Visual Studio Code**.

Install:  
https://code.visualstudio.com/

Open the project in VSCode.

Make sure you are **inside the project directory** (`hello`), then run:

```bash
code .
```

This opens the entire WAFFLE project in VSCode.

---

# ▶️ 4. Run Your WAFFLE App

Start the service (still inside the `hello` directory):

```bash
go run ./cmd/hello
```

In a **web browser**, visit:

```
http://localhost:8080
```

In the browser window you should see:

```
Hello from WAFFLE!
```

🎉 **Success!**  
You now have a functioning WAFFLE service.

Now that you’ve seen it run, let’s explore the pieces WAFFLE generated.

---

# 🧭 4. Guided Tour of Your Generated WAFFLE Project

What follows is a **tour** — you don’t need to edit anything yet.  
This is just to build familiarity with the structure.

---

## ⚙️ 4.1 Application Configuration

Open:

```
internal/app/bootstrap/appconfig.go
```

You’ll see:

```go
package bootstrap

type AppConfig struct {
    Greeting string
}
```

**Purpose:**  
This struct holds app‑specific configuration that *you* define.  
WAFFLE loads core config automatically; your app config lives here.

**When you’ll edit this:**  
When you need custom values (e.g., app name, theme, feature toggles, etc.).

**Optional deeper dive:**  
[Examples of AppConfig patterns](./examples/appconfig_examples.md)

---

## 🗄️ 4.2 Database / Backend Dependencies

Open:

```
internal/app/bootstrap/dbdeps.go
```

```go
package bootstrap

type DBDeps struct{}
```

**Purpose:**  
`DBDeps` is where you place long-lived back-end dependencies:

- MongoDB clients  
- SQL DB handles  
- Redis connections  
- AWS clients  
- etc.

Right now it’s empty — that’s fine.

**When you’ll edit this:**  
As soon as your app needs to talk to a database or external service.

**Curious what this looks like?**  
Examples:

- [Using MongoDB with DBDeps](./examples/dbdeps_mongo.md)  
- [Using Postgres with DBDeps](./examples/dbdeps_postgres.md)  
- [Using Redis with DBDeps](./examples/dbdeps_redis.md)

---

## 🔌 4.3 WAFFLE Hooks (The Heart of Your App)

Open:

```
internal/app/bootstrap/hooks.go
```

This file wires your app into the WAFFLE lifecycle.

You’ll see the key pieces:

- `LoadConfig` – loads core + app config  
- `ConnectDB` – connects your databases (empty for now)  
- `EnsureSchema` – create indexes / bootstrap things (optional)  
- `BuildHandler` – builds your HTTP router  
- `Hooks{...}` – ties everything together

Here is the important part:  
**You don’t need to change anything yet.**

But later, you’ll modify:

- `ConnectDB` when adding Mongo/Postgres/etc.  
- `BuildHandler` when adding routes  
- `EnsureSchema` if you add schemas or indexes  

**Want to peek at real examples?**

- [MongoDB ConnectDB example](./examples/dbdeps_mongo.md)  
- [Adding additional routes](./examples/routes_examples.md)  
- [Using middleware in BuildHandler](./examples/middleware_examples.md)

---

## 🚀 4.4 The WAFFLE Entry Point

Open:

```
cmd/hello/main.go
```

```go
package main

import (
    "context"
    "log"

    "github.com/dalemusser/waffle/app"
    "github.com/you/hello/internal/app/bootstrap"
)

func main() {
    if err := app.Run(context.Background(), bootstrap.Hooks); err != nil {
        log.Fatal(err)
    }
}
```

In WAFFLE projects, runnable commands are placed under the `cmd/` directory.  
Here, `cmd/hello/main.go` is the entry point for your service.  
When you run:

```bash
go run ./cmd/hello
```

you are telling Go to run the `main.go` file inside the `cmd/hello` folder.

This file:

- Starts the WAFFLE lifecycle  
- Loads config  
- Connects databases  
- Builds the router  
- Starts HTTP/HTTPS servers  
- Handles graceful shutdown  

You will rarely need to modify it.

---

# 🧩 5. Add a New Route (Your First Feature)

Let’s actually add to your app now.

Create a new directory:

```bash
mkdir internal/app/features/about
```

### Handler

`internal/app/features/about/about.go`

```go
package about

import "net/http"

func Handler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("WAFFLE apps are deliciously layered."))
}
```

### Routes

`internal/app/features/about/routes.go`

```go
package about

import "github.com/go-chi/chi/v5"

func Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/", Handler)
    return r
}
```

### Register the route

In `BuildHandler` in `hooks.go`:

```go
r.Mount("/about", about.Routes())
```

Now start the server again:

```
go run ./cmd/hello
```

In a **web browser**, visit:

```
http://localhost:8080/about
```

**Optional deeper dive:**  
[Routes and Middleware Guide](./routes_and_middleware_guide.md)

---

# 🧰 6. Optional Enhancements (WAFFLE Toolkit)

WAFFLE includes extra modules you can turn on when needed.

---

## 🌐 CORS Support

To enable CORS:

1. Open:

   ```
   internal/app/bootstrap/hooks.go
   ```

2. Add the import:

   ```go
   import "github.com/dalemusser/waffle/toolkit/cors"
   ```

3. Add the middleware **right after** creating the router:

   ```go
   r := chi.NewRouter()
   r.Use(cors.Default())
   ```

More CORS examples:  
[Advanced CORS patterns](./examples/cors_examples.md)

---

## 🪟 Windows Service Support

Add:

```go
import "github.com/dalemusser/waffle/toolkit/windowsservice"
```

Then create a Windows-only entry point (`main_windows.go`).  
Example coming soon in the WAFFLE Examples section.

---

# 🧱 7. Adding Real Functionality

As your service grows:

- Add Mongo/Postgres/Redis in `ConnectDB`
- Prepare schemas in `EnsureSchema`
- Add middleware in `BuildHandler`
- Add templating or JSON APIs
- Implement authentication
- Add background jobs or cron (outside WAFFLE)

WAFFLE handles the structure;  
**you handle the flavor.**

---

# 🎉 Congratulations!

You've built your first WAFFLE service and learned:

- How WAFFLE apps are scaffolded  
- How to run and explore your project  
- How configuration, hooks, and routing work  
- How to add features  
- How to enable toolkit helpers  
- Where to find deeper examples  

You now have everything you need to build full WAFFLE-powered systems such as:

- **StrataHub**  
- **StrataLog**  
- **StrataSave**

Go build something delicious. 🧇🚀
