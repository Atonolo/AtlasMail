package web

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/Atonolo/AtlasMail/internal/credentials"
	"github.com/Atonolo/AtlasMail/internal/mail"
	"github.com/Atonolo/AtlasMail/internal/model"
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
		Token:   testToken,
	})
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:3123/api/v1/status", nil)
	request.Host = "127.0.0.1:3123"
	request.Header.Set("Authorization", "Bearer "+testToken)
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
		Token:   testToken,
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

type testCredentialBackend struct {
	mu     sync.Mutex
	values map[string]string
}

func (b *testCredentialBackend) Set(_, user, value string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.values == nil {
		b.values = make(map[string]string)
	}
	b.values[user] = value
	return nil
}

func (b *testCredentialBackend) Get(_, user string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.values[user], nil
}

func (b *testCredentialBackend) Delete(_, user string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.values, user)
	return nil
}

func TestAccountSetupStoresCredentialsAndRemovalRevokesThem(t *testing.T) {
	directory := t.TempDir()
	keyring := &testCredentialBackend{values: make(map[string]string)}
	mailService := service.NewWithCredentials(store.New(directory), credentials.NewWithBackend(keyring))
	handler := testHandler(t, Config{
		Service: mailService, Assets: os.DirFS("../../src/ui"), Token: testToken,
	})
	input := `{"account":{"label":"Personal","address":"person@example.org","protocol":"imap-smtp"},"profile":{"username":"person@example.org","authMechanism":"password","imap":{"host":"imap.example.org","port":993,"tlsMode":"implicit"},"smtp":{"host":"smtp.example.org","port":465,"tlsMode":"implicit"},"jmapUrl":"","oauth":null},"secret":"test-password-never-export"}`
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:3123/api/v1/accounts/setup", strings.NewReader(input))
	request.Host = "127.0.0.1:3123"
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("account setup returned %d: %s", response.Code, response.Body.String())
	}
	var created struct {
		Data model.Account `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(response.Body.String(), "test-password-never-export") {
		t.Fatal("account setup response exposed credential")
	}
	keyring.mu.Lock()
	if keyring.values[created.Data.ID] != "test-password-never-export" {
		keyring.mu.Unlock()
		t.Fatal("account setup did not save credential")
	}
	keyring.mu.Unlock()
	connections, err := store.New(directory).ReadConnections()
	if err != nil || strings.Contains(string(connections), "test-password-never-export") {
		t.Fatalf("credential appeared in connection settings: %q %v", connections, err)
	}
	backup, err := store.New(directory).ExportBackup()
	if err != nil || strings.Contains(string(backup), "test-password-never-export") {
		t.Fatalf("credential appeared in backup: %q %v", backup, err)
	}

	request = httptest.NewRequest(http.MethodDelete, "http://127.0.0.1:3123/api/v1/accounts/"+created.Data.ID, nil)
	request.Host = "127.0.0.1:3123"
	request.Header.Set("Authorization", "Bearer "+testToken)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("account removal returned %d: %s", response.Code, response.Body.String())
	}
	keyring.mu.Lock()
	remaining := keyring.values[created.Data.ID]
	keyring.mu.Unlock()
	if remaining != "" {
		t.Fatal("account removal left credentials behind")
	}
}

func TestOAuthStartUsesAuthorizedOriginAndCallbackDoesNotNeedBearer(t *testing.T) {
	mailService := service.NewWithCredentials(store.New(t.TempDir()), credentials.NewWithBackend(&testCredentialBackend{}))
	_, err := mailService.ConfigureAccount(model.AccountInput{
		Label: "OAuth", Address: "person@example.org", Protocol: "jmap",
	}, mail.Profile{
		Username: "person@example.org", AuthMechanism: "oauth2", JMAPURL: "https://mail.example.org/session",
		OAuth: &mail.OAuthSettings{
			AuthorizationURL: "https://provider.example/oauth/authorize",
			TokenURL:         "https://provider.example/oauth/token", ClientID: "public-client", Scope: "mail",
		},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	handler := testHandler(t, Config{
		Service: mailService, Assets: os.DirFS("../../src/ui"), Token: testToken,
		Desktop: true,
	})
	status, err := mailService.Status()
	if err != nil {
		t.Fatal(err)
	}
	accountID := status.Accounts[0].ID
	request := httptest.NewRequest(http.MethodGet,
		"http://127.0.0.1:3123/api/v1/oauth/start?accountId="+accountID, nil)
	request.Host = "127.0.0.1:3123"
	request.Header.Set("Authorization", "Bearer "+testToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "code_challenge_method") {
		t.Fatalf("OAuth start failed: %d %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet,
		"http://127.0.0.1:3123/oauth/callback?state=invalid-state-000000000000000000000000&code=sample", nil)
	request.Host = "127.0.0.1:3123"
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "oauth-callback.js") {
		t.Fatalf("OAuth callback was not safely rejected: %d %s", response.Code, response.Body.String())
	}
}

func testHandler(t *testing.T, config Config) http.Handler {
	t.Helper()
	handler, err := NewHandler(config)
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(http.LocalAddrContextKey) == nil {
			r = r.WithContext(context.WithValue(r.Context(), http.LocalAddrContextKey,
				&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3123}))
		}
		handler.ServeHTTP(w, r)
	})
}

func TestLocalSessionRequiresTokenAndSurvivesPageReload(t *testing.T) {
	handler := testHandler(t, Config{
		Service: service.New(store.New(t.TempDir())), Assets: os.DirFS("../../src/ui"),
		Desktop: true, Token: testToken,
	})
	send := func(path, method, token string, cookie *http.Cookie, localHeader bool) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, "http://127.0.0.1:3123"+path, nil)
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		if cookie != nil {
			request.AddCookie(cookie)
		}
		if localHeader {
			request.Header.Set("X-Atlas-Local", "1")
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder
	}
	for _, path := range []string{"/api/v1/session", "/api/v1/accounts"} {
		if response := send(path, http.MethodPost, "", nil, false); response.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated %s returned %d", path, response.Code)
		}
	}
	if response := send("/api/v1/status", http.MethodGet, "", nil, false); response.Code != http.StatusUnauthorized {
		t.Fatal("local status must not bypass authentication")
	}
	bootstrap := send("/api/v1/session", http.MethodPost, testToken, nil, false)
	if bootstrap.Code != http.StatusOK {
		t.Fatalf("bootstrap failed: %s", bootstrap.Body.String())
	}
	cookies := bootstrap.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode ||
		cookies[0].MaxAge != 0 || !cookies[0].Expires.IsZero() {
		t.Fatalf("expected a restricted session cookie: %#v", cookies)
	}
	if response := send("/api/v1/status", http.MethodGet, "", cookies[0], true); response.Code != http.StatusOK {
		t.Fatalf("reload with session cookie failed: %s", response.Body.String())
	}
	if response := send("/api/v1/status", http.MethodGet, "", cookies[0], false); response.Code != http.StatusUnauthorized {
		t.Fatal("cookie without custom local header must be rejected")
	}
}

func TestHostMustMatchListeningPort(t *testing.T) {
	handler := testHandler(t, Config{
		Service: service.New(store.New(t.TempDir())), Assets: os.DirFS("../../src/ui"), Token: testToken,
	})
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9000/api/v1/status", nil)
	request.Header.Set("Authorization", "Bearer "+testToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("wrong port was accepted: %d", response.Code)
	}
}
