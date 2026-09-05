// Command schema, GraphQL şemasını SDL olarak dışa aktarır.
//
// Next.js ve Electron istemcileri bu dosyadan tip üretiyor. Şema kaynağı
// graph/schema.graphqls ama dışa aktarım gqlgen'in ayrıştırdığı hâli yazıyor:
// böylece dosyanın gerçekten geçerli bir şema olduğu ve sunucunun
// çalıştırdığı şemayla aynı olduğu üretim anında kanıtlanmış oluyor.
//
// Çalıştırma:
//
//	go run ./cmd/schema -out ../schema.graphql
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/masterfabric-go/masterfabric/graph"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
)

func main() {
	out := flag.String("out", "schema.graphql", "yazılacak SDL dosyası")
	flag.Parse()

	schema := graph.NewExecutableSchema(graph.Config{}).Schema()

	var b strings.Builder
	b.WriteString("# Prova GraphQL şeması (üretilmiş dosya).\n")
	b.WriteString("#\n")
	b.WriteString("# Elle düzenlemeyin. Kaynak: backend/graph/schema.graphqls\n")
	b.WriteString("# Yeniden üretmek için: cd backend && go run ./cmd/schema -out ../schema.graphql\n\n")

	formatter.NewFormatter(&b).FormatSchema(sortedSchema(schema))

	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "şema yazılamadı: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("şema yazıldı: %s (%d bayt)\n", *out, b.Len())
}

// sortedSchema, tip sırasını belirlenimci yapar.
//
// gqlparser tipleri map'te tutuyor ve map iterasyonu Go'da rastgele. Sıralama
// olmadan her üretim farklı bir dosya yazar ve fark (diff) okunamaz hâle
// gelir — üretilmiş bir dosyanın versiyon kontrolünde işe yaraması için
// çıktının kararlı olması şart.
func sortedSchema(schema *ast.Schema) *ast.Schema {
	names := make([]string, 0, len(schema.Types))
	for name := range schema.Types {
		names = append(names, name)
	}
	sort.Strings(names)

	ordered := make(map[string]*ast.Definition, len(schema.Types))
	for _, name := range names {
		ordered[name] = schema.Types[name]
	}
	schema.Types = ordered
	return schema
}
