# Goliath

> Polylith-style workspace management for Go

Goliath brings [Polylith](https://polylith.gitbook.io/polylith/) architecture to Go. It manages workspaces containing components (domain logic), bases (entry points), and projects (deployable artifacts) using Go's native module system.

## Installation

**Go (any platform):**

```bash
go install github.com/igbanam/goliath/cmd/goliath@latest
```

**Homebrew (macOS / Linux):**

```bash
brew install igbanam/tap/goliath
```

**Scoop (Windows):**

```bash
scoop bucket add goliath https://github.com/igbanam/scoop-bucket
scoop install goliath
```

**From source:**

```bash
git clone https://github.com/igbanam/goliath
cd goliath
go build -o goliath ./cmd/goliath
```

**Pre-built binaries** for every platform are on the
[Releases](https://github.com/igbanam/goliath/releases) page
(Linux/macOS/Windows × amd64/arm64, with `checksums.txt`).

## Quick Start

```bash
# Create a new workspace
mkdir myapp && cd myapp
goliath create workspace --name myapp --top-ns github.com/myorg

# Create components (domain bricks with public interfaces)
goliath create component user
goliath create component admin --interface user  # implements user interface
goliath create component util

# Create a base (entry point)
goliath create base cli

# Create a deployable project
goliath create project myapp

# View workspace info
goliath info

# Validate workspace
goliath check

# Show dependencies
goliath deps

# Run tests
goliath test
```

## Workspace Structure

```
myapp/
├── go.work                 # Go workspace tying all modules together
├── goliath.yaml            # Workspace configuration
├── components/             # Component bricks
│   └── user/
│       ├── go.mod          # Module: github.com/myorg/user-component
│       ├── api/            # Public interface (package api)
│       │   └── user.go     # UserService interface, User type
│       ├── internal/       # Implementation
│       │   └── user.go     # user struct implementing UserService
│       └── test/           # Component tests
│           └── user_test.go
├── bases/                  # Base bricks (entry points)
│   └── cli/
│       ├── go.mod          # Module: github.com/myorg/cli-base
│       └── main.go         # package main — wires components together
├── projects/               # Deployable projects
│   └── myapp/
│       ├── go.mod          # Module: github.com/myorg/myapp
│       └── main.go
└── development/            # Development workspace
    └── go.mod
```

## Concepts

| Concept | Description |
|---------|-------------|
| **Component** | Domain logic with a public interface (`api/`) and private implementation (`internal/`) |
| **Base** | Entry point (package main) that wires components together |
| **Project** | Deployable artifact combining one implementation per interface + bases |
| **Interface** | Public Go interface in a component's `api/` package |
| **Brick** | A component or base |

## Commands

### `create` — Create workspace, component, base, or project

```bash
goliath create workspace --name myws --top-ns github.com/myorg
goliath create component user
goliath create component admin --interface user
goliath create base cli
goliath create project myapp
```

### `info` — Show workspace overview

```bash
goliath info
```

Shows projects, components, bases, and interfaces in a table format.

### `check` — Validate workspace

```bash
goliath check
```

Validates:
- **Error 101**: Illegal dependency on non-API namespace
- **Error 104**: Circular dependencies between components
- **Error 105**: Illegal name sharing (base same name as component)
- **Error 106**: Multiple components with same interface in a project

### `deps` — Show dependencies

```bash
goliath deps                      # Full workspace dependency matrix
goliath deps --brick user         # Dependencies for a specific brick
goliath deps --project myapp      # Dependencies within a project
goliath deps --brick user --project myapp  # Combined
```

### `diff` — Show changed files

```bash
goliath diff                      # Changed since last stable tag
goliath diff --since HEAD~1       # Changed since specific ref
goliath diff --since stable       # Changed since latest stable tag
```

### `libs` — Show library dependencies

```bash
goliath libs                      # Show all dependencies
goliath libs --compact            # Compact view
goliath libs --out libs.txt       # Write to file
```

### `test` — Run tests

```bash
goliath test                      # Test all bricks
goliath test --all                # All brick + project tests
goliath test --all-bricks         # All brick tests
goliath test --brick user         # Test specific brick
goliath test --brick user:admin   # Test multiple bricks
goliath test --project myapp      # Test specific project
goliath test --dev                # Include development project
goliath test -v                   # Verbose output
```

### `ws` — Output workspace as JSON

```bash
goliath ws                        # Full workspace as JSON
goliath ws --get components       # Get specific key
goliath ws --get components:keys  # Navigate nested keys
goliath ws --get settings         # Get settings
goliath ws --out ws.json          # Write to file
```

### `version` — Show version

```bash
goliath version
```

## Architecture Rules

1. **Components expose interfaces**: Other bricks import only the `api/` package of a component
2. **Implementation is private**: The `internal/` package cannot be imported by other bricks
3. **Bases wire components**: Bases (package main) import component APIs and wire them together
4. **Projects select implementations**: A project includes exactly one component per interface
5. **No circular dependencies**: Components must form a DAG (directed acyclic graph)

## Configuration (goliath.yaml)

```yaml
top-namespace: github.com/myorg    # Module path prefix (required)
interface-pkg: api                 # Package name for interfaces (default: "api")
default-profile: default           # Default profile name
vcs:
  name: git                        # VCS type
  auto-add: false                  # Auto-add new files to git
tag-patterns:
  stable: "^stable-.*"             # Git tag pattern for stable points
  release: "^v[0-9].*"             # Git tag pattern for releases
projects:
  development:
    alias: dev                     # Alias for development project
```

## Feature Parity with clj-poly

| clj-poly (Clojure) | Goliath (Go) | Status |
|---|---|---|
| `poly create workspace` | `goliath create workspace` | ✅ |
| `poly create component` | `goliath create component` | ✅ |
| `poly create base` | `goliath create base` | ✅ |
| `poly create project` | `goliath create project` | ✅ |
| `poly info` | `goliath info` | ✅ |
| `poly check` | `goliath check` | ✅ |
| `poly deps` | `goliath deps` | ✅ |
| `poly diff` | `goliath diff` | ✅ |
| `poly libs` | `goliath libs` | ✅ |
| `poly test` | `goliath test` | ✅ |
| `poly ws` | `goliath ws` | ✅ |
| `poly version` | `goliath version` | ✅ |
| `poly shell` | — | ⏳ Planned |
| `poly doc` | — | ⏳ Planned |

## License

MIT
