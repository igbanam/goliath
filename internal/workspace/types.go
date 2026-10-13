package workspace

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Config represents goliath.yaml
type Config struct {
	TopNamespace   string                   `yaml:"top-namespace" json:"top-namespace"`
	InterfacePkg   string                   `yaml:"interface-pkg" json:"interface-pkg"`
	DefaultProfile string                   `yaml:"default-profile" json:"default-profile"`
	VCS            VCSConfig                `yaml:"vcs" json:"vcs"`
	TagPatterns    TagPatterns              `yaml:"tag-patterns" json:"tag-patterns"`
	Projects       map[string]ProjectConfig `yaml:"projects" json:"projects"`
}

// VCSConfig holds VCS settings
type VCSConfig struct {
	Name    string `yaml:"name" json:"name"`
	AutoAdd bool   `yaml:"auto-add" json:"auto-add"`
}

// TagPatterns holds git tag patterns
type TagPatterns struct {
	Stable  string `yaml:"stable" json:"stable"`
	Release string `yaml:"release" json:"release"`
}

// ProjectConfig holds per-project config
type ProjectConfig struct {
	Alias string `yaml:"alias" json:"alias"`
}

// Workspace represents the full workspace state
type Workspace struct {
	Dir        string              `json:"ws-dir" yaml:"-"`
	Name       string              `json:"name" yaml:"-"`
	Config     *Config             `json:"settings" yaml:"-"`
	Components map[string]*Component `json:"components" yaml:"-"`
	Bases      map[string]*Base      `json:"bases" yaml:"-"`
	Projects   map[string]*Project   `json:"projects" yaml:"-"`
}

// ToJSON returns the workspace as indented JSON
func (w *Workspace) ToJSON() ([]byte, error) {
	return json.MarshalIndent(w, "", "  ")
}

// Brick is the common interface for components and bases
type Brick interface {
	BrickName() string
	BrickPath() string
	ModulePath() string
	BrickInterfaces() []string
	BrickImports() []string
	IsBase() bool
	BrickType() string
}

// Component represents a component brick
type Component struct {
	Name      string   `json:"name" yaml:"name"`
	Dir       string   `json:"path" yaml:"path"`
	Module    string   `json:"module" yaml:"module"`
	Interface string   `json:"interface" yaml:"interface"`
	Imports   []string `json:"imports" yaml:"imports,omitempty"`
	Changed   bool     `json:"changed,omitempty" yaml:"-"`
}

// BrickName returns the component name
func (c *Component) BrickName() string { return c.Name }

// BrickPath returns the component directory
func (c *Component) BrickPath() string { return c.Dir }

// ModulePath returns the Go module path
func (c *Component) ModulePath() string { return c.Module }

// BrickInterfaces returns the interfaces this component implements
func (c *Component) BrickInterfaces() []string { return []string{c.Interface} }

// BrickImports returns the import paths used by this component
func (c *Component) BrickImports() []string { return c.Imports }

// IsBase returns false for components
func (c *Component) IsBase() bool { return false }

// BrickType returns "component"
func (c *Component) BrickType() string { return "component" }

// Base represents a base brick
type Base struct {
	Name    string   `json:"name" yaml:"name"`
	Dir     string   `json:"path" yaml:"path"`
	Module  string   `json:"module" yaml:"module"`
	Imports []string `json:"imports" yaml:"imports,omitempty"`
	Changed bool     `json:"changed,omitempty" yaml:"-"`
}

// BrickName returns the base name
func (b *Base) BrickName() string { return b.Name }

// BrickPath returns the base directory
func (b *Base) BrickPath() string { return b.Dir }

// ModulePath returns the Go module path
func (b *Base) ModulePath() string { return b.Module }

// BrickInterfaces returns empty for bases
func (b *Base) BrickInterfaces() []string { return nil }

// BrickImports returns the import paths used by this base
func (b *Base) BrickImports() []string { return b.Imports }

// IsBase returns true for bases
func (b *Base) IsBase() bool { return true }

// BrickType returns "base"
func (b *Base) BrickType() string { return "base" }

// Project represents a deployable project
type Project struct {
	Name     string   `json:"name" yaml:"name"`
	Dir      string   `json:"path" yaml:"path"`
	Module   string   `json:"module" yaml:"module"`
	Alias    string   `json:"alias,omitempty" yaml:"alias,omitempty"`
	Requires []string `json:"requires" yaml:"requires,omitempty"`
	Replaces []string `json:"replaces" yaml:"replaces,omitempty"`
	Changed  bool     `json:"changed,omitempty" yaml:"-"`
}

// AllBricks returns all bricks (components + bases) sorted by name
func (w *Workspace) AllBricks() []Brick {
	var bricks []Brick
	for _, c := range w.Components {
		bricks = append(bricks, c)
	}
	for _, b := range w.Bases {
		bricks = append(bricks, b)
	}
	sort.Slice(bricks, func(i, j int) bool {
		return bricks[i].BrickName() < bricks[j].BrickName()
	})
	return bricks
}

// BrickByName finds a brick by name
func (w *Workspace) BrickByName(name string) Brick {
	if c, ok := w.Components[name]; ok {
		return c
	}
	if b, ok := w.Bases[name]; ok {
		return b
	}
	return nil
}

// ComponentNames returns sorted component names
func (w *Workspace) ComponentNames() []string {
	names := make([]string, 0, len(w.Components))
	for name := range w.Components {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// BaseNames returns sorted base names
func (w *Workspace) BaseNames() []string {
	names := make([]string, 0, len(w.Bases))
	for name := range w.Bases {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ProjectNames returns sorted project names
func (w *Workspace) ProjectNames() []string {
	names := make([]string, 0, len(w.Projects))
	for name := range w.Projects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// InterfaceNames returns sorted unique interface names from all components
func (w *Workspace) InterfaceNames() []string {
	seen := make(map[string]bool)
	var names []string
	for _, c := range w.Components {
		if !seen[c.Interface] {
			seen[c.Interface] = true
			names = append(names, c.Interface)
		}
	}
	sort.Strings(names)
	return names
}

// ComponentsForInterface returns all components implementing the given interface
func (w *Workspace) ComponentsForInterface(iface string) []*Component {
	var result []*Component
	for _, c := range w.Components {
		if c.Interface == iface {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// ModulePathForComponent returns the module path for a component name
func (w *Workspace) ModulePathForComponent(name string) string {
	if c, ok := w.Components[name]; ok {
		return c.Module
	}
	return ""
}

// ModulePathForBase returns the module path for a base name
func (w *Workspace) ModulePathForBase(name string) string {
	if b, ok := w.Bases[name]; ok {
		return b.Module
	}
	return ""
}

// String returns a human-readable summary
func (w *Workspace) String() string {
	return fmt.Sprintf("workspace %q at %s: %d components, %d bases, %d projects",
		w.Name, w.Dir, len(w.Components), len(w.Bases), len(w.Projects))
}

// ComponentModulePath returns the expected module path for a component
func ComponentModulePath(topNS, name string) string {
	return fmt.Sprintf("%s/%s-component", topNS, name)
}

// BaseModulePath returns the expected module path for a base
func BaseModulePath(topNS, name string) string {
	return fmt.Sprintf("%s/%s-base", topNS, name)
}

// ProjectModulePath returns the expected module path for a project
func ProjectModulePath(topNS, name string) string {
	if name == "development" {
		return fmt.Sprintf("%s/development", topNS)
	}
	return fmt.Sprintf("%s/%s", topNS, name)
}

// ImportIsComponentAPI checks if an import path refers to a component's API package
func ImportIsComponentAPI(importPath, modulePath, interfacePkg string) bool {
	expected := fmt.Sprintf("%s/%s", modulePath, interfacePkg)
	return importPath == expected || strings.HasPrefix(importPath, expected+"/")
}

// ImportBelongsToComponent checks if an import belongs to a given component module
func ImportBelongsToComponent(importPath, modulePath string) bool {
	return importPath == modulePath || strings.HasPrefix(importPath, modulePath+"/")
}
