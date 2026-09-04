package graphql

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// DepthLimit, izin verilen en fazla seçim derinliğini zorlar.
//
// Karmaşıklık limiti tek başına yetmez: alan sayısı düşük ama derinliği yüksek
// bir sorgu (session { score { ... } } zincirini tekrarlayan) karmaşıklık
// bütçesini aşmadan çözücüleri özyinelemeye sokabilir. Derinlik, kesilmesi en
// ucuz olan boyuttur.
type DepthLimit struct {
	Max int
}

var _ interface {
	graphql.OperationContextMutator
	graphql.HandlerExtension
} = DepthLimit{}

// ExtensionName implements graphql.HandlerExtension.
func (DepthLimit) ExtensionName() string { return "DepthLimit" }

// Validate implements graphql.HandlerExtension.
func (d DepthLimit) Validate(graphql.ExecutableSchema) error {
	if d.Max < 1 {
		return fmt.Errorf("derinlik limiti en az 1 olmalı, %d verildi", d.Max)
	}
	return nil
}

// MutateOperationContext implements graphql.OperationContextMutator.
func (d DepthLimit) MutateOperationContext(_ context.Context, oc *graphql.OperationContext) *gqlerror.Error {
	if oc.Operation == nil {
		return nil
	}
	// Introspection sorguları doğaları gereği derindir (__schema → types →
	// fields → type → ofType...). Limit onlara uygulanmaz; introspection'ın
	// üretimde kapalı olması ayrı bir karardır.
	if isIntrospectionOnly(oc.Operation.SelectionSet) {
		return nil
	}
	if depth := selectionDepth(oc.Operation.SelectionSet, oc.Doc); depth > d.Max {
		return gqlerror.Errorf("sorgu çok derin: %d, izin verilen en fazla %d", depth, d.Max)
	}
	return nil
}

// selectionDepth, fragment'ları da izleyerek en derin dalı ölçer.
func selectionDepth(set ast.SelectionSet, doc *ast.QueryDocument) int {
	deepest := 0
	for _, selection := range set {
		var branch int
		switch s := selection.(type) {
		case *ast.Field:
			branch = 1 + selectionDepth(s.SelectionSet, doc)
		case *ast.InlineFragment:
			// Inline fragment kendi başına bir seviye değildir; şemadaki
			// nesneye kaç adım kaldığını değiştirmez.
			branch = selectionDepth(s.SelectionSet, doc)
		case *ast.FragmentSpread:
			if s.Definition != nil {
				branch = selectionDepth(s.Definition.SelectionSet, doc)
			}
		}
		if branch > deepest {
			deepest = branch
		}
	}
	return deepest
}

// isIntrospectionOnly, sorgunun yalnızca şema meta alanlarını istediğini
// bildirir.
func isIntrospectionOnly(set ast.SelectionSet) bool {
	if len(set) == 0 {
		return false
	}
	for _, selection := range set {
		field, ok := selection.(*ast.Field)
		if !ok {
			return false
		}
		if field.Name != "__schema" && field.Name != "__type" && field.Name != "__typename" {
			return false
		}
	}
	return true
}
