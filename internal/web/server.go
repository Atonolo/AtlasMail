package web

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Atonolo/AtlasMail/internal/mail"
	"github.com/Atonolo/AtlasMail/internal/model"
	"github.com/Atonolo/AtlasMail/internal/service"
	"github.com/Atonolo/AtlasMail/internal/store"
)

const (
	maxRequestBytes     = 16 * 1024
	maxMailRequestBytes = 16 * 1024 * 1024
	contentSecurity     = "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'"
)

type Config struct {
	Service      *service.MailService
	Assets       fs.FS
	Token        string
	PublicOrigin string
	Desktop      bool
}

func NewHandler(config Config) (http.Handler, error) {
	if config.Service == nil || config.Assets == nil {
		return nil, errors.New("service and embedded UI assets are required")
	}
	if !validToken(config.Token) {
		return nil, errors.New("a random base64url access token of 43-128 characters is required")
	}
	if config.Desktop && config.PublicOrigin != "" {
		return nil, errors.New("local application sessions cannot have a public origin")
	}
	if config.PublicOrigin != "" {
		parsed, err := url.Parse(config.PublicOrigin)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" ||
			parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") ||
			parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, errors.New("ATLAS_MAIL_PUBLIC_ORIGIN must be an HTTPS origin")
		}
		config.PublicOrigin = parsed.Scheme + "://" + parsed.Host
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, config)
	}), nil
}

func validToken(token string) bool {
	if len(token) < 43 || len(token) > 128 {
		return false
	}
	for _, char := range token {
		if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func handle(w http.ResponseWriter, r *http.Request, config Config) {
	origin, allowed := requestOrigin(r, config.PublicOrigin)
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "code": "ORIGIN_REJECTED"})
		return
	}
	if r.Header.Get("Origin") != "" && r.Header.Get("Origin") != origin {
		writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "code": "ORIGIN_REJECTED"})
		return
	}

	if r.Method == http.MethodGet {
		if asset, contentType, ok := staticAsset(r.URL.Path, config.Desktop); ok {
			content, err := fs.ReadFile(config.Assets, asset)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "code": "ASSET_READ_FAILED"})
				return
			}
			writeResponse(w, http.StatusOK, contentType, content)
			return
		}
		if r.URL.Path == "/oauth/callback" {
			handleOAuthCallback(w, r, config, origin)
			return
		}
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "code": "NOT_FOUND"})
		return
	}
	bearerAuthenticated := authenticate(r.Header.Get("Authorization"), config.Token)
	if config.Desktop && r.Method == http.MethodPost && r.URL.Path == "/api/v1/session" {
		if !bearerAuthenticated {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "code": "AUTH_REQUIRED"})
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name: sessionCookieName(r), Value: config.Token, Path: "/api/v1/",
			HttpOnly: true, SameSite: http.SameSiteStrictMode,
		})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": nil})
		return
	}
	sessionAuthenticated := false
	if config.Desktop && r.Header.Get("X-Atlas-Local") == "1" {
		if cookie, err := r.Cookie(sessionCookieName(r)); err == nil {
			sessionAuthenticated = authenticate("Bearer "+cookie.Value, config.Token)
		}
	}
	if !bearerAuthenticated && !sessionAuthenticated {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "code": "AUTH_REQUIRED"})
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/status":
		status, err := config.Service.Status()
		if err != nil {
			writeFailure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": status})
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/accounts":
		input, err := readAccount(r)
		if err != nil {
			var requestErr *requestError
			if errors.As(err, &requestErr) {
				writeJSON(w, requestErr.status, map[string]any{"ok": false, "code": requestErr.code})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "code": "INVALID_DATA"})
			return
		}
		account, err := config.Service.AddAccount(input)
		if err != nil {
			writeFailure(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": account})
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/accounts/setup":
		var input accountSetupRequest
		if err := decodeRequest(w, r, maxRequestBytes, &input); err != nil {
			writeRequestFailure(w, err)
			return
		}
		account, err := config.Service.ConfigureAccount(input.Account, input.Profile, input.Secret)
		if err != nil {
			writeFailure(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": account})
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/accounts/"):
		accountID := strings.TrimPrefix(r.URL.Path, "/api/v1/accounts/")
		if strings.Contains(accountID, "/") {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "code": "NOT_FOUND"})
			return
		}
		if err := config.Service.RemoveAccount(accountID); err != nil {
			writeFailure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": nil})
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/mail":
		accountID := r.URL.Query().Get("accountId")
		ctx, cancel := contextWithRequestTimeout(r)
		defer cancel()
		messages, err := config.Service.Receive(ctx, accountID)
		if err != nil {
			writeFailure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": messages})
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/mail/send":
		var input mail.SendInput
		if err := decodeRequest(w, r, maxMailRequestBytes, &input); err != nil {
			writeRequestFailure(w, err)
			return
		}
		ctx, cancel := contextWithRequestTimeout(r)
		defer cancel()
		if err := config.Service.Send(ctx, input); err != nil {
			writeFailure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": nil})
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/oauth/start":
		authorizationURL, err := config.Service.BeginOAuth(r.URL.Query().Get("accountId"), origin+"/oauth/callback")
		if err != nil {
			writeFailure(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "data": map[string]string{"authorizationUrl": authorizationURL},
		})
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "code": "NOT_FOUND"})
	}
}

func requestOrigin(r *http.Request, publicOrigin string) (string, bool) {
	if publicOrigin != "" {
		parsed, err := url.Parse(publicOrigin)
		if err != nil || parsed.Host != r.Host {
			return "", false
		}
		return publicOrigin, true
	}
	host, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		return "", false
	}
	if host != "127.0.0.1" && host != "localhost" {
		return "", false
	}
	localAddress, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return "", false
	}
	_, localPort, err := net.SplitHostPort(localAddress.String())
	if err != nil || localPort != port {
		return "", false
	}
	return "http://" + net.JoinHostPort(host, port), true
}

func sessionCookieName(r *http.Request) string {
	_, port, _ := net.SplitHostPort(r.Host)
	return "atlas_mail_session_" + port
}

func staticAsset(path string, desktop bool) (string, string, bool) {
	if desktop && path == "/desktop" {
		return "index.html", "text/html; charset=utf-8", true
	}
	switch path {
	case "/", "/index.html":
		return "index.html", "text/html; charset=utf-8", true
	case "/client.js":
		return "client.js", "text/javascript; charset=utf-8", true
	case "/styles.css":
		return "styles.css", "text/css; charset=utf-8", true
	case "/favicon.svg":
		return "favicon.svg", "image/svg+xml", true
	case "/oauth-callback.js":
		return "oauth-callback.js", "text/javascript; charset=utf-8", true
	default:
		return "", "", false
	}
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request, config Config, origin string) {
	status := http.StatusOK
	success := false
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		status = http.StatusBadRequest
	} else if err := config.Service.CompleteOAuth(r.Context(), r.URL.Query().Get("state"),
		r.URL.Query().Get("code"), origin+"/oauth/callback"); err != nil {
		status = http.StatusBadRequest
	} else {
		success = true
	}
	message := "Authorization failed. Return to Atlas Mail and try again."
	if success {
		message = "Authorization completed. Return to Atlas Mail."
	}
	content := []byte("<!doctype html><html lang=\"en\"><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width\"><title>Atlas Mail authorization</title><body data-oauth-success=\"" + strconv.FormatBool(success) + "\"><main><h1>Atlas Mail</h1><p>" + message + "</p></main><script src=\"/oauth-callback.js\" defer></script></body></html>")
	writeResponse(w, status, "text/html; charset=utf-8", content)
}

func authenticate(header, token string) bool {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	actual := []byte(strings.TrimPrefix(header, prefix))
	expected := []byte(token)
	return len(actual) == len(expected) && subtle.ConstantTimeCompare(actual, expected) == 1
}

type requestError struct {
	status int
	code   string
}

func (e *requestError) Error() string { return e.code }

type accountSetupRequest struct {
	Account model.AccountInput `json:"account"`
	Profile mail.Profile       `json:"profile"`
	Secret  string             `json:"secret"`
}

func decodeRequest(w http.ResponseWriter, r *http.Request, maxBytes int64, target any) error {
	mediaType, parameters, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") ||
		(len(parameters) > 0 && (len(parameters) != 1 || !strings.EqualFold(parameters["charset"], "utf-8"))) {
		return &requestError{http.StatusBadRequest, "INVALID_CONTENT_TYPE"}
	}
	content, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBytes))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return &requestError{http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE"}
		}
		return &requestError{http.StatusBadRequest, "INVALID_JSON"}
	}
	if err := model.ValidateJSON(content); err != nil {
		return &requestError{http.StatusBadRequest, "INVALID_JSON"}
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &requestError{http.StatusBadRequest, "INVALID_JSON"}
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return &requestError{http.StatusBadRequest, "INVALID_JSON"}
	}
	return nil
}

func writeRequestFailure(w http.ResponseWriter, err error) {
	var requestErr *requestError
	if errors.As(err, &requestErr) {
		writeJSON(w, requestErr.status, map[string]any{"ok": false, "code": requestErr.code})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "code": "INVALID_JSON"})
}

func contextWithRequestTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 45*time.Second)
}

func readAccount(r *http.Request) (model.AccountInput, error) {
	mediaType, parameters, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaErr != nil || !strings.EqualFold(mediaType, "application/json") ||
		(len(parameters) > 0 && (len(parameters) != 1 || !strings.EqualFold(parameters["charset"], "utf-8"))) {
		return model.AccountInput{}, &requestError{http.StatusBadRequest, "INVALID_CONTENT_TYPE"}
	}
	data, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, maxRequestBytes))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return model.AccountInput{}, &requestError{http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE"}
		}
		return model.AccountInput{}, &requestError{http.StatusBadRequest, "INVALID_JSON"}
	}
	input, err := model.DecodeAccountInput(data)
	if err != nil {
		return model.AccountInput{}, &requestError{http.StatusBadRequest, "INVALID_JSON"}
	}
	return input, nil
}

func writeFailure(w http.ResponseWriter, err error) {
	var modelErr *model.Error
	if errors.As(err, &modelErr) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "code": modelErr.Code})
		return
	}
	var mailErr *mail.Error
	if errors.As(err, &mailErr) {
		status := http.StatusBadRequest
		switch mailErr.Code {
		case "AUTH_FAILED", "AUTH_REQUIRED", "CREDENTIAL_READ_FAILED":
			status = http.StatusUnauthorized
		case "TLS_CERTIFICATE_INVALID", "TLS_REQUIRED":
			status = http.StatusBadGateway
		case "CONNECTION_FAILED", "IMAP_CONNECTION_FAILED", "SMTP_CONNECTION_FAILED",
			"JMAP_CONNECTION_FAILED", "MAIL_RECEIVE_FAILED", "SMTP_SEND_FAILED":
			status = http.StatusBadGateway
		case "SEND_STATUS_UNCERTAIN":
			status = http.StatusConflict
		case "CREDENTIAL_STORE_UNAVAILABLE", "CREDENTIAL_REVOKE_FAILED":
			status = http.StatusServiceUnavailable
		case "ACCOUNT_NOT_FOUND":
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"ok": false, "code": mailErr.Code})
		return
	}
	var storeErr *store.Error
	if errors.As(err, &storeErr) {
		log.Printf("Atlas Mail storage error: %s", storeErr.Code)
		status := http.StatusInternalServerError
		if storeErr.Code == "STORE_LOCKED" {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]any{"ok": false, "code": storeErr.Code})
		return
	}
	log.Print("Atlas Mail request failed: UNEXPECTED_ERROR")
	writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "code": "UNEXPECTED_ERROR"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	content, err := json.Marshal(value)
	if err != nil {
		content = []byte(`{"ok":false,"code":"UNEXPECTED_ERROR"}`)
		status = http.StatusInternalServerError
	}
	writeResponse(w, status, "application/json; charset=utf-8", content)
}

func writeResponse(w http.ResponseWriter, status int, contentType string, content []byte) {
	headers := w.Header()
	headers.Set("Content-Type", contentType)
	headers.Set("Cache-Control", "no-store")
	headers.Set("Content-Security-Policy", contentSecurity)
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
	headers.Set("Referrer-Policy", "no-referrer")
	headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	w.WriteHeader(status)
	_, _ = w.Write(content)
}

func ListenAddress(port string) (string, error) {
	if port == "" {
		port = "3123"
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return "", fmt.Errorf("ATLAS_MAIL_PORT must be an integer from 1 to 65535")
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(number)), nil
}
