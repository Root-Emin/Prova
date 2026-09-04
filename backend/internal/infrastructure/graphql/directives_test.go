package graphql

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/shared/authctx"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeRBAC, sabit bir izin listesi döndürür.
type fakeRBAC struct {
	granted []string
	err     error
}

func (f fakeRBAC) HasPermission(_ context.Context, _, _ uuid.UUID, permission string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for _, p := range f.granted {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}
func (f fakeRBAC) HasAnyPermission(context.Context, uuid.UUID, uuid.UUID, []string) (bool, error) {
	return false, f.err
}
func (f fakeRBAC) GetUserPermissions(context.Context, uuid.UUID, uuid.UUID) ([]string, error) {
	return f.granted, f.err
}
func (f fakeRBAC) InvalidateCache(context.Context, uuid.UUID, uuid.UUID) error { return nil }

// fieldCtx, verilen nullability ile bir alan bağlamı kurar.
func fieldCtx(ctx context.Context, nonNull bool) context.Context {
	return graphql.WithFieldContext(ctx, &graphql.FieldContext{
		Field: graphql.CollectedField{
			Field: &ast.Field{
				Definition: &ast.FieldDefinition{
					Name: "override",
					Type: &ast.Type{NamedType: "ScoreOverride", NonNull: nonNull},
				},
			},
		},
	})
}

func authenticated(ctx context.Context) context.Context {
	return authctx.WithViewer(ctx, &authctx.Viewer{UserID: uuid.New(), OrgID: uuid.New()})
}

func nextReturns(value any) graphql.Resolver {
	return func(context.Context) (any, error) { return value, nil }
}

func TestAuthDirective_RejectsAnonymousCallers(t *testing.T) {
	d := &Directives{Log: testLogger()}

	_, err := d.Auth(context.Background(), nil, nextReturns("gizli"))

	var gqlErr *gqlerror.Error
	if !errors.As(err, &gqlErr) {
		t.Fatalf("GraphQL hatası bekleniyordu: %v", err)
	}
	if gqlErr.Extensions["code"] != CodeUnauthenticated {
		t.Fatalf("kod UNAUTHENTICATED olmalı: %v", gqlErr.Extensions)
	}
}

func TestAuthDirective_LetsAuthenticatedCallersThrough(t *testing.T) {
	d := &Directives{Log: testLogger()}

	got, err := d.Auth(authenticated(context.Background()), nil, nextReturns("veri"))

	if err != nil || got != "veri" {
		t.Fatalf("doğrulanmış çağıran geçmeliydi: %v %v", got, err)
	}
}

// Faz 1'in asıl kontrolü: yetkisiz kullanıcıda ezme alanı null gelir, sorgunun
// tamamı patlamaz. Çalışan puanını görebilmeli, ezilip ezilmediğini değil.
func TestPermissionDirective_ReturnsNullForNullableFieldWithoutPermission(t *testing.T) {
	d := &Directives{RBAC: fakeRBAC{granted: []string{"session:read"}}, Log: testLogger()}
	ctx := fieldCtx(authenticated(context.Background()), false)

	got, err := d.Permission(ctx, nil, nextReturns("ezme kaydı"), "score:override")

	if err != nil {
		t.Fatalf("nullable alanda hata değil null beklenir: %v", err)
	}
	if got != nil {
		t.Fatalf("yetkisiz kullanıcıda alan null olmalı, %v geldi", got)
	}
}

// Nullable olmayan alanda null döndürmek GraphQL'de geçersizdir; orada karar
// hataya çevrilmeli.
func TestPermissionDirective_ErrorsForNonNullableFieldWithoutPermission(t *testing.T) {
	d := &Directives{RBAC: fakeRBAC{}, Log: testLogger()}
	ctx := fieldCtx(authenticated(context.Background()), true)

	_, err := d.Permission(ctx, nil, nextReturns("veri"), "llm:write")

	var gqlErr *gqlerror.Error
	if !errors.As(err, &gqlErr) {
		t.Fatalf("GraphQL hatası bekleniyordu: %v", err)
	}
	if gqlErr.Extensions["code"] != CodeForbidden {
		t.Fatalf("kod FORBIDDEN olmalı: %v", gqlErr.Extensions)
	}
}

func TestPermissionDirective_AllowsWhenPermissionGranted(t *testing.T) {
	d := &Directives{RBAC: fakeRBAC{granted: []string{"score:override"}}, Log: testLogger()}
	ctx := fieldCtx(authenticated(context.Background()), false)

	got, err := d.Permission(ctx, nil, nextReturns("ezme kaydı"), "score:override")

	if err != nil || got != "ezme kaydı" {
		t.Fatalf("yetkili kullanıcıda alan dolu gelmeli: %v %v", got, err)
	}
}

// Joker izinler RBAC servisinde eşleşiyor; directive o kararı kendisi
// vermemeli, yoksa iki ayrı eşleştirme mantığı zamanla ayrışır.
func TestPermissionDirective_DelegatesWildcardMatchingToRBAC(t *testing.T) {
	d := &Directives{RBAC: fakeRBAC{granted: []string{"score:override"}}, Log: testLogger()}
	ctx := fieldCtx(authenticated(context.Background()), false)

	got, _ := d.Permission(ctx, nil, nextReturns("ok"), "score:read")

	if got != nil {
		t.Fatal("directive kendi başına joker eşleştirmesi yapmamalı")
	}
}

// RBAC bağlı değilse kapalı taraf seçilmeli: yetki servisi olmadan "izin var"
// varsaymak, izin sistemini hiç kurmamakla aynı şey.
func TestPermissionDirective_DeniesWhenRBACIsMissing(t *testing.T) {
	d := &Directives{Log: testLogger()}
	ctx := fieldCtx(authenticated(context.Background()), true)

	if _, err := d.Permission(ctx, nil, nextReturns("veri"), "llm:write"); err == nil {
		t.Fatal("RBAC yokken erişim reddedilmeli")
	}
}

// Alan bağlamı okunamıyorsa güvenli taraf hatadır: sessizce null döndürmek,
// yetki eksikliğini boş veriyle karıştırır.
func TestPermissionDirective_ErrorsWhenFieldContextIsUnavailable(t *testing.T) {
	d := &Directives{RBAC: fakeRBAC{}, Log: testLogger()}

	if _, err := d.Permission(authenticated(context.Background()), nil, nextReturns("veri"), "llm:write"); err == nil {
		t.Fatal("alan bağlamı yokken hata dönmeli")
	}
}

func TestPermissionDirective_SurfacesRBACFailures(t *testing.T) {
	d := &Directives{RBAC: fakeRBAC{err: errors.New("redis kapalı")}, Log: testLogger()}
	ctx := fieldCtx(authenticated(context.Background()), false)

	if _, err := d.Permission(ctx, nil, nextReturns("veri"), "score:override"); err == nil {
		t.Fatal("izin kontrolü başarısızsa sessizce null dönmemeli")
	}
}
