//go:build tools

// Bu dosya derleme dışıdır; yalnızca kod üreticisini go.mod'da tutar.
// Aksi hâlde `go mod tidy` gqlgen'i kaldırır ve şema değiştiğinde üretim
// çalışmaz.
package tools

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/99designs/gqlgen/graphql/introspection"
)
