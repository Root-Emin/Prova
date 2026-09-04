// Package service, Prova'nın domain servislerini barındırır: prompt derleme,
// alıntı doğrulama, kademe yönlendirme ve LLM sağlayıcı portu.
package service

import (
	"context"
	"time"
)

// Role, sohbet mesajının kimden geldiği.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message, sağlayıcıya gönderilen tek mesaj.
type Message struct {
	Role    Role
	Content string
}

// Target, çağrının hangi sağlayıcıya ve modele gideceği.
//
// Model adı ve base URL koda gömülmez; ikisi de LLM profilinden gelir ve
// yönetici ekranından çalışma anında değiştirilebilir. APIKey ise profilden
// gelmez, env'den okunur: bir sır asla veritabanına yazılmaz, çünkü
// veritabanı yedeği sırların yedeği hâline gelir.
type Target struct {
	BaseURL string
	APIKey  string
	Model   string
}

// CompletionRequest, tek bir tamamlama isteği.
type CompletionRequest struct {
	Target
	Messages    []Message
	Temperature float64
	TopP        float64
	MaxTokens   int
	// JSONMode, modelden geçerli JSON döndürmesini ister. Hem karakter yanıtı
	// hem de puanlama yapılandırılmış veri döndürüyor; serbest metni
	// ayrıştırmaya çalışmak, modelin biçim değiştirdiği ilk gün kırılır.
	JSONMode bool
}

// Usage, çağrının token tüketimi. Maliyet buradan hesaplanır.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// CompletionResponse, sağlayıcının yanıtı.
type CompletionResponse struct {
	Content string
	Usage   Usage
	Model   string
	Latency time.Duration
}

// Provider, OpenAI-uyumlu bir sohbet tamamlama sağlayıcısı.
//
// Tek bir arayüz yeterli: sağlayıcı değiştirmek base URL ve model adını
// değiştirmekten ibaret olmalı. Sağlayıcıya özel bir arayüz, o sağlayıcıyı
// koda gömmenin dolaylı yolu olurdu.
type Provider interface {
	// Complete, tamamlamayı bekleyip tek parça döndürür.
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

	// Stream, üretilen parçaları geldikçe onDelta'ya verir ve sonunda tam
	// yanıtı döndürür. onDelta hata döndürürse akış kesilir.
	Stream(ctx context.Context, req CompletionRequest, onDelta func(string) error) (*CompletionResponse, error)

	// Health, sağlayıcının erişilebilir olduğunu doğrular. Devre kesici bunu
	// yarı açık durumda tek deneme için kullanır.
	Health(ctx context.Context, target Target) error
}
