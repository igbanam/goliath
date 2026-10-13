package commands

import (
	"testing"

	"github.com/igbanam/goliath/internal/workspace"
)

func TestCheckWorkspace_Clean(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"user": {
				Name:      "user",
				Module:    "github.com/test/user-component",
				Interface: "user",
				Imports:   []string{"fmt"},
			},
		},
		Bases: map[string]*workspace.Base{
			"cli": {
				Name:    "cli",
				Module:  "github.com/test/cli-base",
				Imports: []string{"github.com/test/user-component/api"},
			},
		},
		Projects: map[string]*workspace.Project{
			"development": {Name: "development", Alias: "dev"},
		},
	}

	result := CheckWorkspace(ws)
	if !result.OK() {
		t.Errorf("CheckWorkspace() should be OK for clean workspace, got findings: %v", result.Findings)
	}
}

func TestCheckWorkspace_NameCollision(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"user": {
				Name:      "user",
				Module:    "github.com/test/user-component",
				Interface: "user",
			},
		},
		Bases: map[string]*workspace.Base{
			"user": {
				Name:   "user",
				Module: "github.com/test/user-base",
			},
		},
		Projects: map[string]*workspace.Project{},
	}

	result := CheckWorkspace(ws)
	if result.OK() {
		t.Error("CheckWorkspace() should find name collision between base and component")
	}

	found105 := false
	for _, f := range result.Findings {
		if f.Code == 105 {
			found105 = true
		}
	}
	if !found105 {
		t.Error("CheckWorkspace() should report error 105 for name collision")
	}
}

func TestCheckWorkspace_IllegalDependency(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"user": {
				Name:      "user",
				Module:    "github.com/test/user-component",
				Interface: "user",
			},
			"admin": {
				Name:      "admin",
				Module:    "github.com/test/admin-component",
				Interface: "user",
				Imports:   []string{"github.com/test/user-component/internal"}, // ILLEGAL!
			},
		},
		Bases:    map[string]*workspace.Base{},
		Projects: map[string]*workspace.Project{},
	}

	result := CheckWorkspace(ws)

	found101 := false
	for _, f := range result.Findings {
		if f.Code == 101 {
			found101 = true
		}
	}
	if !found101 {
		t.Errorf("CheckWorkspace() should report error 101 for illegal dependency, got: %v", result.Findings)
	}
}

func TestCheckWorkspace_LegalAPIDependency(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"user": {
				Name:      "user",
				Module:    "github.com/test/user-component",
				Interface: "user",
			},
			"admin": {
				Name:      "admin",
				Module:    "github.com/test/admin-component",
				Interface: "user",
				Imports:   []string{"github.com/test/user-component/api"}, // LEGAL
			},
		},
		Bases:    map[string]*workspace.Base{},
		Projects: map[string]*workspace.Project{},
	}

	result := CheckWorkspace(ws)

	for _, f := range result.Findings {
		if f.Code == 101 {
			t.Errorf("CheckWorkspace() should NOT report error 101 for legal API dependency, got: %v", f)
		}
	}
}

func TestCheckWorkspace_CircularDependency(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"a": {
				Name:      "a",
				Module:    "github.com/test/a-component",
				Interface: "a",
				Imports:   []string{"github.com/test/b-component/api"},
			},
			"b": {
				Name:      "b",
				Module:    "github.com/test/b-component",
				Interface: "b",
				Imports:   []string{"github.com/test/a-component/api"},
			},
		},
		Bases:    map[string]*workspace.Base{},
		Projects: map[string]*workspace.Project{},
	}

	result := CheckWorkspace(ws)

	found104 := false
	for _, f := range result.Findings {
		if f.Code == 104 {
			found104 = true
		}
	}
	if !found104 {
		t.Errorf("CheckWorkspace() should report error 104 for circular dependency, got: %v", result.Findings)
	}
}

func TestCheckWorkspace_NoCircularDependency(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"a": {
				Name:      "a",
				Module:    "github.com/test/a-component",
				Interface: "a",
				Imports:   []string{"github.com/test/b-component/api"},
			},
			"b": {
				Name:      "b",
				Module:    "github.com/test/b-component",
				Interface: "b",
				Imports:   []string{"github.com/test/c-component/api"},
			},
			"c": {
				Name:      "c",
				Module:    "github.com/test/c-component",
				Interface: "c",
			},
		},
		Bases:    map[string]*workspace.Base{},
		Projects: map[string]*workspace.Project{},
	}

	result := CheckWorkspace(ws)

	for _, f := range result.Findings {
		if f.Code == 104 {
			t.Errorf("CheckWorkspace() should NOT report error 104 for acyclic deps, got: %v", f)
		}
	}
}

func TestCheckWorkspace_DuplicateInterfaceInProject(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"user": {
				Name:      "user",
				Module:    "github.com/test/user-component",
				Interface: "user",
			},
			"admin": {
				Name:      "admin",
				Module:    "github.com/test/admin-component",
				Interface: "user", // Same interface!
			},
		},
		Bases: map[string]*workspace.Base{},
		Projects: map[string]*workspace.Project{
			"myapp": {
				Name:     "myapp",
				Requires: []string{"github.com/test/user-component", "github.com/test/admin-component"},
			},
		},
	}

	result := CheckWorkspace(ws)

	found106 := false
	for _, f := range result.Findings {
		if f.Code == 106 {
			found106 = true
		}
	}
	if !found106 {
		t.Errorf("CheckWorkspace() should report error 106 for duplicate interface in project, got: %v", result.Findings)
	}
}

func TestCheckWorkspace_DuplicateInterfaceInDevelopmentOK(t *testing.T) {
	// Development project can have multiple implementations of the same interface
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"user": {
				Name:      "user",
				Module:    "github.com/test/user-component",
				Interface: "user",
			},
			"admin": {
				Name:      "admin",
				Module:    "github.com/test/admin-component",
				Interface: "user",
			},
		},
		Bases: map[string]*workspace.Base{},
		Projects: map[string]*workspace.Project{
			"development": {
				Name:     "development",
				Alias:    "dev",
				Requires: []string{"github.com/test/user-component", "github.com/test/admin-component"},
			},
		},
	}

	result := CheckWorkspace(ws)

	for _, f := range result.Findings {
		if f.Code == 106 {
			t.Errorf("CheckWorkspace() should NOT report error 106 for development project, got: %v", f)
		}
	}
}

func TestCheckResult_HasErrors(t *testing.T) {
	r := &CheckResult{}
	if r.HasErrors() {
		t.Error("Empty result should not have errors")
	}

	r.Findings = append(r.Findings, Finding{Code: 201, Severity: SeverityWarning})
	if r.HasErrors() {
		t.Error("Warning-only result should not have errors")
	}

	r.Findings = append(r.Findings, Finding{Code: 101, Severity: SeverityError})
	if !r.HasErrors() {
		t.Error("Result with error should have errors")
	}
}
