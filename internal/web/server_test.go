package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Atonolo/AtlasMail/internal/service"
	"github.com/Atonolo/AtlasMail/internal/store"
)

const testToken = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"

func TestWebServiceAuthenticatesAndPersistsAccount(t *testing.T) {
	handler := testHandler(t, Config{
		Service: service.New(store.New(t.TempDir())),
		Assets:  os.DirFS("../../src/ui"),
		Token:   testToken,
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status returned %d", response.StatusCode)
	}
	response.Body.Close()

	request, err = http.NewRequest(http.MethodPost, server.URL+"/api/v1/accounts",
		strings.NewReader(`{"label":"Test","address":"test@example.invalid","protocol":"jmap"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("account creation returned %d: %s", response.StatusCode, body)
	}
	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			Address string `json:"address"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Data.Address != "test@example.invalid" {
		t.Fatalf("unexpected API response: %#v", result)
	}
}

func TestDesktopModeAllowsOnlyItsLoopbackOrigin(t *testing.T) {
	handler := testHandler(t, Config{
		Service: service.New(store.New(t.TempDir())),
		Assets:  os.DirFS("../../src/ui"),
		Desktop: true,
	})
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:3123/api/v1/status", nil)
	request.Host = "127.0.0.1:3123"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("desktop status returned %d: %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "http://evil.example/api/v1/status", nil)
	request.Host = "evil.example"
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("invalid host returned %d", response.Code)
	}
}

func TestWebRejectsInvalidTokenAndPublicOrigin(t *testing.T) {
	base := Config{Service: service.New(store.New(t.TempDir())), Assets: os.DirFS("../../src/ui")}
	base.Token = "short"
	if _, err := NewHandler(base); err == nil {
		t.Fatal("short token must be rejected")
	}
	base.Token = testToken
	base.PublicOrigin = "http://example.invalid"
	if _, err := NewHandler(base); err == nil {
		t.Fatal("non-HTTPS public origin must be rejected")
	}
}

func TestEmbeddedUIIsServedWithSecurityHeaders(t *testing.T) {
	handler := testHandler(t, Config{
		Service: service.New(store.New(t.TempDir())),
		Assets:  os.DirFS("../../src/ui"),
		Desktop: true,
	})
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:3123/desktop", nil)
	request.Host = "127.0.0.1:3123"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Atlas Mail") {
		t.Fatalf("desktop UI did not load: %d", response.Code)
	}
	if response.Header().Get("Content-Security-Policy") == "" ||
		response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("UI response is missing security headers")
	}
}

func TestWebRejectsCrossOriginAndMalformedAccountRequests(t *testing.T) {
	handler := testHandler(t, Config{
		Service: service.New(store.New(t.TempDir())),
		Assets:  os.DirFS("../../src/ui"),
		Token:   testToken,
	})
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:3123/api/v1/status", nil)
	request.Host = "127.0.0.1:3123"
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-origin request returned %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "http://127.0.0.1:3123/api/v1/accounts",
		strings.NewReader(`{"label":"Test","address":"test@example.invalid","protocol":"jmap","secret":"no"}`))
	request.Host = "127.0.0.1:3123"
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "INVALID_JSON") {
		t.Fatalf("unexpected-field request returned %d: %s", response.Code, response.Body.String())
	}
}

func testHandler(t *testing.T, config Config) http.Handler {
	t.Helper()
	handler, err := NewHandler(config)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
