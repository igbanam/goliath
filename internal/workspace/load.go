package workspace

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigFileName is the workspace config file name
const ConfigFileName = "goliath.yaml"

// GoWorkFileName is the Go workspace file name
const GoWorkFileName = "go.work"

// FindWorkspaceDir walks up from start looking for goliath.yaml
func FindWorkspaceDir(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	for {
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return "", fmt.Errorf("not in a goliath workspace (no %s found in any parent directory)", ConfigFileName)
		}
		dir = parent
	}
}

// IsWorkspaceDir checks if dir contains a goliath.yaml
func IsWorkspaceDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ConfigFileName))
	return err == nil
}

// Load reads a workspace from the given directory
func Load(dir string) (*Workspace, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Load config
	configPath := filepath.Join(absDir, ConfigFileName)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", ConfigFileName, err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(configData, config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", ConfigFileName, err)
	}

	// Set defaults
	if config.InterfacePkg == "" {
		config.InterfacePkg = "api"
	}
	if config.DefaultProfile == "" {
		config.DefaultProfile = "default"
	}
	if config.VCS.Name == "" {
		config.VCS.Name = "git"
	}
	if config.TagPatterns.Stable == "" {
		config.TagPatterns.Stable = "^stable-.*"
	}
	if config.TagPatterns.Release == "" {
		config.TagPatterns.Release = "^v[0-9].*"
	}
	if config.Projects == nil {
		config.Projects = make(map[string]ProjectConfig)
	}

	ws := &Workspace{
		Dir:        absDir,
		Name:       filepath.Base(absDir),
		Config:     config,
		Components: make(map[string]*Component),
		Bases:      make(map[string]*Base),
		Projects:   make(map[string]*Project),
	}

	// Load components
	if err := ws.loadComponents(); err != nil {
		return nil, fmt.Errorf("failed to load components: %w", err)
	}

	// Load bases
	if err := ws.loadBases(); err != nil {
		return nil, fmt.Errorf("failed to load bases: %w", err)
	}

	// Load projects
	if err := ws.loadProjects(); err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}

	// Always include development project
	if _, ok := ws.Projects["development"]; !ok {
		ws.Projects["development"] = &Project{
			Name:   "development",
			Dir:    filepath.Join(absDir, "development"),
			Module: ProjectModulePath(config.TopNamespace, "development"),
			Alias:  "dev",
		}
	}

	return ws, nil
}

// loadComponents scans the components/ directory
func (w *Workspace) loadComponents() error {
	componentsDir := filepath.Join(w.Dir, "components")
	entries, err := os.ReadDir(componentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No components directory is OK
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		compDir := filepath.Join(componentsDir, name)

		comp := &Component{
			Name: name,
			Dir:  compDir,
		}

		// Parse go.mod
		modulePath, err := parseGoModModule(compDir)
		if err != nil {
			return fmt.Errorf("failed to parse go.mod for component %q: %w", name, err)
		}
		comp.Module = modulePath

		// Determine interface name by analyzing the api/ package
		interfacePkg := w.Config.InterfacePkg
		if interfacePkg == "" {
			interfacePkg = "api"
		}
		comp.Interface = detectComponentInterface(compDir, interfacePkg, name)

		// Parse imports from Go source files
		imports, err := parseImportsFromDir(compDir)
		if err == nil {
			comp.Imports = imports
		}

		w.Components[name] = comp
	}

	return nil
}

// loadBases scans the bases/ directory
func (w *Workspace) loadBases() error {
	basesDir := filepath.Join(w.Dir, "bases")
	entries, err := os.ReadDir(basesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		baseDir := filepath.Join(basesDir, name)

		base := &Base{
			Name: name,
			Dir:  baseDir,
		}

		// Parse go.mod
		modulePath, err := parseGoModModule(baseDir)
		if err != nil {
			return fmt.Errorf("failed to parse go.mod for base %q: %w", name, err)
		}
		base.Module = modulePath

		// Parse imports from Go source files
		imports, err := parseImportsFromDir(baseDir)
		if err == nil {
			base.Imports = imports
		}

		w.Bases[name] = base
	}

	return nil
}

// loadProjects scans the projects/ directory
func (w *Workspace) loadProjects() error {
	projectsDir := filepath.Join(w.Dir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		projDir := filepath.Join(projectsDir, name)

		proj := &Project{
			Name: name,
			Dir:  projDir,
		}

		// Parse go.mod
		modulePath, _, requires, replaces, err := parseGoModFull(projDir)
		if err == nil {
			proj.Module = modulePath
			proj.Requires = requires
			proj.Replaces = replaces
		}

		// Get alias from config
		if pc, ok := w.Config.Projects[name]; ok {
			proj.Alias = pc.Alias
		}

		w.Projects[name] = proj
	}

	return nil
}

// go.mod parsing using simple regex (no external deps)

var (
	moduleRe  = regexp.MustCompile(`^module\s+(\S+)`)
	requireRe = regexp.MustCompile(`^\s*(\S+)\s+v[\d.]+`)
	replaceRe = regexp.MustCompile(`^\s*(\S+)\s+=>\s+(\S+)`)
)

// parseGoModModule extracts the module path from a go.mod file
func parseGoModModule(dir string) (string, error) {
	content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if m := moduleRe.FindStringSubmatch(line); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("no module directive found in go.mod")
}

// parseGoModFull extracts module path, requires, and replaces from go.mod
func parseGoModFull(dir string) (module string, goVersion string, requires []string, replaces []string, err error) {
	content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", "", nil, nil, err
	}

	lines := strings.Split(string(content), "\n")
	inRequire := false
	inReplace := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Module directive
		if m := moduleRe.FindStringSubmatch(trimmed); m != nil {
			module = m[1]
			continue
		}

		// Go version
		if strings.HasPrefix(trimmed, "go ") {
			goVersion = strings.TrimPrefix(trimmed, "go ")
			continue
		}

		// Block start/end
		if strings.HasSuffix(trimmed, "(") {
			if strings.HasPrefix(trimmed, "require") {
				inRequire = true
			}
			if strings.HasPrefix(trimmed, "replace") {
				inReplace = true
			}
			continue
		}
		if trimmed == ")" {
			inRequire = false
			inReplace = false
			continue
		}

		// Single-line require
		if strings.HasPrefix(trimmed, "require ") && !inRequire {
			rest := strings.TrimPrefix(trimmed, "require ")
			if m := requireRe.FindStringSubmatch(rest); m != nil {
				requires = append(requires, m[1])
			}
			continue
		}

		// Single-line replace
		if strings.HasPrefix(trimmed, "replace ") && !inReplace {
			rest := strings.TrimPrefix(trimmed, "replace ")
			if m := replaceRe.FindStringSubmatch(rest); m != nil {
				replaces = append(replaces, m[1])
			}
			continue
		}

		// Inside require block
		if inRequire {
			if m := requireRe.FindStringSubmatch(trimmed); m != nil {
				requires = append(requires, m[1])
			}
			continue
		}

		// Inside replace block
		if inReplace {
			if m := replaceRe.FindStringSubmatch(trimmed); m != nil {
				replaces = append(replaces, m[1])
			}
			continue
		}
	}

	if module == "" {
		return "", "", nil, nil, fmt.Errorf("no module directive found in go.mod")
	}

	return module, goVersion, requires, replaces, nil
}

// parseImportsFromDir parses all .go files in a directory tree and returns unique import paths
func parseImportsFromDir(dir string) ([]string, error) {
	var imports []string
	seen := make(map[string]bool)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil // Skip test files for dependency analysis
		}

		fileImports, err := parseGoFileImports(path)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		for _, imp := range fileImports {
			if !seen[imp] {
				seen[imp] = true
				imports = append(imports, imp)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(imports)
	return imports, nil
}

// parseGoFileImports extracts import paths from a single Go file
func parseGoFileImports(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Use go/parser for robust parsing
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, content, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	var imports []string
	for _, imp := range f.Imports {
		// Remove quotes from import path
		path := strings.Trim(imp.Path.Value, "\"")
		imports = append(imports, path)
	}

	return imports, nil
}

// SaveConfig writes the workspace config to goliath.yaml
func SaveConfig(ws *Workspace) error {
	data, err := yaml.Marshal(ws.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(ws.Dir, ConfigFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", ConfigFileName, err)
	}

	return nil
}

// ReadGoWork reads the go.work file and returns the list of use directives
func ReadGoWork(wsDir string) ([]string, error) {
	content, err := os.ReadFile(filepath.Join(wsDir, GoWorkFileName))
	if err != nil {
		return nil, err
	}

	var uses []string
	lines := strings.Split(string(content), "\n")
	inUse := false

	useRe := regexp.MustCompile(`^\s*(\S+)\s*$`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		if trimmed == "use (" {
			inUse = true
			continue
		}
		if trimmed == ")" && inUse {
			inUse = false
			continue
		}
		if strings.HasPrefix(trimmed, "use ") {
			// Single use directive
			rest := strings.TrimPrefix(trimmed, "use ")
			uses = append(uses, strings.TrimSpace(rest))
			continue
		}
		if inUse {
			if m := useRe.FindStringSubmatch(trimmed); m != nil {
				uses = append(uses, m[1])
			}
		}
	}

	return uses, nil
}

// WriteGoWork writes the go.work file with the given use directives
func WriteGoWork(wsDir string, uses []string) error {
	sort.Strings(uses)

	var sb strings.Builder
	sb.WriteString("go 1.23\n\nuse (\n")
	for _, u := range uses {
		sb.WriteString("\t" + u + "\n")
	}
	sb.WriteString(")\n")

	// Add empty go.work.sum reference if needed
	return os.WriteFile(filepath.Join(wsDir, GoWorkFileName), []byte(sb.String()), 0644)
}

// AddToGoWork adds a module path to the go.work file
func AddToGoWork(wsDir, modulePath string) error {
	uses, err := ReadGoWork(wsDir)
	if err != nil {
		// Create new go.work
		uses = []string{}
	}

	// Check if already present
	for _, u := range uses {
		if u == modulePath {
			return nil // Already present
		}
	}

	uses = append(uses, modulePath)
	return WriteGoWork(wsDir, uses)
}
