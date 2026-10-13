package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectComponentInterface(t *testing.T) {
	dir := t.TempDir()

	// Create a component with api/user.go containing UserService
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	apiFile := `package api

type UserService interface {
	Create(name string) error
}

type User struct {
	ID string
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "user.go"), []byte(apiFile), 0644); err != nil {
		t.Fatal(err)
	}

	iface := detectComponentInterface(dir, "api", "mycomp")
	if iface != "user" {
		t.Errorf("detectComponentInterface() = %q, want %q", iface, "user")
	}
}

func TestDetectComponentInterface_Fallback(t *testing.T) {
	dir := t.TempDir()

	// No api directory — should fall back to component name
	iface := detectComponentInterface(dir, "api", "mycomp")
	if iface != "mycomp" {
		t.Errorf("detectComponentInterface() = %q, want %q (fallback)", iface, "mycomp")
	}
}

func TestDetectComponentInterface_CustomIfaceName(t *testing.T) {
	dir := t.TempDir()

	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Interface with different name pattern
	apiFile := `package api

type PaymentService interface {
	Pay(amount int) error
}

type Payment struct {
	ID string
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "payment.go"), []byte(apiFile), 0644); err != nil {
		t.Fatal(err)
	}

	iface := detectComponentInterface(dir, "api", "billing")
	if iface != "payment" {
		t.Errorf("detectComponentInterface() = %q, want %q", iface, "payment")
	}
}

func TestExtractInterfaceFromFile(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		want     string
	}{
		{
			"UserService",
			"package api\n\ntype UserService interface {\n\tCreate() error\n}\n",
			"user",
		},
		{
			"UtilService",
			"package api\n\ntype UtilService interface {\n\tHelp() error\n}\n",
			"util",
		},
		{
			"no interface",
			"package api\n\ntype User struct {\n\tID string\n}\n",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(dir, "test.go")
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			got := extractInterfaceFromFile(filePath)
			if got != tt.want {
				t.Errorf("extractInterfaceFromFile() = %q, want %q", got, tt.want)
			}
		})
	}
}
