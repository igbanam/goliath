package workspace

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// detectComponentInterface determines the interface name for a component
// by analyzing the api/ package to find the interface type name.
// Falls back to the component name if detection fails.
func detectComponentInterface(compDir, interfacePkg, compName string) string {
	apiDir := filepath.Join(compDir, interfacePkg)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return compName
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(apiDir, entry.Name())
		iface := extractInterfaceFromFile(filePath)
		if iface != "" {
			return iface
		}
	}

	return compName
}

// extractInterfaceFromFile extracts the interface name (e.g., "User" from "UserService")
func extractInterfaceFromFile(filePath string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return ""
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			// Check if it's an interface type
			if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				// Extract base name from "UserService" -> "user"
				name := typeSpec.Name.Name
				name = strings.TrimSuffix(name, "Service")
				if name != "" {
					return strings.ToLower(name)
				}
				return strings.ToLower(name)
			}
		}
	}

	return ""
}
