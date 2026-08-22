package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"goscrapy/internal/api"
	"goscrapy/internal/auth"
	"goscrapy/internal/model"
	"goscrapy/internal/renderer"
)

func TestCreateSnapshotWithoutFallbackFetcher(t *testing.T) {
	previousErrorWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = io.Discard
	defer func() { gin.DefaultErrorWriter = previousErrorWriter }()

	authSvc := auth.New("snapshot-test-secret", time.Hour)
	token, _, err := authSvc.Authenticate(auth.DefaultUser, auth.DefaultPassword)
	if err != nil {
		t.Fatal(err)
	}
	router := api.NewRouter(&api.Deps{
		Auth: authSvc,
		Snap: renderer.NewService("", nil),
	})

	req, err := http.NewRequest(http.MethodPost, "/api/v1/snapshots",
		bytes.NewBufferString(`{"url":"http://target.invalid/page"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
	var envelope model.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != model.CodeUnavailable {
		t.Fatalf("code=%d, want %d", envelope.Code, model.CodeUnavailable)
	}
}
