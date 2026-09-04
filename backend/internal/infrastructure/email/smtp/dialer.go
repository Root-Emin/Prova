package smtp

import (
	"context"
	"net"
	"time"
)

// netDialer, bağlam ve zaman aşımı taşıyan TCP bağlantısı kurar.
//
// net/smtp.Dial bağlam almıyor; onu kullanmak, yavaş bir rölede isteğin
// süresiz takılması demek olurdu.
type netDialer struct {
	timeout time.Duration
}

func (d *netDialer) dial(ctx context.Context, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: d.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	// Okuma/yazma için de bir tavan konuyor: bağlantı kurulduktan sonra
	// sessizleşen bir sunucu, zaman aşımı olmadan goroutine'i sonsuza kadar
	// bekletir.
	if d.timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(d.timeout))
	}
	return conn, nil
}
