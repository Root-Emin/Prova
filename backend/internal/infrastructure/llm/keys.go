package llm

import (
	"os"
	"strings"
)

// EnvKeys, sağlayıcı API anahtarlarını ortam değişkenlerinden çözer.
//
// Anahtarlar LLM profilinde tutulmuyor. Profil, yönetici ekranından
// düzenlenebilen ve GraphQL üzerinden okunabilen bir belge; oraya bir sır
// koymak, sırrı yönetici arayüzünde ve veritabanı yedeğinde görünür kılardı.
type EnvKeys struct {
	// defaultKey, sağlayıcıya özel anahtar tanımlı değilse kullanılır.
	defaultKey string
	// byProvider, sağlayıcı adından anahtara. Farklı kademeler farklı
	// sağlayıcılarda olabilir (ucuz bir yerel model + güçlü bir bulut
	// modeli), ve o durumda tek anahtar yetmez.
	byProvider map[string]string
}

// NewEnvKeys reads LLM_API_KEY and LLM_API_KEYS from the environment.
//
// LLM_API_KEYS biçimi: "vllm=...,huggingface=hf_..."
func NewEnvKeys() *EnvKeys {
	keys := &EnvKeys{
		defaultKey: os.Getenv("LLM_API_KEY"),
		byProvider: map[string]string{},
	}
	for _, pair := range strings.Split(os.Getenv("LLM_API_KEYS"), ",") {
		name, value, found := strings.Cut(strings.TrimSpace(pair), "=")
		if !found {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			keys.byProvider[name] = strings.TrimSpace(value)
		}
	}
	return keys
}

// For implements service.APIKeys.
func (k *EnvKeys) For(provider string) string {
	if key, ok := k.byProvider[strings.ToLower(strings.TrimSpace(provider))]; ok && key != "" {
		return key
	}
	return k.defaultKey
}
