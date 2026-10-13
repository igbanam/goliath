package workspace

import (
	"encoding/json"
	"testing"
)

func TestComponentModulePath(t *testing.T) {
	got := ComponentModulePath("github.com/myorg", "user")
	want := "github.com/myorg/user-component"
	if got != want {
		t.Errorf("ComponentModulePath() = %q, want %q", got, want)
	}
}

func TestBaseModulePath(t *testing.T) {
	got := BaseModulePath("github.com/myorg", "cli")
	want := "github.com/myorg/cli-base"
	if got != want {
		t.Errorf("BaseModulePath() = %q, want %q", got, want)
	}
}

func TestProjectModulePath(t *testing.T) {
	tests := []struct {
		name   string
		topNS  string
		proj   string
		want   string
	}{
		{"regular project", "github.com/myorg", "myapp", "github.com/myorg/myapp"},
		{"development", "github.com/myorg", "development", "github.com/myorg/development"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProjectModulePath(tt.topNS, tt.proj)
			if got != tt.want {
				t.Errorf("ProjectModulePath(%q, %q) = %q, want %q", tt.topNS, tt.proj, got, tt.want)
			}
		})
	}
}

func TestWorkspaceAllBricks(t *testing.T) {
	ws := &Workspace{
		Components: map[string]*Component{
			"user":  {Name: "user", Interface: "user"},
			"admin": {Name: "admin", Interface: "user"},
		},
		Bases: map[string]*Base{
			"cli": {Name: "cli"},
		},
	}

	bricks := ws.AllBricks()
	if len(bricks) != 3 {
		t.Fatalf("AllBricks() returned %d bricks, want 3", len(bricks))
	}

	// Should be sorted by name: admin, cli, user
	if bricks[0].BrickName() != "admin" {
		t.Errorf("bricks[0] = %q, want %q", bricks[0].BrickName(), "admin")
	}
	if bricks[1].BrickName() != "cli" {
		t.Errorf("bricks[1] = %q, want %q", bricks[1].BrickName(), "cli")
	}
	if bricks[2].BrickName() != "user" {
		t.Errorf("bricks[2] = %q, want %q", bricks[2].BrickName(), "user")
	}
}

func TestWorkspaceBrickByName(t *testing.T) {
	ws := &Workspace{
		Components: map[string]*Component{
			"user": {Name: "user", Interface: "user"},
		},
		Bases: map[string]*Base{
			"cli": {Name: "cli"},
		},
	}

	// Find component
	brick := ws.BrickByName("user")
	if brick == nil {
		t.Fatal("BrickByName(\"user\") returned nil")
	}
	if brick.IsBase() {
		t.Error("user should not be a base")
	}

	// Find base
	brick = ws.BrickByName("cli")
	if brick == nil {
		t.Fatal("BrickByName(\"cli\") returned nil")
	}
	if !brick.IsBase() {
		t.Error("cli should be a base")
	}

	// Not found
	brick = ws.BrickByName("nonexistent")
	if brick != nil {
		t.Errorf("BrickByName(\"nonexistent\") = %v, want nil", brick)
	}
}

func TestWorkspaceComponentNames(t *testing.T) {
	ws := &Workspace{
		Components: map[string]*Component{
			"user":  {Name: "user"},
			"admin": {Name: "admin"},
			"util":  {Name: "util"},
		},
	}
	names := ws.ComponentNames()
	want := []string{"admin", "user", "util"}
	if len(names) != len(want) {
		t.Fatalf("ComponentNames() = %v, want %v", names, want)
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("ComponentNames()[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestWorkspaceInterfaceNames(t *testing.T) {
	ws := &Workspace{
		Components: map[string]*Component{
			"user":  {Name: "user", Interface: "user"},
			"admin": {Name: "admin", Interface: "user"},
			"util":  {Name: "util", Interface: "util"},
		},
	}
	names := ws.InterfaceNames()
	want := []string{"user", "util"}
	if len(names) != len(want) {
		t.Fatalf("InterfaceNames() = %v, want %v", names, want)
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("InterfaceNames()[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestWorkspaceComponentsForInterface(t *testing.T) {
	ws := &Workspace{
		Components: map[string]*Component{
			"user":  {Name: "user", Interface: "user"},
			"admin": {Name: "admin", Interface: "user"},
			"util":  {Name: "util", Interface: "util"},
		},
	}
	comps := ws.ComponentsForInterface("user")
	if len(comps) != 2 {
		t.Fatalf("ComponentsForInterface(\"user\") returned %d components, want 2", len(comps))
	}
	// Sorted by name: admin, user
	if comps[0].Name != "admin" {
		t.Errorf("comps[0].Name = %q, want %q", comps[0].Name, "admin")
	}
	if comps[1].Name != "user" {
		t.Errorf("comps[1].Name = %q, want %q", comps[1].Name, "user")
	}
}

func TestWorkspaceToJSON(t *testing.T) {
	ws := &Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &Config{
			TopNamespace: "github.com/test",
		},
		Components: map[string]*Component{
			"user": {Name: "user", Module: "github.com/test/user-component", Interface: "user"},
		},
		Bases:    map[string]*Base{},
		Projects: map[string]*Project{},
	}

	data, err := ws.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify it's valid JSON and contains expected fields
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("ToJSON() produced invalid JSON: %v", err)
	}

	if decoded["name"] != "test" {
		t.Errorf("JSON name = %v, want %q", decoded["name"], "test")
	}
	if decoded["ws-dir"] != "/tmp/test" {
		t.Errorf("JSON ws-dir = %v, want %q", decoded["ws-dir"], "/tmp/test")
	}
}

func TestComponentBrickInterface(t *testing.T) {
	c := &Component{
		Name:      "user",
		Dir:       "components/user",
		Module:    "github.com/test/user-component",
		Interface: "user",
		Imports:   []string{"github.com/test/util-component/api"},
	}

	if c.BrickName() != "user" {
		t.Errorf("BrickName() = %q, want %q", c.BrickName(), "user")
	}
	if c.BrickPath() != "components/user" {
		t.Errorf("BrickPath() = %q, want %q", c.BrickPath(), "components/user")
	}
	if c.ModulePath() != "github.com/test/user-component" {
		t.Errorf("ModulePath() = %q", c.ModulePath())
	}
	if c.IsBase() {
		t.Error("Component.IsBase() = true, want false")
	}
	if c.BrickType() != "component" {
		t.Errorf("BrickType() = %q, want %q", c.BrickType(), "component")
	}
	ifaces := c.BrickInterfaces()
	if len(ifaces) != 1 || ifaces[0] != "user" {
		t.Errorf("BrickInterfaces() = %v, want [user]", ifaces)
	}
}

func TestBaseBrickInterface(t *testing.T) {
	b := &Base{
		Name:    "cli",
		Dir:     "bases/cli",
		Module:  "github.com/test/cli-base",
		Imports: []string{"github.com/test/user-component/api"},
	}

	if b.BrickName() != "cli" {
		t.Errorf("BrickName() = %q, want %q", b.BrickName(), "cli")
	}
	if !b.IsBase() {
		t.Error("Base.IsBase() = false, want true")
	}
	if b.BrickType() != "base" {
		t.Errorf("BrickType() = %q, want %q", b.BrickType(), "base")
	}
	if len(b.BrickInterfaces()) != 0 {
		t.Errorf("BrickInterfaces() = %v, want empty", b.BrickInterfaces())
	}
}

func TestImportIsComponentAPI(t *testing.T) {
	modulePath := "github.com/test/user-component"
	interfacePkg := "api"

	tests := []struct {
		name       string
		importPath string
		want       bool
	}{
		{"exact match", "github.com/test/user-component/api", true},
		{"subpackage", "github.com/test/user-component/api/sub", true},
		{"internal not api", "github.com/test/user-component/internal", false},
		{"different component", "github.com/test/other-component/api", false},
		{"unrelated", "github.com/spf13/cobra", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ImportIsComponentAPI(tt.importPath, modulePath, interfacePkg)
			if got != tt.want {
				t.Errorf("ImportIsComponentAPI(%q) = %v, want %v", tt.importPath, got, tt.want)
			}
		})
	}
}

func TestImportBelongsToComponent(t *testing.T) {
	modulePath := "github.com/test/user-component"

	tests := []struct {
		name       string
		importPath string
		want       bool
	}{
		{"exact match", "github.com/test/user-component", true},
		{"api package", "github.com/test/user-component/api", true},
		{"internal package", "github.com/test/user-component/internal", true},
		{"different component", "github.com/test/other-component", false},
		{"prefix but not child", "github.com/test/user-component-extra", false},
		{"unrelated", "github.com/spf13/cobra", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ImportBelongsToComponent(tt.importPath, modulePath)
			if got != tt.want {
				t.Errorf("ImportBelongsToComponent(%q) = %v, want %v", tt.importPath, got, tt.want)
			}
		})
	}
}

func TestWorkspaceModulePathForComponent(t *testing.T) {
	ws := &Workspace{
		Components: map[string]*Component{
			"user": {Name: "user", Module: "github.com/test/user-component"},
		},
	}
	got := ws.ModulePathForComponent("user")
	want := "github.com/test/user-component"
	if got != want {
		t.Errorf("ModulePathForComponent(\"user\") = %q, want %q", got, want)
	}

	got = ws.ModulePathForComponent("nonexistent")
	if got != "" {
		t.Errorf("ModulePathForComponent(\"nonexistent\") = %q, want \"\"", got)
	}
}

func TestWorkspaceBaseNames(t *testing.T) {
	ws := &Workspace{
		Bases: map[string]*Base{
			"cli":   {Name: "cli"},
			"admin": {Name: "admin"},
		},
	}
	names := ws.BaseNames()
	want := []string{"admin", "cli"}
	if len(names) != len(want) {
		t.Fatalf("BaseNames() = %v, want %v", names, want)
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("BaseNames()[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestWorkspaceProjectNames(t *testing.T) {
	ws := &Workspace{
		Projects: map[string]*Project{
			"myapp":       {Name: "myapp"},
			"development": {Name: "development"},
		},
	}
	names := ws.ProjectNames()
	want := []string{"development", "myapp"}
	if len(names) != len(want) {
		t.Fatalf("ProjectNames() = %v, want %v", names, want)
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("ProjectNames()[%d] = %q, want %q", i, n, want[i])
		}
	}
}
