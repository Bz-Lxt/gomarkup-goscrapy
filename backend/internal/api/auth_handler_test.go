package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goscrapy/internal/auth"
	"goscrapy/internal/config"
	"goscrapy/internal/model"
)

func testRouter() *httptest.Server {
	d := &Deps{
		Cfg:  &config.Config{Role: "master"},
		Auth: auth.New("test-secret", time.Hour),
	}
	return httptest.NewServer(NewRouter(d))
}

func TestHealthAndLogin(t *testing.T) {
	srv := testRouter()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("health %d", resp.StatusCode)
	}
	var env model.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Code != 0 {
		t.Fatalf("code=%d", env.Code)
	}

	r, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/rules", nil)
	resp2, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 401 {
		t.Fatalf("expected 401 got %d", resp2.StatusCode)
	}

	body := bytes.NewBufferString(`{"username":"admin","password":"bad"}`)
	resp3, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != 401 {
		t.Fatalf("bad login %d", resp3.StatusCode)
	}

	body = bytes.NewBufferString(`{"username":"admin","password":"Admin@12345"}`)
	resp4, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != 200 {
		t.Fatalf("login %d", resp4.StatusCode)
	}
	var login model.Envelope
	if err := json.NewDecoder(resp4.Body).Decode(&login); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(login.Data)
	var ld model.LoginData
	_ = json.Unmarshal(data, &ld)
	if ld.Token == "" || ld.Username != "admin" {
		t.Fatalf("token missing %+v", ld)
	}
}
