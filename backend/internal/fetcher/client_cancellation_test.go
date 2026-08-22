package fetcher

import (
	"bufio"
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestFetchPreservesCancellationWhileReadingBody(t *testing.T) {
	headersSent := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := New(nil, time.Second)
	client.base.DialContext = func(context.Context, string, string) (net.Conn, error) {
		clientConn, serverConn := net.Pipe()
		go func() {
			defer serverConn.Close()
			rd := bufio.NewReader(serverConn)
			for {
				line, err := rd.ReadString('\n')
				if err != nil {
					return
				}
				if line == "\r\n" {
					break
				}
			}
			_, _ = serverConn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\n"))
			close(headersSent)
			<-ctx.Done()
			_, _ = rd.ReadByte()
		}()
		return clientConn, nil
	}
	go func() {
		<-headersSent
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := client.Fetch(ctx, "http://example.test/page", false)
	if err == nil {
		t.Fatal("Fetch returned nil error after cancellation interrupted the response body")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch error = %v; want an error wrapping context.Canceled", err)
	}
}
