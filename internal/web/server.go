package web

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Atonolo/AtlasMail/internal/model"
	"github.com/Atonolo/AtlasMail/internal/service"
	"github.com/Atonolo/AtlasMail/internal/store"
)

const (
	maxRequestBytes = 16 * 1024
	contentSecurity = "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'"
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
	if !config.Desktop && !validToken(config.Token) {
		return nil, errors.New("ATLAS_MAIL_WEB_TOKEN must be a random base64url token of 43-128 characters")
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
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "code": "NOT_FOUND"})
		return
	}
	if !config.Desktop && !authenticate(r.Header.Get("Authorization"), config.Token) {
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
	return "http://" + net.JoinHostPort(host, port), true
}

func staticAsset(path string, desktop bool) (string, string, bool) {
	if desktop && path == "/desktop" {
		return "index.html", "text/html; charset=utf-8", true
	}
	switch path {
	case "/", "/index.html":
		return "index.html", "text/html; charset=utf-8", true
	case "/app.js":
		return "app.js", "text/javascript; charset=utf-8", true
	case "/styles.css":
		return "styles.css", "text/css; charset=utf-8", true
	case "/favicon.svg":
		return "favicon.svg", "image/svg+xml", true
	default:
		return "", "", false
	}
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

func readAccount(r *http.Request) (model.AccountInput, error) {
	mediaType, parameters, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaErr != nil || !strings.EqualFold(mediaType, "application/json") ||
		(len(parameters) > 0 && (len(parameters) != 1 || !strings.EqualFold(parameters["charset"], "utf-8"))) {
		return model.AccountInput{}, &requestError{http.StatusBadRequest, "INVALID_CONTENT_TYPE"}
	}
	reader := http.MaxBytesReader(nil, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var input model.AccountInput
	if err := decoder.Decode(&input); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return model.AccountInput{}, &requestError{http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE"}
		}
		return model.AccountInput{}, &requestError{http.StatusBadRequest, "INVALID_JSON"}
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
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
	var storeErr *store.Error
	if errors.As(err, &storeErr) {
		status := http.StatusInternalServerError
		if storeErr.Code == "STORE_LOCKED" {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]any{"ok": false, "code": storeErr.Code})
		return
	}
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
