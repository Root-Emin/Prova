package graphql

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/99designs/gqlgen/graphql"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	"github.com/masterfabric-go/masterfabric/internal/shared/authctx"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// Hata kodları. İstemci bunlara göre davranır: UNAUTHENTICATED yeniden giriş
// demektir, FORBIDDEN ise girişin sorunu olmadığını.
const (
	CodeUnauthenticated = "UNAUTHENTICATED"
	CodeForbidden       = "FORBIDDEN"
)

// Directives, şemadaki @auth ve @permission uygulamalarını taşır.
type Directives struct {
	RBAC service.RBACService
	Log  *slog.Logger
}

// Auth, geçerli bir oturum token'ı ister.
//
// Bu directive alan bazındadır çünkü kod isteme ve kod doğrulama mutation'ları
// token'sız çağrılmak zorundadır; HTTP seviyesindeki bir kapı ikisini
// ayıramaz.
func (d *Directives) Auth(ctx context.Context, _ any, next graphql.Resolver) (any, error) {
	if _, ok := authctx.ViewerFrom(ctx); !ok {
		return nil, &gqlerror.Error{
			Message:    "oturum açmanız gerekiyor",
			Extensions: map[string]any{"code": CodeUnauthenticated},
		}
	}
	return next(ctx)
}

// Permission, adı verilen izni ister.
//
// Yetki yoksa nullable alanlar için hata yerine null döner. Bunun nedeni
// pratiktir: bir çalışan kendi puanını görebilmeli, ama puanın yönetici
// tarafından ezilip ezilmediğini görmemeli. Ezme alanını hata yapmak, tüm
// skor sorgusunu — çalışanın görmeye hakkı olan kısmı dâhil — patlatırdı.
//
// Nullable olmayan alanlarda null döndürmek GraphQL'de geçersizdir; orada
// karar hataya çevrilir.
func (d *Directives) Permission(ctx context.Context, _ any, next graphql.Resolver, requires string) (any, error) {
	viewer, ok := authctx.ViewerFrom(ctx)
	if !ok {
		return d.denied(ctx, CodeUnauthenticated, "oturum açmanız gerekiyor")
	}
	if d.RBAC == nil {
		// RBAC bağlı değilse kapalı taraf seçilir. Yetki servisi olmadan
		// "izin var" varsaymak, izin sistemini hiç kurmamakla aynı şey.
		return d.denied(ctx, CodeForbidden, "yetki servisi kullanılamıyor")
	}

	allowed, err := d.RBAC.HasPermission(ctx, viewer.UserID, viewer.OrgID, requires)
	if err != nil {
		d.Log.ErrorContext(ctx, "izin kontrolü başarısız", "permission", requires, "error", err)
		return nil, &gqlerror.Error{
			Message:    "izin kontrolü yapılamadı",
			Extensions: map[string]any{"code": CodeForbidden},
		}
	}
	if !allowed {
		return d.denied(ctx, CodeForbidden, fmt.Sprintf("%s izni gerekiyor", requires))
	}
	return next(ctx)
}

// denied, alanın nullable olup olmamasına göre null ya da hata döndürür.
func (d *Directives) denied(ctx context.Context, code, message string) (any, error) {
	if fieldIsNullable(ctx) {
		return nil, nil
	}
	return nil, &gqlerror.Error{
		Message:    message,
		Extensions: map[string]any{"code": code},
	}
}

// fieldIsNullable, o an çözülen alanın şemadaki nullability'sini okur.
func fieldIsNullable(ctx context.Context) bool {
	fc := graphql.GetFieldContext(ctx)
	if fc == nil || fc.Field.Definition == nil || fc.Field.Definition.Type == nil {
		// Bağlam okunamıyorsa güvenli taraf hatadır: sessizce null döndürmek,
		// yetki eksikliğini boş veriyle karıştırır.
		return false
	}
	return !fc.Field.Definition.Type.NonNull
}
