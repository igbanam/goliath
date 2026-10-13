package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/igbanam/goliath/internal/workspace"
)

// TestCmd runs tests
var TestCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests for bricks and/or projects",
	Long: `Run Go tests for components, bases, and/or projects.

By default, runs tests for all changed bricks.
Use flags to control which tests are run.

Examples:
  goliath test                    Run tests for all bricks
  goliath test --all              Run all brick and project tests
  goliath test --all-bricks       Run all brick tests
  goliath test --brick user       Run tests for the user component
  goliath test --brick user:admin Run tests for user and admin components
  goliath test --project myapp    Run tests for the myapp project
  goliath test --dev              Include development project tests`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		brickFilter, _ := cmd.Flags().GetString("brick")
		projectFilter, _ := cmd.Flags().GetString("project")
		all, _ := cmd.Flags().GetBool("all")
		allBricks, _ := cmd.Flags().GetBool("all-bricks")
		includeDev, _ := cmd.Flags().GetBool("dev")
		verbose, _ := cmd.Flags().GetBool("verbose")

		return runTests(ws, testOptions{
			brickFilter:   brickFilter,
			projectFilter: projectFilter,
			all:           all,
			allBricks:     allBricks,
			includeDev:    includeDev,
			verbose:       verbose,
		})
	},
}

type testOptions struct {
	brickFilter   string
	projectFilter string
	all           bool
	allBricks     bool
	includeDev    bool
	verbose       bool
}

func init() {
	TestCmd.Flags().String("brick", "", "Comma or colon-separated list of bricks to test (e.g., user:admin)")
	TestCmd.Flags().String("project", "", "Comma or colon-separated list of projects to test")
	TestCmd.Flags().Bool("all", false, "Run all brick tests and all project tests")
	TestCmd.Flags().Bool("all-bricks", false, "Run all brick tests")
	TestCmd.Flags().Bool("dev", false, "Include development project in test run")
	TestCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
}

// runTests executes tests based on the options
func runTests(ws *workspace.Workspace, opts testOptions) error {
	// Determine which directories to test
	var testDirs []struct {
		name string
		dir  string
		kind string // "component", "base", "project"
	}

	// Parse brick filter
	var brickNames []string
	if opts.brickFilter != "" {
		brickNames = strings.FieldsFunc(opts.brickFilter, func(r rune) bool {
			return r == ':' || r == ','
		})
	}

	// Parse project filter
	var projectNames []string
	if opts.projectFilter != "" {
		projectNames = strings.FieldsFunc(opts.projectFilter, func(r rune) bool {
			return r == ':' || r == ','
		})
	}

	// Collect test targets
	if len(brickNames) > 0 {
		// Test specific bricks
		for _, name := range brickNames {
			if comp, ok := ws.Components[name]; ok {
				testDirs = append(testDirs, struct {
					name string
					dir  string
					kind string
				}{comp.Name, comp.Dir, "component"})
			} else if base, ok := ws.Bases[name]; ok {
				testDirs = append(testDirs, struct {
					name string
					dir  string
					kind string
				}{base.Name, base.Dir, "base"})
			} else {
				return fmt.Errorf("brick %q not found", name)
			}
		}
	} else if len(projectNames) > 0 {
		// Test specific projects
		for _, name := range projectNames {
			proj, ok := ws.Projects[name]
			if !ok {
				// Try alias
				for _, p := range ws.Projects {
					if p.Alias == name {
						proj = p
						ok = true
						break
					}
				}
			}
			if !ok {
				return fmt.Errorf("project %q not found", name)
			}
			testDirs = append(testDirs, struct {
				name string
				dir  string
				kind string
			}{proj.Name, proj.Dir, "project"})
		}
	} else {
		// Default: test all bricks (and projects if --all)
		for _, name := range ws.ComponentNames() {
			comp := ws.Components[name]
			testDirs = append(testDirs, struct {
				name string
				dir  string
				kind string
			}{comp.Name, comp.Dir, "component"})
		}
		for _, name := range ws.BaseNames() {
			base := ws.Bases[name]
			testDirs = append(testDirs, struct {
				name string
				dir  string
				kind string
			}{base.Name, base.Dir, "base"})
		}

		if opts.all {
			for _, name := range ws.ProjectNames() {
				if name == "development" && !opts.includeDev {
					continue
				}
				proj := ws.Projects[name]
				testDirs = append(testDirs, struct {
					name string
					dir  string
					kind string
				}{proj.Name, proj.Dir, "project"})
			}
		}
	}

	if len(testDirs) == 0 {
		fmt.Println("No tests to run.")
		return nil
	}

	// Run tests
	failed := false
	passed := 0

	for _, target := range testDirs {
		fmt.Printf("Testing %s %q... ", target.kind, target.name)

		// Check if there are any test files
		hasTests := hasTestFiles(target.dir)
		if !hasTests {
			fmt.Println("no tests")
			continue
		}

		args := []string{"test", "./..."}
		if opts.verbose {
			args = append(args, "-v")
		}

		cmd := exec.Command("go", args...)
		cmd.Dir = target.dir
		// Disable go.work for running tests in individual modules
		// The go.work at workspace root causes issues when running from subdirectories
		cmd.Env = append(os.Environ(), "GOWORK=off")

		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("FAIL\n")
			if opts.verbose {
				fmt.Printf("  %s\n", strings.TrimSpace(string(output)))
			}
			failed = true
		} else {
			fmt.Printf("ok\n")
			passed++
			if opts.verbose {
				fmt.Printf("  %s\n", strings.TrimSpace(string(output)))
			}
		}
	}

	fmt.Printf("\n%d passed", passed)
	if failed {
		fmt.Printf(", some failed")
		fmt.Println()
		os.Exit(1)
	} else {
		fmt.Println()
	}

	return nil
}

// hasTestFiles checks if a directory contains any Go test files
func hasTestFiles(dir string) bool {
	found := false
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if found {
			return filepath.SkipAll
		}
		if !info.IsDir() && strings.HasSuffix(path, "_test.go") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
