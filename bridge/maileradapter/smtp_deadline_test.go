package maileradapter

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/bridge"
)

// A mail server that accepts the connection and then says nothing does not
// hold the caller past its deadline, on either the plain or the TLS path.
func TestSMTPMailer_HonoursDeadline(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		for {
			c, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			defer c.Close() // hold it open, never greet
		}
	}()
	host, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)

	for _, useTLS := range []bool{false, true} {
		m := NewSMTPMailer(host, port, "", "", "noreply@example.test", WithSMTPTLS(useTLS))
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		start := time.Now()
		sendErr := m.SendEmail(ctx, &bridge.EmailMessage{To: []string{"a@example.test"}, Subject: "hi", Text: "hello"})
		cancel()
		assert.Error(t, sendErr, "tls=%v", useTLS)
		assert.Less(t, time.Since(start), 3*time.Second, "tls=%v: the deadline, not the server, ends the call", useTLS)
	}
}
