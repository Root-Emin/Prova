package graphql

import (
	"context"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

// buildQuery, verilen derinlikte iç içe geçmiş bir seçim ağacı kurar.
func buildQuery(depth int) *ast.OperationDefinition {
	var build func(remaining int) ast.SelectionSet
	build = func(remaining int) ast.SelectionSet {
		if remaining <= 0 {
			return nil
		}
		return ast.SelectionSet{&ast.Field{Name: "child", SelectionSet: build(remaining - 1)}}
	}
	return &ast.OperationDefinition{SelectionSet: build(depth)}
}

func run(t *testing.T, limit int, op *ast.OperationDefinition) error {
	t.Helper()
	oc := &graphql.OperationContext{Operation: op, Doc: &ast.QueryDocument{}}
	limiter := DepthLimit{Max: limit}
	if err := limiter.MutateOperationContext(context.Background(), oc); err != nil {
		return err
	}
	return nil
}

func TestDepthLimit_AcceptsQueriesWithinLimit(t *testing.T) {
	if err := run(t, 5, buildQuery(5)); err != nil {
		t.Fatalf("limitteki sorgu reddedilmemeli: %v", err)
	}
}

func TestDepthLimit_RejectsQueriesBeyondLimit(t *testing.T) {
	err := run(t, 5, buildQuery(6))
	if err == nil {
		t.Fatal("limiti aşan sorgu reddedilmeliydi")
	}
	if !strings.Contains(err.Error(), "çok derin") {
		t.Fatalf("hata mesajı derinliği açıklamalı: %v", err)
	}
}

// Fragment içine gizlenmiş derinlik de sayılmalı: aksi hâlde limit, sorguyu
// iki parçaya bölerek aşılabilir.
func TestDepthLimit_CountsDepthInsideFragments(t *testing.T) {
	fragment := &ast.FragmentDefinition{
		Name:         "Deep",
		SelectionSet: buildQuery(6).SelectionSet,
	}
	op := &ast.OperationDefinition{
		SelectionSet: ast.SelectionSet{
			&ast.Field{
				Name: "root",
				SelectionSet: ast.SelectionSet{
					&ast.FragmentSpread{Name: "Deep", Definition: fragment},
				},
			},
		},
	}

	if err := run(t, 5, op); err == nil {
		t.Fatal("fragment içindeki derinlik de sayılmalı")
	}
}

// Inline fragment şemadaki nesneye kaç adım kaldığını değiştirmez; bir seviye
// saymak, tip daraltması kullanan her sorguyu haksız yere cezalandırırdı.
func TestDepthLimit_InlineFragmentIsNotALevel(t *testing.T) {
	op := &ast.OperationDefinition{
		SelectionSet: ast.SelectionSet{
			&ast.InlineFragment{SelectionSet: buildQuery(5).SelectionSet},
		},
	}
	if err := run(t, 5, op); err != nil {
		t.Fatalf("inline fragment ek seviye saymamalı: %v", err)
	}
}

// Introspection doğası gereği derindir (__schema → types → fields → type →
// ofType...). Limit ona uygulanırsa playground hiç çalışmaz; introspection'ın
// üretimde kapalı olması ayrı bir karardır.
func TestDepthLimit_SkipsIntrospectionQueries(t *testing.T) {
	op := &ast.OperationDefinition{
		SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "__schema", SelectionSet: buildQuery(20).SelectionSet},
		},
	}
	if err := run(t, 5, op); err != nil {
		t.Fatalf("introspection sorgusu derinlik limitine takılmamalı: %v", err)
	}
}

// Introspection ile veri alanlarını karıştıran bir sorgu muafiyeti almamalı,
// yoksa limit tek bir __typename eklenerek atlatılır.
func TestDepthLimit_MixedIntrospectionAndDataIsNotExempt(t *testing.T) {
	op := &ast.OperationDefinition{
		SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "__typename"},
			&ast.Field{Name: "session", SelectionSet: buildQuery(10).SelectionSet},
		},
	}
	if err := run(t, 5, op); err == nil {
		t.Fatal("__typename eklemek derinlik limitini atlatmamalı")
	}
}

func TestDepthLimit_ValidateRejectsNonsenseConfiguration(t *testing.T) {
	limiter := DepthLimit{Max: 0}
	if err := limiter.Validate(nil); err == nil {
		t.Fatal("sıfır derinlik limiti kabul edilmemeli")
	}
}
