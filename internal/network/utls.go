package network

import (
	"context"
	"net"
	"net/http"
	"time"

	utls "github.com/refraction-networking/utls"
)

// uTLSDialer supplies an experimental browser-like TLS handshake.
// It uses uTLS in place of the standard TLS dialer.
// Compatibility with all servers and proxy combinations is not guaranteed.
type uTLSDialer struct {
	netDialer *net.Dialer
}

// newUTLSDialer creates a dialer using the Chrome handshake profile.
func newUTLSDialer(timeout time.Duration) *uTLSDialer {
	return &uTLSDialer{
		netDialer: &net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		},
	}
}

// DialContext opens a connection and performs a cancellable uTLS handshake.
// The selected profile is HelloChrome_Auto.
func (d *uTLSDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	plainConn, err := d.netDialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	// Configure the uTLS connection.
	cfg := &utls.Config{
		InsecureSkipVerify: false,
		MinVersion:         utls.VersionTLS12,
	}

	uconn := utls.UClient(plainConn, cfg, utls.HelloChrome_Auto)
	if err := uconn.HandshakeContext(ctx); err != nil {
		plainConn.Close()
		return nil, err
	}
	return uconn, nil
}

// EnableUTLS installs a uTLS dialer when enabled.
// Otherwise it leaves the transport unchanged.
func EnableUTLS(tr *http.Transport, useUTLS bool) {
	if !useUTLS {
		return
	}
	dialer := newUTLSDialer(5 * time.Second)
	tr.DialTLSContext = dialer.DialContext
}
