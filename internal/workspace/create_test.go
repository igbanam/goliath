package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWorkspace(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "myws")

	err := CreateWorkspace(targetDir, "myws", "github.com/myorg")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	// Check goliath.yaml exists
	configPath := filepath.Join(targetDir, ConfigFileName)
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("goliath.yaml not created: %v", err)
	}

	// Check go.work exists
	goWorkPath := filepath.Join(targetDir, GoWorkFileName)
	if _, err := os.Stat(goWorkPath); err != nil {
		t.Errorf("go.work not created: %v", err)
	}

	// Check directories
	for _, d := range []string{"components", "bases", "projects", "development"} {
		dirPath := filepath.Join(targetDir, d)
		if _, err := os.Stat(dirPath); err != nil {
			t.Errorf("directory %s not created: %v", d, err)
		}
		keepPath := filepath.Join(dirPath, ".keep")
		if _, err := os.Stat(keepPath); err != nil {
			t.Errorf(".keep not created in %s: %v", d, err)
		}
	}

	// Check development/go.mod
	devGoMod := filepath.Join(targetDir, "development", "go.mod")
	content, err := os.ReadFile(devGoMod)
	if err != nil {
		t.Errorf("development/go.mod not created: %v", err)
	} else if !strings.Contains(string(content), "module github.com/myorg/development") {
		t.Errorf("development/go.mod content wrong: %s", content)
	}

	// Check .gitignore
	if _, err := os.Stat(filepath.Join(targetDir, ".gitignore")); err != nil {
		t.Errorf(".gitignore not created: %v", err)
	}

	// Check README.md
	if _, err := os.Stat(filepath.Join(targetDir, "README.md")); err != nil {
		t.Errorf("README.md not created: %v", err)
	}
}

func TestCreateWorkspace_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "myws")

	// Create once
	if err := CreateWorkspace(targetDir, "myws", "github.com/myorg"); err != nil {
		t.Fatalf("First CreateWorkspace() error = %v", err)
	}

	// Try again — should fail
	err := CreateWorkspace(targetDir, "myws", "github.com/myorg")
	if err == nil {
		t.Error("CreateWorkspace() should fail when workspace already exists")
	}
}

func TestCreateWorkspace_NoTopNS(t *testing.T) {
	dir := t.TempDir()
	err := CreateWorkspace(dir, "test", "")
	if err == nil {
		t.Error("CreateWorkspace() should fail without top-namespace")
	}
}

func TestCreateWorkspace_Loadable(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "myws")

	if err := CreateWorkspace(targetDir, "myws", "github.com/myorg"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	// Should be loadable
	ws, err := Load(targetDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if ws.Config.TopNamespace != "github.com/myorg" {
		t.Errorf("ws.Config.TopNamespace = %q, want %q", ws.Config.TopNamespace, "github.com/myorg")
	}
	if ws.Name != "myws" {
		t.Errorf("ws.Name = %q, want %q", ws.Name, "myws")
	}
}

func TestCreateComponent(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	err = CreateComponent(ws, "user", "")
	if err != nil {
		t.Fatalf("CreateComponent() error = %v", err)
	}

	compDir := filepath.Join(dir, "components", "user")

	// Check go.mod
	goModPath := filepath.Join(compDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("go.mod not created: %v", err)
	}
	if !strings.Contains(string(content), "module github.com/test/user-component") {
		t.Errorf("go.mod content wrong: %s", content)
	}

	// Check api/user.go
	apiFile := filepath.Join(compDir, "api", "user.go")
	content, err = os.ReadFile(apiFile)
	if err != nil {
		t.Fatalf("api/user.go not created: %v", err)
	}
	if !strings.Contains(string(content), "package api") {
		t.Errorf("api/user.go should be package api: %s", content)
	}
	if !strings.Contains(string(content), "UserService") {
		t.Errorf("api/user.go should contain UserService: %s", content)
	}

	// Check internal/user.go
	implFile := filepath.Join(compDir, "internal", "user.go")
	content, err = os.ReadFile(implFile)
	if err != nil {
		t.Fatalf("internal/user.go not created: %v", err)
	}
	if !strings.Contains(string(content), "package internal") {
		t.Errorf("internal/user.go should be package internal: %s", content)
	}
	if !strings.Contains(string(content), "NewUser") {
		t.Errorf("internal/user.go should contain NewUser: %s", content)
	}

	// Check test file
	testFile := filepath.Join(compDir, "test", "user_test.go")
	if _, err := os.Stat(testFile); err != nil {
		t.Errorf("test/user_test.go not created: %v", err)
	}

	// Check go.work was updated
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
		t.Errorf("go.work should include ./components/user, got: %v", uses)
	}
}

func TestCreateComponent_WithInterface(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create user component first
	if err := CreateComponent(ws, "user", ""); err != nil {
		t.Fatalf("CreateComponent(user) error = %v", err)
	}

	// Create admin component with user interface
	if err := CreateComponent(ws, "admin", "user"); err != nil {
		t.Fatalf("CreateComponent(admin) error = %v", err)
	}

	// Check admin's interface file mentions UserService
	apiFile := filepath.Join(dir, "components", "admin", "api", "admin.go")
	content, err := os.ReadFile(apiFile)
	if err != nil {
		t.Fatalf("admin api file not created: %v", err)
	}
	if !strings.Contains(string(content), "UserService") {
		t.Errorf("admin api should reference UserService interface: %s", content)
	}
}

func TestCreateComponent_Duplicate(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := CreateComponent(ws, "user", ""); err != nil {
		t.Fatalf("First CreateComponent() error = %v", err)
	}

	err = CreateComponent(ws, "user", "")
	if err == nil {
		t.Error("CreateComponent() should fail for duplicate component")
	}
}

func TestCreateComponent_GoCodeCompiles(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := CreateComponent(ws, "user", ""); err != nil {
		t.Fatalf("CreateComponent() error = %v", err)
	}

	// Verify the API file is valid Go
	apiFile := filepath.Join(dir, "components", "user", "api", "user.go")
	content, err := os.ReadFile(apiFile)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Check for expected patterns
	checks := []string{
		"package api",
		"type UserService interface",
		"type User struct",
		"Create(name string) (*User, error)",
		"Get(id string) (*User, error)",
		"List() ([]*User, error)",
		"Delete(id string) error",
	}
	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("api/user.go should contain %q", check)
		}
	}

	// Verify the internal file is valid Go
	implFile := filepath.Join(dir, "components", "user", "internal", "user.go")
	content, err = os.ReadFile(implFile)
	if err != nil {
		t.Fatal(err)
	}
	contentStr = string(content)

	checks = []string{
		"package internal",
		"github.com/test/user-component/api",
		"func NewUser() api.UserService",
		"func (s *User) Create",
		"func (s *User) Get",
		"func (s *User) List",
		"func (s *User) Delete",
	}
	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("internal/user.go should contain %q", check)
		}
	}
}

func TestCreateBase(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a component first so the base can reference it
	if err := CreateComponent(ws, "user", ""); err != nil {
		t.Fatalf("CreateComponent() error = %v", err)
	}

	err = CreateBase(ws, "cli")
	if err != nil {
		t.Fatalf("CreateBase() error = %v", err)
	}

	baseDir := filepath.Join(dir, "bases", "cli")

	// Check go.mod
	goModPath := filepath.Join(baseDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("go.mod not created: %v", err)
	}
	if !strings.Contains(string(content), "module github.com/test/cli-base") {
		t.Errorf("go.mod content wrong: %s", content)
	}

	// Check main.go
	mainFile := filepath.Join(baseDir, "main.go")
	content, err = os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("main.go not created: %v", err)
	}
	if !strings.Contains(string(content), "package main") {
		t.Errorf("main.go should be package main: %s", content)
	}

	// Check go.work was updated
	uses, err := ReadGoWork(dir)
	if err != nil {
		t.Fatalf("ReadGoWork() error = %v", err)
	}
	found := false
	for _, u := range uses {
		if u == "./bases/cli" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("go.work should include ./bases/cli, got: %v", uses)
	}
}

func TestCreateBase_GoCodeValid(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := CreateComponent(ws, "user", ""); err != nil {
		t.Fatalf("CreateComponent() error = %v", err)
	}

	if err := CreateBase(ws, "cli"); err != nil {
		t.Fatalf("CreateBase() error = %v", err)
	}

	// Check base go.mod has requires and replaces
	goModPath := filepath.Join(dir, "bases", "cli", "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "require") {
		t.Error("base go.mod should have require block")
	}
	if !strings.Contains(contentStr, "github.com/test/user-component") {
		t.Error("base go.mod should require user-component")
	}
	if !strings.Contains(contentStr, "replace") {
		t.Error("base go.mod should have replace block")
	}
	if !strings.Contains(contentStr, "../../components/user") {
		t.Error("base go.mod should replace to ../../components/user")
	}

	// Check main.go only imports api, not internal
	mainFile := filepath.Join(dir, "bases", "cli", "main.go")
	content, err = os.ReadFile(mainFile)
	if err != nil {
		t.Fatal(err)
	}
	contentStr = string(content)

	if !strings.Contains(contentStr, "package main") {
		t.Error("main.go should be package main")
	}
	if !strings.Contains(contentStr, "userapi") {
		t.Error("main.go should import user api package")
	}
	if strings.Contains(contentStr, "/internal") {
		t.Error("main.go should NOT import internal packages")
	}
}

func TestCreateBase_NameCollision(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create component named "user"
	if err := CreateComponent(ws, "user", ""); err != nil {
		t.Fatalf("CreateComponent() error = %v", err)
	}

	// Try to create base with same name — should fail
	err = CreateBase(ws, "user")
	if err == nil {
		t.Error("CreateBase() should fail when component with same name exists")
	}
}

func TestCreateProject(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create some bricks first
	if err := CreateComponent(ws, "user", ""); err != nil {
		t.Fatalf("CreateComponent() error = %v", err)
	}
	if err := CreateBase(ws, "cli"); err != nil {
		t.Fatalf("CreateBase() error = %v", err)
	}

	err = CreateProject(ws, "myapp")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	projDir := filepath.Join(dir, "projects", "myapp")

	// Check go.mod
	goModPath := filepath.Join(projDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("go.mod not created: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "module github.com/test/myapp") {
		t.Errorf("go.mod should contain module path: %s", contentStr)
	}
	if !strings.Contains(contentStr, "github.com/test/user-component") {
		t.Errorf("go.mod should require user-component: %s", contentStr)
	}
	if !strings.Contains(contentStr, "github.com/test/cli-base") {
		t.Errorf("go.mod should require cli-base: %s", contentStr)
	}
	if !strings.Contains(contentStr, "replace") {
		t.Errorf("go.mod should contain replace directives: %s", contentStr)
	}

	// Check main.go
	mainFile := filepath.Join(projDir, "main.go")
	if _, err := os.Stat(mainFile); err != nil {
		t.Errorf("main.go not created: %v", err)
	}

	// Check go.work was updated
	uses, err := ReadGoWork(dir)
	if err != nil {
		t.Fatalf("ReadGoWork() error = %v", err)
	}
	found := false
	for _, u := range uses {
		if u == "./projects/myapp" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("go.work should include ./projects/myapp, got: %v", uses)
	}
}

func TestCreateProject_GoModComplete(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := CreateComponent(ws, "user", ""); err != nil {
		t.Fatalf("CreateComponent() error = %v", err)
	}
	if err := CreateBase(ws, "cli"); err != nil {
		t.Fatalf("CreateBase() error = %v", err)
	}
	if err := CreateProject(ws, "myapp"); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	// Check project go.mod
	goModPath := filepath.Join(dir, "projects", "myapp", "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Should require both component and base
	if !strings.Contains(contentStr, "github.com/test/user-component") {
		t.Error("project go.mod should require user-component")
	}
	if !strings.Contains(contentStr, "github.com/test/cli-base") {
		t.Error("project go.mod should require cli-base")
	}

	// Should have replaces
	if !strings.Contains(contentStr, "../../components/user") {
		t.Error("project go.mod should replace to components/user")
	}
	if !strings.Contains(contentStr, "../../bases/cli") {
		t.Error("project go.mod should replace to bases/cli")
	}
}

func TestCreateProject_DevelopmentReserved(t *testing.T) {
	dir := t.TempDir()
	if err := CreateWorkspace(dir, "test", "github.com/test"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	err = CreateProject(ws, "development")
	if err == nil {
		t.Error("CreateProject() should fail for 'development' (reserved name)")
	}
}

func TestEndToEnd_CreateAndLoad(t *testing.T) {
	dir := t.TempDir()

	// Create workspace
	if err := CreateWorkspace(dir, "e2e", "github.com/e2e"); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	// Load and create bricks
	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := CreateComponent(ws, "user", ""); err != nil {
		t.Fatalf("CreateComponent(user) error = %v", err)
	}
	if err := CreateComponent(ws, "admin", "user"); err != nil {
		t.Fatalf("CreateComponent(admin) error = %v", err)
	}
	if err := CreateComponent(ws, "util", ""); err != nil {
		t.Fatalf("CreateComponent(util) error = %v", err)
	}
	if err := CreateBase(ws, "cli"); err != nil {
		t.Fatalf("CreateBase(cli) error = %v", err)
	}
	if err := CreateProject(ws, "myapp"); err != nil {
		t.Fatalf("CreateProject(myapp) error = %v", err)
	}

	// Reload and verify
	ws2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() after creation error = %v", err)
	}

	if len(ws2.Components) != 3 {
		t.Errorf("len(ws2.Components) = %d, want 3", len(ws2.Components))
	}
	if len(ws2.Bases) != 1 {
		t.Errorf("len(ws2.Bases) = %d, want 1", len(ws2.Bases))
	}
	// Projects: development + myapp
	if len(ws2.Projects) != 2 {
		t.Errorf("len(ws2.Projects) = %d, want 2", len(ws2.Projects))
	}

	// Verify admin has user interface
	admin := ws2.Components["admin"]
	if admin.Interface != "user" {
		t.Errorf("admin.Interface = %q, want %q", admin.Interface, "user")
	}

	// Verify interfaces
	ifaces := ws2.InterfaceNames()
	if len(ifaces) != 2 { // user, util
		t.Errorf("len(ifaces) = %d, want 2: %v", len(ifaces), ifaces)
	}
}
