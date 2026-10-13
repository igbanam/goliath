package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper to create a temporary workspace for testing
func createTestWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create goliath.yaml
	config := `top-namespace: github.com/test
interface-pkg: api
default-profile: default
vcs:
  name: git
  auto-add: false
tag-patterns:
  stable: "^stable-.*"
  release: "^v[0-9].*"
projects:
  development:
    alias: dev
`
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	// Create directories
	for _, d := range []string{"components", "bases", "projects", "development"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create go.work
	goWork := `go 1.23

use (
	./development
)
`
	if err := os.WriteFile(filepath.Join(dir, GoWorkFileName), []byte(goWork), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestFindWorkspaceDir(t *testing.T) {
	dir := createTestWorkspace(t)

	// Should find from the root
	found, err := FindWorkspaceDir(dir)
	if err != nil {
		t.Fatalf("FindWorkspaceDir(%q) error = %v", dir, err)
	}
	if found != dir {
		t.Errorf("FindWorkspaceDir(%q) = %q, want %q", dir, found, dir)
	}

	// Should find from a subdirectory
	subDir := filepath.Join(dir, "components", "user")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	found, err = FindWorkspaceDir(subDir)
	if err != nil {
		t.Fatalf("FindWorkspaceDir(%q) error = %v", subDir, err)
	}
	if found != dir {
		t.Errorf("FindWorkspaceDir(%q) = %q, want %q", subDir, found, dir)
	}
}

func TestFindWorkspaceDir_NotFound(t *testing.T) {
	// Use a temp dir that definitely doesn't have goliath.yaml in any parent
	dir := t.TempDir()
	_, err := FindWorkspaceDir(dir)
	if err == nil {
		t.Error("FindWorkspaceDir() should return error when not in a workspace")
	}
}

func TestIsWorkspaceDir(t *testing.T) {
	dir := createTestWorkspace(t)
	if !IsWorkspaceDir(dir) {
		t.Error("IsWorkspaceDir() = false, want true")
	}

	nonWsDir := t.TempDir()
	if IsWorkspaceDir(nonWsDir) {
		t.Error("IsWorkspaceDir() = true for empty dir, want false")
	}
}

func TestLoad_BasicWorkspace(t *testing.T) {
	dir := createTestWorkspace(t)

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if ws.Name != filepath.Base(dir) {
		t.Errorf("ws.Name = %q, want %q", ws.Name, filepath.Base(dir))
	}
	if ws.Config.TopNamespace != "github.com/test" {
		t.Errorf("ws.Config.TopNamespace = %q, want %q", ws.Config.TopNamespace, "github.com/test")
	}
	if ws.Config.InterfacePkg != "api" {
		t.Errorf("ws.Config.InterfacePkg = %q, want %q", ws.Config.InterfacePkg, "api")
	}

	// Should always have development project
	if _, ok := ws.Projects["development"]; !ok {
		t.Error("ws.Projects should include 'development'")
	}
}

func TestLoad_WithComponents(t *testing.T) {
	dir := createTestWorkspace(t)

	// Create a component
	compDir := filepath.Join(dir, "components", "user")
	if err := os.MkdirAll(filepath.Join(compDir, "api"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(compDir, "internal"), 0755); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/user-component

go 1.23
`
	if err := os.WriteFile(filepath.Join(compDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	apiFile := `package api

type UserService interface {
	Create(name string) error
}
`
	if err := os.WriteFile(filepath.Join(compDir, "api", "user.go"), []byte(apiFile), 0644); err != nil {
		t.Fatal(err)
	}

	implFile := `package internal

import "github.com/test/user-component/api"

type userService struct{}
`
	if err := os.WriteFile(filepath.Join(compDir, "internal", "user.go"), []byte(implFile), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(ws.Components) != 1 {
		t.Fatalf("len(ws.Components) = %d, want 1", len(ws.Components))
	}

	comp := ws.Components["user"]
	if comp == nil {
		t.Fatal("ws.Components[\"user\"] is nil")
	}
	if comp.Module != "github.com/test/user-component" {
		t.Errorf("comp.Module = %q, want %q", comp.Module, "github.com/test/user-component")
	}
	if comp.Interface != "user" {
		t.Errorf("comp.Interface = %q, want %q", comp.Interface, "user")
	}
}

func TestLoad_WithBases(t *testing.T) {
	dir := createTestWorkspace(t)

	// Create a base
	baseDir := filepath.Join(dir, "bases", "cli")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/cli-base

go 1.23
`
	if err := os.WriteFile(filepath.Join(baseDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	mainFile := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	if err := os.WriteFile(filepath.Join(baseDir, "main.go"), []byte(mainFile), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(ws.Bases) != 1 {
		t.Fatalf("len(ws.Bases) = %d, want 1", len(ws.Bases))
	}

	base := ws.Bases["cli"]
	if base == nil {
		t.Fatal("ws.Bases[\"cli\"] is nil")
	}
	if base.Module != "github.com/test/cli-base" {
		t.Errorf("base.Module = %q, want %q", base.Module, "github.com/test/cli-base")
	}
}

func TestLoad_WithProjects(t *testing.T) {
	dir := createTestWorkspace(t)

	// Create a project
	projDir := filepath.Join(dir, "projects", "myapp")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/test/myapp

go 1.23

require github.com/test/cli-base v0.0.0

replace github.com/test/cli-base => ../../bases/cli
`
	if err := os.WriteFile(filepath.Join(projDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(ws.Projects) < 2 { // development + myapp
		t.Fatalf("len(ws.Projects) = %d, want at least 2", len(ws.Projects))
	}

	proj := ws.Projects["myapp"]
	if proj == nil {
		t.Fatal("ws.Projects[\"myapp\"] is nil")
	}
	if proj.Module != "github.com/test/myapp" {
		t.Errorf("proj.Module = %q, want %q", proj.Module, "github.com/test/myapp")
	}
}

func TestSaveConfig(t *testing.T) {
	dir := createTestWorkspace(t)

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Modify config
	ws.Config.TopNamespace = "github.com/modified"

	if err := SaveConfig(ws); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Reload and verify
	ws2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() after save error = %v", err)
	}
	if ws2.Config.TopNamespace != "github.com/modified" {
		t.Errorf("ws2.Config.TopNamespace = %q, want %q", ws2.Config.TopNamespace, "github.com/modified")
	}
}

func TestReadGoWork(t *testing.T) {
	dir := t.TempDir()

	goWork := `go 1.23

use (
	./bases/cli
	./components/user
	./development
)
`
	if err := os.WriteFile(filepath.Join(dir, GoWorkFileName), []byte(goWork), 0644); err != nil {
		t.Fatal(err)
	}

	uses, err := ReadGoWork(dir)
	if err != nil {
		t.Fatalf("ReadGoWork() error = %v", err)
	}

	if len(uses) != 3 {
		t.Fatalf("ReadGoWork() returned %d uses, want 3", len(uses))
	}
}

func TestWriteGoWork(t *testing.T) {
	dir := t.TempDir()

	uses := []string{"./development", "./components/user", "./bases/cli"}
	if err := WriteGoWork(dir, uses); err != nil {
		t.Fatalf("WriteGoWork() error = %v", err)
	}

	// Read back
	readUses, err := ReadGoWork(dir)
	if err != nil {
		t.Fatalf("ReadGoWork() after write error = %v", err)
	}

	if len(readUses) != 3 {
		t.Fatalf("ReadGoWork() returned %d uses, want 3", len(readUses))
	}

	// Should be sorted
	if readUses[0] != "./bases/cli" {
		t.Errorf("uses[0] = %q, want %q", readUses[0], "./bases/cli")
	}
}

func TestAddToGoWork(t *testing.T) {
	dir := createTestWorkspace(t)

	// Add a module
	if err := AddToGoWork(dir, "./components/user"); err != nil {
		t.Fatalf("AddToGoWork() error = %v", err)
	}

	uses, err := ReadGoWork(dir)
	if err != nil {
		t.Fatalf("ReadGoWork() error = %v", err)
	}

	found := false
	for _, u := range uses {
		if u == "./components/user" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AddToGoWork() did not add the module")
	}

	// Adding again should be idempotent
	if err := AddToGoWork(dir, "./components/user"); err != nil {
		t.Fatalf("AddToGoWork() second call error = %v", err)
	}

	uses2, _ := ReadGoWork(dir)
	count := 0
	for _, u := range uses2 {
		if u == "./components/user" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("AddToGoWork() added duplicate, count = %d, want 1", count)
	}
}

func TestParseGoModModule(t *testing.T) {
	dir := t.TempDir()

	goMod := `module github.com/test/my-module

go 1.23

require (
	github.com/spf13/cobra v1.7.0
)
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	module, err := parseGoModModule(dir)
	if err != nil {
		t.Fatalf("parseGoModModule() error = %v", err)
	}
	if module != "github.com/test/my-module" {
		t.Errorf("parseGoModModule() = %q, want %q", module, "github.com/test/my-module")
	}
}

func TestParseGoModFull(t *testing.T) {
	dir := t.TempDir()

	goMod := `module github.com/test/my-module

go 1.23

require (
	github.com/spf13/cobra v1.7.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/test/dep => ../dep
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	module, goVer, requires, replaces, err := parseGoModFull(dir)
	if err != nil {
		t.Fatalf("parseGoModFull() error = %v", err)
	}

	if module != "github.com/test/my-module" {
		t.Errorf("module = %q, want %q", module, "github.com/test/my-module")
	}
	if goVer != "1.23" {
		t.Errorf("goVer = %q, want %q", goVer, "1.23")
	}
	if len(requires) != 2 {
		t.Errorf("len(requires) = %d, want 2", len(requires))
	}
	if len(replaces) != 1 {
		t.Errorf("len(replaces) = %d, want 1", len(replaces))
	}
	if replaces[0] != "github.com/test/dep" {
		t.Errorf("replaces[0] = %q, want %q", replaces[0], "github.com/test/dep")
	}
}

func TestParseGoFileImports(t *testing.T) {
	dir := t.TempDir()

	goFile := `package main

import (
	"fmt"
	"os"

	userapi "github.com/test/user-component/api"
	_ "github.com/test/some-init"
)

func main() {
	fmt.Println("hello")
	_ = os.Args
	_ = userapi.UserService(nil)
}
`
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte(goFile), 0644); err != nil {
		t.Fatal(err)
	}

	imports, err := parseGoFileImports(filePath)
	if err != nil {
		t.Fatalf("parseGoFileImports() error = %v", err)
	}

	want := []string{
		"fmt",
		"github.com/test/some-init",
		"github.com/test/user-component/api",
		"os",
	}
	if len(imports) != len(want) {
		t.Fatalf("parseGoFileImports() returned %d imports, want %d: %v", len(imports), len(want), imports)
	}
	for _, w := range want {
		found := false
		for _, imp := range imports {
			if imp == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("parseGoFileImports() missing %q in %v", w, imports)
		}
	}
}

func TestParseGoFileImports_SingleImport(t *testing.T) {
	dir := t.TempDir()

	goFile := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte(goFile), 0644); err != nil {
		t.Fatal(err)
	}

	imports, err := parseGoFileImports(filePath)
	if err != nil {
		t.Fatalf("parseGoFileImports() error = %v", err)
	}
	if len(imports) != 1 || imports[0] != "fmt" {
		t.Errorf("parseGoFileImports() = %v, want [fmt]", imports)
	}
}
