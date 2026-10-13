# Goliath Command Reference

Complete reference for all Goliath commands.

## Global Flags

All commands that operate on a workspace support:

- `--ws-dir PATH` — Path to workspace directory (alternative to running from workspace root)

---

## create

Create workspaces, components, bases, and projects.

### `goliath create workspace`

Create a new workspace.

```bash
goliath create workspace --top-ns github.com/myorg
goliath create workspace --name myws --top-ns github.com/myorg
```

**Flags:**
- `--name NAME` — Workspace name (creates subdirectory if given)
- `--top-ns NS` — Top namespace / module path prefix (required)

**Creates:**
- `goliath.yaml` — Workspace configuration
- `go.work` — Go workspace file
- `components/`, `bases/`, `projects/`, `development/` — Brick directories
- `.gitignore`, `README.md`

### `goliath create component`

Create a new component brick.

```bash
goliath create component user
goliath create component admin --interface user
```

**Arguments:**
- `name` — Component name (required)

**Flags:**
- `--interface IFACE` — Interface name (defaults to component name)

**Creates:**
- `components/<name>/go.mod`
- `components/<name>/api/<name>.go` — Public interface
- `components/<name>/internal/<name>.go` — Implementation
- `components/<name>/test/<name>_test.go` — Tests

### `goliath create base`

Create a new base brick (entry point).

```bash
goliath create base cli
```

**Arguments:**
- `name` — Base name (required)

**Creates:**
- `bases/<name>/go.mod` — With requires/replaces for all components
- `bases/<name>/main.go` — Entry point wiring components

### `goliath create project`

Create a new deployable project.

```bash
goliath create project myapp
```

**Arguments:**
- `name` — Project name (required, cannot be "development")

**Creates:**
- `projects/<name>/go.mod` — With one component per interface + bases
- `projects/<name>/main.go`
- `projects/<name>/test/` — Test directory

---

## info

Show workspace overview.

```bash
goliath info
```

**Output sections:**
1. Workspace name, directory, top namespace
2. Counts: projects, interfaces, bases, components
3. Projects table: name, alias, module path
4. Bricks table: interface, name, type, module path

---

## check

Validate workspace structure and dependencies.

```bash
goliath check
```

**Exit codes:**
- `0` — No errors (warnings OK)
- `1` — Errors found

**Validation rules:**

| Code | Severity | Description |
|------|----------|-------------|
| 101 | Error | Illegal dependency on non-API namespace |
| 104 | Error | Circular dependencies between components |
| 105 | Error | Base has same name as a component |
| 106 | Error | Multiple components with same interface in a project |

---

## deps

Show dependency information.

```bash
goliath deps                              # Workspace matrix
goliath deps --brick user                 # Brick dependencies
goliath deps --project myapp              # Project dependencies
goliath deps --brick user --project myapp # Combined
goliath deps --swap-axes                  # Swap matrix axes
goliath deps --out deps.txt               # Write to file
```

**Flags:**
- `--brick NAME` — Show deps for specific brick
- `--project NAME` — Show deps for specific project (name or alias)
- `--swap-axes` — Swap x/y axes
- `--out FILE` — Write output to file

---

## diff

Show changed files since last stable point.

```bash
goliath diff
goliath diff --since HEAD~1
goliath diff --since stable
goliath diff --since v1.0.0
```

**Flags:**
- `--since REF` — Git ref to diff against (tag, SHA, or ref)

**Stable patterns** (from goliath.yaml):
- `stable` → `^stable-.*`
- `release` → `^v[0-9].*`

---

## libs

Show all library dependencies.

```bash
goliath libs
goliath libs --compact
goliath libs --out libs.txt
```

**Flags:**
- `--compact` — Compact output format
- `--out FILE` — Write output to file

**Output columns:**
- `library` — Module path
- `version` — Version
- Per-project columns — `x` if used, `-` if not

---

## test

Run tests for bricks and/or projects.

```bash
goliath test                      # All brick tests
goliath test --all                # All brick + project tests
goliath test --all-bricks         # All brick tests only
goliath test --brick user         # Specific brick
goliath test --brick user:admin   # Multiple bricks (colon-separated)
goliath test --project myapp      # Specific project
goliath test --project myapp:other# Multiple projects
goliath test --dev                # Include development
goliath test -v                   # Verbose
```

**Flags:**
- `--brick NAMES` — Colon-separated brick names
- `--project NAMES` — Colon-separated project names
- `--all` — All brick + project tests
- `--all-bricks` — All brick tests
- `--dev` — Include development project
- `-v, --verbose` — Verbose output

---

## ws

Output workspace as JSON data.

```bash
goliath ws
goliath ws --get components
goliath ws --get components:keys
goliath ws --get components:user
goliath ws --get settings
goliath ws --get settings:vcs
goliath ws --out ws.json
```

**Flags:**
- `--get KEY` — Get specific key (colon-separated path)
- `--out FILE` — Write to file

**Special keys:**
- `keys` — List all keys at current level
- `count` — Count elements at current level

---

## version

Show Goliath version.

```bash
goliath version
```

**Output:**
```
goliath 0.1.0 (2026-10-13) - https://github.com/igbanam/goliath
```
