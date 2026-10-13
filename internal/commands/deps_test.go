package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/igbanam/goliath/internal/workspace"
)

// captureStdout runs fn and returns whatever it wrote to stdout.
func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	_ = fn() // print-layout errors are irrelevant here

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// TestPrintWorkspaceDeps_VerticalHeaders asserts the workspace dependency matrix
// prints its column headers vertically (one rune of a brick name per line) so
// long names don't widen the table.
func TestPrintWorkspaceDeps_VerticalHeaders(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"alpha": {Name: "alpha", Module: "github.com/test/alpha-component", Interface: "alpha"},
			"beta":  {Name: "beta", Module: "github.com/test/beta-component", Interface: "beta"},
		},
		Bases:    map[string]*workspace.Base{},
		Projects: map[string]*workspace.Project{},
	}

	out := captureStdout(t, func() error { return printWorkspaceDeps(ws, false) })
	verticalHeaderCols(t, out)
}

// TestPrintProjectDeps_VerticalHeaders asserts the per-project matrix
// (`goliath deps --project`) uses the same vertical headers.
func TestPrintProjectDeps_VerticalHeaders(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"alpha": {Name: "alpha", Module: "github.com/test/alpha-component", Interface: "alpha"},
			"beta":  {Name: "beta", Module: "github.com/test/beta-component", Interface: "beta"},
		},
		Bases: map[string]*workspace.Base{
			"shower": {Name: "shower", Module: "github.com/test/shower-base"},
		},
		Projects: map[string]*workspace.Project{
			"myapp": {
				Name:     "myapp",
				Requires: []string{"github.com/test/alpha-component", "github.com/test/beta-component", "github.com/test/shower-base"},
			},
		},
	}

	out := captureStdout(t, func() error { return printProjectDeps(ws, "myapp") })
	verticalHeaderCols(t, out)
}

// verticalHeaderCols stacks the header block of a matrix output
// column-by-column and asserts each brick name appears as a vertical run.
func verticalHeaderCols(t *testing.T, out string) {
	t.Helper()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Find the separator row (dashes); everything above it is the header block.
	sep := -1
	for i, ln := range lines {
		if strings.Contains(ln, "---") {
			sep = i
			break
		}
	}
	if sep < 0 {
		t.Fatalf("expected a separator line of dashes in output:\n%s", out)
	}
	headerLines := lines[:sep]

	// Stack the header block column-by-column.
	maxLen := 0
	for _, ln := range headerLines {
		if n := len([]rune(ln)); n > maxLen {
			maxLen = n
		}
	}
	var cols []string
	for col := 0; col < maxLen; col++ {
		var sb strings.Builder
		for _, ln := range headerLines {
			rn := []rune(ln)
			if col < len(rn) {
				sb.WriteRune(rn[col])
			}
		}
		cols = append(cols, sb.String())
	}
	wantCols := "\n" + strings.Join(cols, "\n") + "\n"

	for _, want := range []string{"alpha", "beta"} {
		if !strings.Contains(wantCols, want) {
			t.Errorf("expected vertical header to contain column name %q.\ncolumns: %q\noutput:\n%s", want, cols, out)
		}
	}

	// Data rows must still be present, each prefixed by its full brick name.
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Errorf("expected output to contain data rows for alpha and beta.\noutput:\n%s", out)
	}
}

// TestPrintWorkspaceDeps_PreservesDependencyMarks asserts that switching to
// vertical headers did not change which cells are marked.
func TestPrintWorkspaceDeps_PreservesDependencyMarks(t *testing.T) {
	ws := &workspace.Workspace{
		Dir:  "/tmp/test",
		Name: "test",
		Config: &workspace.Config{
			TopNamespace: "github.com/test",
			InterfacePkg: "api",
		},
		Components: map[string]*workspace.Component{
			"alpha": {
				Name: "alpha", Module: "github.com/test/alpha-component", Interface: "alpha",
				Imports: []string{"github.com/test/beta-component/api"},
			},
			"beta": {Name: "beta", Module: "github.com/test/beta-component", Interface: "beta"},
		},
		Bases:    map[string]*workspace.Base{},
		Projects: map[string]*workspace.Project{},
	}

	out := captureStdout(t, func() error { return printWorkspaceDeps(ws, false) })

	// alpha uses beta, so the row starting with "alpha" must contain an 'x'.
	for _, ln := range strings.Split(out, "\n") {
		if strings.HasPrefix(ln, "alpha") {
			if !strings.Contains(ln, "x") {
				t.Errorf("expected alpha's row to be marked as using beta ('x'), got %q", ln)
			}
			return
		}
	}
	t.Fatalf("expected a data row starting with 'alpha'.\noutput:\n%s", out)
}
