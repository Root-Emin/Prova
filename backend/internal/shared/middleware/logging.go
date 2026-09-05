package middleware

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	mflogger "github.com/masterfabric-go/masterfabric/internal/shared/logger"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
//
// Gömülü arayüz, sarmalayıcının ResponseWriter'ın tüm metotlarını devralmasını
// sağlar ama YALNIZCA http.ResponseWriter'ınkileri. Sunucunun sunduğu diğer
// arayüzler (Hijacker, Flusher) gömme ile taşınmaz ve aşağıda tek tek
// yeniden açılır.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack, WebSocket yükseltmesinin geçebilmesi için bağlantıyı devreder.
//
// Bu metot olmadan sarmalayıcı http.Hijacker'ı gizler ve her WebSocket
// el sıkışması "unable to upgrade" ile 501 döner — abonelikler sessizce
// çalışmaz hâle gelir, çünkü sorgu ve mutation yolları etkilenmez ve hata
// yalnızca abonelik denendiğinde görünür.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("altındaki ResponseWriter http.Hijacker uygulamıyor")
	}

	conn, buffered, err := hijacker.Hijack()
	if err != nil {
		return conn, buffered, err
	}

	// HTTP sunucusunun istek başına koyduğu okuma/yazma son tarihleri
	// devralınan bağlantıda yaşamaya devam ediyor. Devralınan bağlantı artık
	// bir istek/yanıt çifti değil, uzun ömürlü bir kanal: SERVER_WRITE_TIMEOUT
	// kadar sonra abonelik sessizce kopardı ve bunu yalnızca on beş saniyeden
	// uzun süren bir oturumda fark ederdik. Canlılık artık WebSocket'in kendi
	// ping/pong'una bırakılıyor.
	if conn != nil {
		_ = conn.SetDeadline(time.Time{})
	}
	return conn, buffered, nil
}

// Flush, akışlı yanıtların (server-sent events) tamponu boşaltmasını sağlar.
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap, http.ResponseController'ın altındaki yazıcıya ulaşmasını sağlar.
// Go 1.20+ bu metodu arıyor; olmadan okuma/yazma zaman aşımı ayarları
// sarmalayıcıda kaybolur.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// Logging logs every HTTP request with structured context.
func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			ctxLog := mflogger.WithContext(r.Context(), log)
			ctxLog.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}
