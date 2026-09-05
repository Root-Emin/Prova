package middleware

import (
	"bufio"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// hijackableRecorder, http.Hijacker uygulayan bir test yazıcısı.
type hijackableRecorder struct {
	*httptest.ResponseRecorder
	conn *fakeConn
}

func (h *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h.conn == nil {
		h.conn = &fakeConn{}
	}
	return h.conn, bufio.NewReadWriter(bufio.NewReader(nil), bufio.NewWriter(io.Discard)), nil
}

// fakeConn, son tarih çağrılarını kaydeden bir net.Conn.
type fakeConn struct {
	net.Conn
	deadlines []time.Time
}

func (c *fakeConn) SetDeadline(t time.Time) error {
	c.deadlines = append(c.deadlines, t)
	return nil
}

// Sarmalayıcı http.Hijacker'ı gizlerse her WebSocket el sıkışması 501 döner
// ve abonelikler sessizce çalışmaz hâle gelir — sorgu ve mutation yolları
// etkilenmediği için hata yalnızca abonelik denendiğinde görünür.
func TestLogging_PreservesHijackerForWebSocketUpgrades(t *testing.T) {
	var canHijack bool

	handler := Logging(slog.New(slog.NewTextHandler(io.Discard, nil)))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, canHijack = w.(http.Hijacker)
		}))

	recorder := &hijackableRecorder{ResponseRecorder: httptest.NewRecorder()}
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/graphql", nil))

	if !canHijack {
		t.Fatal("sarmalayıcı http.Hijacker'ı geçirmeli, yoksa WebSocket yükseltmesi 501 döner")
	}
}

func TestLogging_PreservesFlusher(t *testing.T) {
	var canFlush bool

	handler := Logging(slog.New(slog.NewTextHandler(io.Discard, nil)))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, canFlush = w.(http.Flusher)
		}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !canFlush {
		t.Fatal("sarmalayıcı http.Flusher'ı geçirmeli, yoksa akışlı yanıtlar tamponda kalır")
	}
}

func TestLogging_StillCapturesStatusCode(t *testing.T) {
	handler := Logging(slog.New(slog.NewTextHandler(io.Discard, nil)))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusTeapot {
		t.Fatalf("durum kodu korunmalı: %d", recorder.Code)
	}
}

// Devralınan bağlantı artık bir istek/yanıt çifti değil, uzun ömürlü bir
// kanal. Sunucunun istek başına koyduğu yazma son tarihi kalırsa abonelik
// SERVER_WRITE_TIMEOUT kadar sonra sessizce kopar — ve bu, yalnızca o
// süreden uzun süren bir oturumda fark edilir.
func TestLogging_ClearsDeadlinesOnHijackedConnections(t *testing.T) {
	handler := Logging(slog.New(slog.NewTextHandler(io.Discard, nil)))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Error("Hijacker bekleniyordu")
				return
			}
			if _, _, err := hijacker.Hijack(); err != nil {
				t.Errorf("hijack başarısız: %v", err)
			}
		}))

	recorder := &hijackableRecorder{ResponseRecorder: httptest.NewRecorder()}
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/graphql", nil))

	if recorder.conn == nil || len(recorder.conn.deadlines) == 0 {
		t.Fatal("devralınan bağlantıda son tarih temizlenmeliydi")
	}
	if !recorder.conn.deadlines[len(recorder.conn.deadlines)-1].IsZero() {
		t.Fatalf("son tarih sıfırlanmalıydı: %v", recorder.conn.deadlines)
	}
}
