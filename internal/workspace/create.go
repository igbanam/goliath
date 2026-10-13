package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateWorkspace creates a new goliath workspace in the given directory
func CreateWorkspace(dir, name, topNS string) error {
	if topNS == "" {
		return fmt.Errorf("top-namespace is required (use --top-ns)")
	}

	// If name is given and dir is ".", create a subdirectory
	targetDir := dir
	if name != "" && dir == "." {
		targetDir = name
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if already a workspace
	if IsWorkspaceDir(absDir) {
		return fmt.Errorf("directory %s is already a goliath workspace", absDir)
	}

	// Determine workspace name
	wsName := name
	if wsName == "" {
		wsName = filepath.Base(absDir)
	}

	// Create directory structure
	dirs := []string{
		"components",
		"bases",
		"projects",
		"development",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(absDir, d), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
		// Create .keep file
		if err := os.WriteFile(filepath.Join(absDir, d, ".keep"), []byte{}, 0644); err != nil {
			return fmt.Errorf("failed to create .keep in %s: %w", d, err)
		}
	}

	// Create goliath.yaml
	config := &Config{
		TopNamespace:   topNS,
		InterfacePkg:   "api",
		DefaultProfile: "default",
		VCS: VCSConfig{
			Name:    "git",
			AutoAdd: false,
		},
		TagPatterns: TagPatterns{
			Stable:  "^stable-.*",
			Release: "^v[0-9].*",
		},
		Projects: map[string]ProjectConfig{
			"development": {Alias: "dev"},
		},
	}

	ws := &Workspace{
		Dir:    absDir,
		Name:   wsName,
		Config: config,
	}
	if err := SaveConfig(ws); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create development/go.mod
	devModule := ProjectModulePath(topNS, "development")
	devGoMod := fmt.Sprintf(`module %s

go 1.23
`, devModule)
	if err := os.WriteFile(filepath.Join(absDir, "development", "go.mod"), []byte(devGoMod), 0644); err != nil {
		return fmt.Errorf("failed to create development/go.mod: %w", err)
	}

	// Create go.work
	if err := WriteGoWork(absDir, []string{"./development"}); err != nil {
		return fmt.Errorf("failed to create go.work: %w", err)
	}

	// Create .gitignore
	gitignore := `# Binaries
*.exe
*.dll
*.so
*.dylib
/goliath
/dist/

# Go workspace cache
go.work.sum

# IDE
.idea/
.vscode/
*.swp
`
	if err := os.WriteFile(filepath.Join(absDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Create README.md
	readme := fmt.Sprintf(`# %s

A [Goliath](https://github.com/igbanam/goliath) workspace — Polylith-style architecture for Go.

## Structure

- ` + "`components/`" + ` — Component bricks (domain logic with public interfaces)
- ` + "`bases/`" + ` — Base bricks (entry points that wire components together)
- ` + "`projects/`" + ` — Deployable projects (combine bases + components)
- ` + "`development/`" + ` — Development workspace

## Getting Started

` + "```bash" + `
# Create a component
goliath create component user

# Create a base
goliath create base cli

# Create a project
goliath create project myapp

# Show workspace info
goliath info

# Validate workspace
goliath check
` + "```" + `
`, wsName)
	if err := os.WriteFile(filepath.Join(absDir, "README.md"), []byte(readme), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	return nil
}

// CreateComponent creates a new component brick
func CreateComponent(ws *Workspace, name, iface string) error {
	if name == "" {
		return fmt.Errorf("component name is required")
	}
	if iface == "" {
		iface = name
	}

	// Check if component already exists
	if _, exists := ws.Components[name]; exists {
		return fmt.Errorf("component %q already exists", name)
	}

	topNS := ws.Config.TopNamespace
	interfacePkg := ws.Config.InterfacePkg
	if interfacePkg == "" {
		interfacePkg = "api"
	}

	modulePath := ComponentModulePath(topNS, name)
	compDir := filepath.Join(ws.Dir, "components", name)

	// Create directory structure
	dirs := []string{
		filepath.Join(compDir, interfacePkg),
		filepath.Join(compDir, "internal"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Create go.mod
	goMod := fmt.Sprintf(`module %s

go 1.23
`, modulePath)
	if err := os.WriteFile(filepath.Join(compDir, "go.mod"), []byte(goMod), 0644); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	// Create interface file (api/<name>.go)
	ifaceFile := generateComponentInterface(name, iface, interfacePkg, topNS, modulePath)
	if err := os.WriteFile(filepath.Join(compDir, interfacePkg, name+".go"), []byte(ifaceFile), 0644); err != nil {
		return fmt.Errorf("failed to create interface file: %w", err)
	}

	// Create implementation file (internal/<name>.go)
	implFile := generateComponentImpl(name, iface, interfacePkg, modulePath)
	if err := os.WriteFile(filepath.Join(compDir, "internal", name+".go"), []byte(implFile), 0644); err != nil {
		return fmt.Errorf("failed to create implementation file: %w", err)
	}

	// Create a test file
	testFile := generateComponentTest(name, iface, interfacePkg, modulePath)
	testDir := filepath.Join(compDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return fmt.Errorf("failed to create test directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, name+"_test.go"), []byte(testFile), 0644); err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}

	// Add to go.work
	relPath := fmt.Sprintf("./components/%s", name)
	if err := AddToGoWork(ws.Dir, relPath); err != nil {
		return fmt.Errorf("failed to update go.work: %w", err)
	}

	// Update workspace in memory
	ws.Components[name] = &Component{
		Name:      name,
		Dir:       compDir,
		Module:    modulePath,
		Interface: iface,
	}

	return nil
}

// CreateBase creates a new base brick
func CreateBase(ws *Workspace, name string) error {
	if name == "" {
		return fmt.Errorf("base name is required")
	}

	// Check if base already exists
	if _, exists := ws.Bases[name]; exists {
		return fmt.Errorf("base %q already exists", name)
	}

	// Check name collision with components
	if _, exists := ws.Components[name]; exists {
		return fmt.Errorf("a component named %q already exists (Error 105: illegal name sharing)", name)
	}

	topNS := ws.Config.TopNamespace
	modulePath := BaseModulePath(topNS, name)
	baseDir := filepath.Join(ws.Dir, "bases", name)

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", baseDir, err)
	}

	// Create go.mod with requires and replaces for all components
	var goMod strings.Builder
	goMod.WriteString(fmt.Sprintf("module %s\n\ngo 1.23\n", modulePath))

	if len(ws.Components) > 0 {
		goMod.WriteString("\nrequire (\n")
		for _, comp := range ws.Components {
			goMod.WriteString(fmt.Sprintf("\t%s v0.0.0\n", comp.Module))
		}
		goMod.WriteString(")\n")

		goMod.WriteString("\nreplace (\n")
		for _, comp := range ws.Components {
			goMod.WriteString(fmt.Sprintf("\t%s => ../../components/%s\n", comp.Module, comp.Name))
		}
		goMod.WriteString(")\n")
	}

	if err := os.WriteFile(filepath.Join(baseDir, "go.mod"), []byte(goMod.String()), 0644); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	// Create main.go
	mainFile := generateBaseMain(name, topNS, modulePath, ws)
	if err := os.WriteFile(filepath.Join(baseDir, "main.go"), []byte(mainFile), 0644); err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}

	// Add to go.work
	relPath := fmt.Sprintf("./bases/%s", name)
	if err := AddToGoWork(ws.Dir, relPath); err != nil {
		return fmt.Errorf("failed to update go.work: %w", err)
	}

	// Update workspace in memory
	ws.Bases[name] = &Base{
		Name:   name,
		Dir:    baseDir,
		Module: modulePath,
	}

	return nil
}

// CreateProject creates a new deployable project
func CreateProject(ws *Workspace, name string) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	if name == "development" {
		return fmt.Errorf("cannot create project named 'development' (reserved)")
	}

	// Check if project already exists
	if _, exists := ws.Projects[name]; exists {
		return fmt.Errorf("project %q already exists", name)
	}

	topNS := ws.Config.TopNamespace
	modulePath := ProjectModulePath(topNS, name)
	projDir := filepath.Join(ws.Dir, "projects", name)

	if err := os.MkdirAll(projDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", projDir, err)
	}

	// Create go.mod with requires and replaces for all bricks
	var goMod strings.Builder
	goMod.WriteString(fmt.Sprintf("module %s\n\ngo 1.23\n", modulePath))

	// Group components by interface — only include one implementation per interface
	ifaceSeen := make(map[string]string) // interface -> component name
	for _, comp := range ws.Components {
		if _, exists := ifaceSeen[comp.Interface]; !exists {
			ifaceSeen[comp.Interface] = comp.Name
		}
	}

	// Build list of modules to include (one per interface + all bases)
	var allModules []string
	for _, compName := range ifaceSeen {
		allModules = append(allModules, ws.Components[compName].Module)
	}
	for _, base := range ws.Bases {
		allModules = append(allModules, base.Module)
	}

	if len(allModules) > 0 {
		goMod.WriteString("\nrequire (\n")
		for _, mod := range allModules {
			goMod.WriteString(fmt.Sprintf("\t%s v0.0.0\n", mod))
		}
		goMod.WriteString(")\n")

		goMod.WriteString("\nreplace (\n")
		for _, compName := range ifaceSeen {
			comp := ws.Components[compName]
			goMod.WriteString(fmt.Sprintf("\t%s => ../../components/%s\n", comp.Module, comp.Name))
		}
		for _, base := range ws.Bases {
			goMod.WriteString(fmt.Sprintf("\t%s => ../../bases/%s\n", base.Module, base.Name))
		}
		goMod.WriteString(")\n")
	}

	if err := os.WriteFile(filepath.Join(projDir, "go.mod"), []byte(goMod.String()), 0644); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	// Create main.go
	mainFile := generateProjectMain(name, topNS, modulePath, ws)
	if err := os.WriteFile(filepath.Join(projDir, "main.go"), []byte(mainFile), 0644); err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}

	// Create test directory
	testDir := filepath.Join(projDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return fmt.Errorf("failed to create test directory: %w", err)
	}

	// Add to go.work
	relPath := fmt.Sprintf("./projects/%s", name)
	if err := AddToGoWork(ws.Dir, relPath); err != nil {
		return fmt.Errorf("failed to update go.work: %w", err)
	}

	// Add to config
	if ws.Config.Projects == nil {
		ws.Config.Projects = make(map[string]ProjectConfig)
	}
	ws.Config.Projects[name] = ProjectConfig{}
	if err := SaveConfig(ws); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Update workspace in memory
	ws.Projects[name] = &Project{
		Name:   name,
		Dir:    projDir,
		Module: modulePath,
	}

	return nil
}

// titleCase converts first letter to uppercase
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Template generation functions
// Using strings.NewReplacer instead of fmt.Sprintf to avoid format verb counting issues

func generateComponentInterface(name, iface, interfacePkg, topNS, modulePath string) string {
	tcName := titleCase(name)
	tcIface := titleCase(iface)

	tmpl := `// Package __IFACEPKG__ provides the public interface for the __NAME__ component.
package __IFACEPKG__

// __TCIFACE__Service is the public interface for the __NAME__ component.
// Other components should depend on this interface, not the implementation.
type __TCIFACE__Service interface {
	// Create creates a new __TCNAME__.
	Create(name string) (*__TCNAME__, error)
	// Get retrieves a __TCNAME__ by ID.
	Get(id string) (*__TCNAME__, error)
	// List returns all __TCNAME__s.
	List() ([]*__TCNAME__, error)
	// Delete removes a __TCNAME__ by ID.
	Delete(id string) error
}

// __TCNAME__ represents a __NAME__ entity.
type __TCNAME__ struct {
	ID   string
	Name string
}
`
	r := strings.NewReplacer(
		"__IFACEPKG__", interfacePkg,
		"__NAME__", name,
		"__TCNAME__", tcName,
		"__TCIFACE__", tcIface,
	)
	return r.Replace(tmpl)
}

func generateComponentImpl(name, iface, interfacePkg, modulePath string) string {
	tcName := titleCase(name)
	tcIface := titleCase(iface)

	tmpl := `package internal

import (
	"fmt"
	"sync"

	"__MODULEPATH__/__IFACEPKG__"
)

// __TCNAME__ implements __IFACEPKG__.__TCIFACE__Service.
type __TCNAME__ struct {
	mu    sync.RWMutex
	items map[string]*__IFACEPKG__.__TCNAME__
}

// New__TCNAME__ creates a new __NAME__ service.
func New__TCNAME__() __IFACEPKG__.__TCIFACE__Service {
	return &__TCNAME__{
		items: make(map[string]*__IFACEPKG__.__TCNAME__),
	}
}

func (s *__TCNAME__) Create(name string) (*__IFACEPKG__.__TCNAME__, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("id-%d", len(s.items)+1)
	item := &__IFACEPKG__.__TCNAME__{ID: id, Name: name}
	s.items[id] = item
	return item, nil
}

func (s *__TCNAME__) Get(id string) (*__IFACEPKG__.__TCNAME__, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("__NAME__ not found: %s", id)
	}
	return item, nil
}

func (s *__TCNAME__) List() ([]*__IFACEPKG__.__TCNAME__, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*__IFACEPKG__.__TCNAME__, 0, len(s.items))
	for _, item := range s.items {
		result = append(result, item)
	}
	return result, nil
}

func (s *__TCNAME__) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("__NAME__ not found: %s", id)
	}
	delete(s.items, id)
	return nil
}
`
	r := strings.NewReplacer(
		"__MODULEPATH__", modulePath,
		"__IFACEPKG__", interfacePkg,
		"__NAME__", name,
		"__TCNAME__", tcName,
		"__TCIFACE__", tcIface,
	)
	return r.Replace(tmpl)
}

func generateComponentTest(name, iface, interfacePkg, modulePath string) string {
	tcName := titleCase(name)
	tcIface := titleCase(iface)

	tmpl := `package test

import (
	"testing"

	"__MODULEPATH__/internal"
)

func Test__TCIFACE___CreateAndGet(t *testing.T) {
	svc := internal.New__TCNAME__()

	item, err := svc.Create("test-__NAME__")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if item.Name != "test-__NAME__" {
		t.Errorf("item.Name = %q, want %q", item.Name, "test-__NAME__")
	}

	got, err := svc.Get(item.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != item.ID {
		t.Errorf("got.ID = %q, want %q", got.ID, item.ID)
	}
}

func Test__TCIFACE___List(t *testing.T) {
	svc := internal.New__TCNAME__()

	_, _ = svc.Create("item1")
	_, _ = svc.Create("item2")

	items, err := svc.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
}

func Test__TCIFACE___Delete(t *testing.T) {
	svc := internal.New__TCNAME__()

	item, _ := svc.Create("to-delete")

	err := svc.Delete(item.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = svc.Get(item.ID)
	if err == nil {
		t.Error("Get() after Delete() should return error")
	}
}
`
	r := strings.NewReplacer(
		"__MODULEPATH__", modulePath,
		"__IFACEPKG__", interfacePkg,
		"__NAME__", name,
		"__TCNAME__", tcName,
		"__TCIFACE__", tcIface,
	)
	return r.Replace(tmpl)
}

func generateBaseMain(name, topNS, modulePath string, ws *Workspace) string {
	interfacePkg := ws.Config.InterfacePkg
	if interfacePkg == "" {
		interfacePkg = "api"
	}

	// Build a compilable main.go that actually uses the imported packages
	var imports []string
	var wires []string

	// Sort component names for deterministic output
	compNames := make([]string, 0, len(ws.Components))
	for n := range ws.Components {
		compNames = append(compNames, n)
	}
	for i := 0; i < len(compNames); i++ {
		for j := i + 1; j < len(compNames); j++ {
			if compNames[i] > compNames[j] {
				compNames[i], compNames[j] = compNames[j], compNames[i]
			}
		}
	}

	for _, compName := range compNames {
		comp := ws.Components[compName]
		tcIface := titleCase(comp.Interface)

		// Import the API package with alias
		imports = append(imports, fmt.Sprintf("\t%[1]sapi \"%[2]s/%[3]s\"", compName, comp.Module, interfacePkg))

		// Wire: reference the service type (compiles without needing internal)
		wires = append(wires, fmt.Sprintf("\tvar %[1]sSvc %[1]sapi.%[2]sService", compName, tcIface))
		wires = append(wires, fmt.Sprintf("\t_ = %[1]sSvc // %[1]s: implements %[2]s", compName, tcIface))
	}

	importBlock := ""
	if len(imports) > 0 {
		importBlock = "import (\n\t\"fmt\"\n" + strings.Join(imports, "\n") + "\n)\n\n"
	} else {
		importBlock = "import \"fmt\"\n\n"
	}

	wireBlock := ""
	if len(wires) > 0 {
		wireBlock = strings.Join(wires, "\n") + "\n"
	}

	return fmt.Sprintf(`// Package main is the %s base — an entry point that wires components together.
package main

%sfunc main() {
	fmt.Printf("Hello from %s base\\n")
%s}
`, name, importBlock, name, wireBlock)
}

func generateProjectMain(name, topNS, modulePath string, ws *Workspace) string {
	interfacePkg := ws.Config.InterfacePkg
	if interfacePkg == "" {
		interfacePkg = "api"
	}

	// A project combines components (via their API packages).
	// Bases are main packages (not importable), so we don't import them directly.
	// Instead, we import the component APIs that the project uses.

	// Group components by interface — include one per interface
	ifaceSeen := make(map[string]*Component)
	for _, comp := range ws.Components {
		if _, exists := ifaceSeen[comp.Interface]; !exists {
			ifaceSeen[comp.Interface] = comp
		}
	}

	var imports []string
	var uses []string

	// Sort interface names for deterministic output
	ifaceNames := make([]string, 0, len(ifaceSeen))
	for iface := range ifaceSeen {
		ifaceNames = append(ifaceNames, iface)
	}
	for i := 0; i < len(ifaceNames); i++ {
		for j := i + 1; j < len(ifaceNames); j++ {
			if ifaceNames[i] > ifaceNames[j] {
				ifaceNames[i], ifaceNames[j] = ifaceNames[j], ifaceNames[i]
			}
		}
	}

	for _, iface := range ifaceNames {
		comp := ifaceSeen[iface]
		tcIface := titleCase(comp.Interface)
		imports = append(imports, fmt.Sprintf("\t%[1]sapi %q", comp.Name, comp.Module+"/"+interfacePkg))
		uses = append(uses, fmt.Sprintf("\tvar _ %[1]sapi.%[2]sService // %[1]s component", comp.Name, tcIface))
	}

	importBlock := ""
	if len(imports) > 0 {
		importBlock = "import (\n\t\"fmt\"\n" + strings.Join(imports, "\n") + "\n)\n\n"
	} else {
		importBlock = "import \"fmt\"\n\n"
	}

	useBlock := ""
	if len(uses) > 0 {
		useBlock = strings.Join(uses, "\n") + "\n"
	}

	return fmt.Sprintf(`// Package main is the entry point for the %s project.
package main

%sfunc main() {
	fmt.Printf("Starting %s project\\n")
%s}
`, name, importBlock, name, useBlock)
}
