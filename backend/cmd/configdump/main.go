// Command configdump, yapılandırmanın güvenlikle ilgili varsayılanlarını
// makine okunur biçimde basar.
//
// Doğrulama betiği bunu okuyor. Varsayılanları kaynak kodda aramak yerine
// yapılandırmayı gerçekten yükleyip sormak, "varsayılan kapalı" iddiasını
// kodun kendisine sordurur.
package main

import (
	"fmt"

	"github.com/masterfabric-go/masterfabric/internal/shared/config"
)

func main() {
	cfg := config.Load()

	fmt.Printf("environment=%s\n", cfg.Environment)
	fmt.Printf("store_raw_audio=%t\n", cfg.LLM.StoreRawAudio)
	fmt.Printf("kafka_enabled=%t\n", cfg.Kafka.Enabled)
	fmt.Printf("graphql_max_depth=%d\n", cfg.GraphQL.MaxDepth)
	fmt.Printf("graphql_max_complexity=%d\n", cfg.GraphQL.MaxComplexity)
	fmt.Printf("graphql_max_batch=%d\n", cfg.GraphQL.MaxBatch)
	fmt.Printf("access_ttl=%s\n", cfg.Token.AccessTTL)
	fmt.Printf("refresh_ttl=%s\n", cfg.Token.RefreshTTL)
	fmt.Printf("deletion_grace=%s\n", cfg.Lifecycle.DeletionGracePeriod)

	// Üretim sertleştirmesi: aynı yapılandırma production'da neye dönüşüyor.
	cfg.Environment = config.EnvProduction
	cfg.Harden()
	fmt.Printf("production_playground=%t\n", cfg.GraphQL.PlaygroundEnabled)
	fmt.Printf("production_introspection=%t\n", cfg.GraphQL.IntrospectionEnabled)
}
