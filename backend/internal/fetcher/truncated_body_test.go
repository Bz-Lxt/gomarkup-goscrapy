package fetcher

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestFetchReturnsTruncatedBodyError(t *testing.T) {
	client := New(nil, time.Second)
	client.base.DialContext = func(context.Context, string, string) (net.Conn, error) {
		server, peer := net.Pipe()
		go func() {
			defer peer.Close()
			reader := bufio.NewReader(peer)
			for {
				line, readErr := reader.ReadString('\n')
				if readErr != nil || line == "\r\n" {
					break
				}
			}
			_, _ = io.WriteString(peer, "HTTP/1.1 200 OK\r\nContent-Length: 1024\r\n\r\nincomplete response")
		}()
		return server, nil
	}

	result, err := client.Fetch(context.Background(), "http://truncated.test/page", false)
	if result != nil {
		t.Fatalf("Fetch returned a result for a truncated response: %+v", result)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Fetch error = %v, want io.ErrUnexpectedEOF in error chain", err)
	}
}
