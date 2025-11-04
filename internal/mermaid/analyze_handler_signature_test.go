package mermaid

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestHandlerEventTypeFromSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "EventSuffixParameter",
			src: `package handler

import "context"

type SomeEvent struct{}

type WorkflowStateMachine struct{}

var handler = func(ctx context.Context, event *SomeEvent, sm *WorkflowStateMachine) {}
`,
			want: "SomeEvent",
		},
		{
			name: "FallbackSkipsStateMachine",
			src: `package handler

import "context"

type Notification struct{}

type WorkflowStateMachine struct{}

var handler = func(ctx context.Context, n *Notification, sm *WorkflowStateMachine) {}
`,
			want: "Notification",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "handler.go", tt.src, 0)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Defs:  make(map[*ast.Ident]types.Object),
				Uses:  make(map[*ast.Ident]types.Object),
			}
			conf := &types.Config{Importer: importer.Default()}
			if _, err := conf.Check("handler", fset, []*ast.File{file}, info); err != nil {
				t.Fatalf("failed to type-check source: %v", err)
			}

			var fn *ast.FuncLit
			ast.Inspect(file, func(n ast.Node) bool {
				if lit, ok := n.(*ast.FuncLit); ok {
					fn = lit
					return false
				}
				return true
			})

			if fn == nil {
				t.Fatalf("failed to locate function literal in test source")
			}

			got := handlerEventTypeFromSignature(fn, info)
			if got != tt.want {
				t.Fatalf("handlerEventTypeFromSignature() = %q, want %q", got, tt.want)
			}
		})
	}
}
