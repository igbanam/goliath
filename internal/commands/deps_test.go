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
	verticalHeaderCols(t, out, []string{"alpha", "beta"})
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
	verticalHeaderCols(t, out, []string{"alpha", "beta", "shower"})
}

// verticalHeaderCols asserts the header block of a matrix output prints each
// column name vertically (one rune per line), aligned to the BOTTOM of the
// header block (extra space on top for shorter names), one rune wide per
// column with a single separating space.
func verticalHeaderCols(t *testing.T, out string, colNames []string) {
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

	// The block must be exactly as tall as the longest column name.
	height := 0
	for _, n := range colNames {
		if len(n) > height {
			height = len(n)
		}
	}
	if got := len(headerLines); got != height {
		t.Fatalf("header block height = %d, want %d (one line per rune of the longest name %q).\noutput:\n%s", got, height, colNames, out)
	}

	// The label gutter sits on the BOTTOM header line.
	if got := strings.TrimRight(headerLines[len(headerLines)-1], " "); !strings.HasPrefix(got, "brick") {
		t.Errorf("expected the bottom header line to start with the 'brick' label, got %q.\noutput:\n%s", headerLines[len(headerLines)-1], out)
	}

	// Stack the whole header block column-by-column so a column's letters can
	// be read as one string.
	maxLen := 0
	for _, ln := range headerLines {
		if n := len([]rune(ln)); n > maxLen {
			maxLen = n
		}
	}
	var cols []string
	for col := 0; col < maxLen; col++ {
		cols = append(cols, stackRunes(headerLines, col))
	}
	wantCols := "\n" + strings.Join(cols, "\n") + "\n"

	// Each name must appear bottom-aligned: a shorter name is space-padded on
	// top so its last rune sits on the bottom header line.
	for _, name := range colNames {
		padded := strings.Repeat(" ", height-len(name)) + name
		if !strings.Contains(wantCols, padded) {
			t.Errorf("expected vertical header to contain %q bottom-aligned.\ncolumns: %q\noutput:\n%s", name, cols, out)
		}
	}

	// Column pitch must be exactly 2 runes (1 rune of name + 1 separating
	// space). The gutter is the longest name + 1; columns start right after, 2
	// runes apart. Stacking the gutter position of each column pins both the
	// pitch and the alignment.
	gutter := height + 1
	for j, name := range colNames {
		got := stackRunes(headerLines, gutter+j*2)
		want := strings.Repeat(" ", height-len(name)) + name
		if got != want {
			t.Errorf("column %d (%s) stacked = %q, want %q (bottom-aligned, pitch 2)", j, name, got, want)
		}
	}

	// Data rows must still be present, each prefixed by its full brick name.
	for _, name := range colNames {
		if !strings.HasPrefix(strings.TrimRight(out, "\n"), name) && !strings.Contains(out, "\n"+name) {
			t.Errorf("expected output to contain a data row for %q.\noutput:\n%s", name, out)
		}
	}
}

// stackRunes returns the runes at the given column index across all lines.
func stackRunes(lines []string, col int) string {
	var sb strings.Builder
	for _, ln := range lines {
		rn := []rune(ln)
		if col < len(rn) {
			sb.WriteRune(rn[col])
		}
	}
	return sb.String()
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
