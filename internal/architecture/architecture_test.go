package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Enforce dependency direction: feature code cannot start another worker or
// couple the continuity module to legacy HTTP handlers and request globals.
func TestFeatureAndWorkerBoundaries(t *testing.T) {
	root := filepath.Join("..", "..", "internal")
	rules := map[string][]string{"businesscare": {"operations", "products", "workers", "db"}, "mercadopago": {"businesscare", "workers"}, "operations": {"workers"}, "auth": {"operations", "mercadopago", "businesscare", "workers"}}
	for pkg, forbidden := range rules {
		entries, e := os.ReadDir(filepath.Join(root, pkg))
		if e != nil {
			t.Fatal(e)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			file, e := parser.ParseFile(token.NewFileSet(), filepath.Join(root, pkg, entry.Name()), nil, 0)
			if e != nil {
				t.Fatal(e)
			}
			for _, item := range file.Imports {
				path, _ := strconv.Unquote(item.Path.Value)
				for _, dependency := range forbidden {
					if path == "leguiburger/internal/"+dependency {
						t.Errorf("%s imports forbidden %s", pkg, dependency)
					}
				}
			}
		}
	}
}
func TestContinuityRoutesAlwaysValidateLiveSessions(t *testing.T) {
	file, e := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "businesscare", "handler.go"), nil, 0)
	if e != nil {
		t.Fatal(e)
	}
	registrations := 0
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		member, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || member.Sel.Name != "HandleFunc" {
			return true
		}
		registrations++
		if len(call.Args) != 2 {
			t.Fatal("unexpected route registration")
		}
		wrapper, ok := call.Args[1].(*ast.CallExpr)
		if !ok {
			t.Error("route bypasses session middleware")
			return true
		}
		selector, ok := wrapper.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "AuthMiddlewareWithDB" {
			t.Error("route bypasses live session middleware")
		}
		return true
	})
	if registrations < 2 {
		t.Fatal("no continuity registrations checked")
	}
}
