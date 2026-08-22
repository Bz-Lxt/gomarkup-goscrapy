package fetcher_test

import (
	"errors"
	"fmt"
	"testing"

	"goscrapy/internal/fetcher"
)

func TestRetryableRecognizesWrappedTransportError(t *testing.T) {
	transportErr := errors.New("connection reset")
	err := fmt.Errorf("download page: %w", transportErr)
	if !fetcher.Retryable(0, err) {
		t.Fatal("a transport error must remain retryable when operation context is added")
	}
}
