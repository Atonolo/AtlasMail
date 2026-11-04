package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Atonolo/AtlasMail/internal/mail"
)

type oauthToken struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type pendingOAuth struct {
	accountID string
	verifier  string
	redirect  string
	expiresAt time.Time
}

func (s *MailService) BeginOAuth(accountID, redirectURI string) (string, error) {
	if _, err := s.account(accountID); err != nil {
		return "", err
	}
	profile, err := s.profiles.Get(accountID)
	if err != nil {
		return "", err
	}
	if profile.AuthMechanism != "oauth2" || profile.OAuth == nil {
		return "", &mail.Error{Code: "OAUTH_NOT_CONFIGURED"}
	}
	if !validRedirectURI(redirectURI) {
		return "", &mail.Error{Code: "INVALID_OAUTH_REDIRECT"}
	}
	stateBytes := make([]byte, 32)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", &mail.Error{Code: "OAUTH_START_FAILED"}
	}
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", &mail.Error{Code: "OAUTH_START_FAILED"}
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	digest := sha256.Sum256([]byte(verifier))
	authorizationURL, err := url.Parse(profile.OAuth.AuthorizationURL)
	if err != nil {
		return "", &mail.Error{Code: "INVALID_OAUTH_CONFIGURATION"}
	}
	query := authorizationURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", profile.OAuth.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("scope", profile.OAuth.Scope)
	query.Set("state", state)
	query.Set("code_challenge", base64.RawURLEncoding.EncodeToString(digest[:]))
	query.Set("code_challenge_method", "S256")
	authorizationURL.RawQuery = query.Encode()
	s.oauthMu.Lock()
	if s.pendingOAuth == nil {
		s.pendingOAuth = make(map[string]pendingOAuth)
	}
	now := time.Now()
	for pendingState, pending := range s.pendingOAuth {
		if now.After(pending.expiresAt) {
			delete(s.pendingOAuth, pendingState)
		}
	}
	s.pendingOAuth[state] = pendingOAuth{
		accountID: accountID, verifier: verifier, redirect: redirectURI, expiresAt: now.Add(10 * time.Minute),
	}
	s.oauthMu.Unlock()
	return authorizationURL.String(), nil
}

func (s *MailService) CompleteOAuth(ctx context.Context, state, code, redirectURI string) error {
	if len(state) < 32 || len(state) > 128 || len(code) == 0 || len(code) > 4096 {
		return &mail.Error{Code: "OAUTH_CALLBACK_INVALID"}
	}
	s.oauthMu.Lock()
	pending, exists := s.pendingOAuth[state]
	delete(s.pendingOAuth, state)
	s.oauthMu.Unlock()
	if !exists || time.Now().After(pending.expiresAt) || pending.redirect != redirectURI {
		return &mail.Error{Code: "OAUTH_STATE_INVALID"}
	}
	profile, err := s.profiles.Get(pending.accountID)
	if err != nil || profile.OAuth == nil {
		return &mail.Error{Code: "OAUTH_NOT_CONFIGURED"}
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {profile.OAuth.ClientID},
		"code_verifier": {pending.verifier},
	}
	token, err := requestOAuthToken(ctx, profile.OAuth.TokenURL, form)
	if err != nil {
		return err
	}
	content, err := json.Marshal(token)
	if err != nil || len(content) > 2048 {
		return &mail.Error{Code: "OAUTH_TOKEN_TOO_LARGE"}
	}
	if err := s.credentials.Put(pending.accountID, string(content)); err != nil {
		return &mail.Error{Code: "CREDENTIAL_STORE_UNAVAILABLE"}
	}
	return nil
}

func (s *MailService) refreshOAuth(ctx context.Context, accountID string, profile mail.Profile, token oauthToken) (oauthToken, error) {
	if profile.OAuth == nil || token.RefreshToken == "" {
		return oauthToken{}, &mail.Error{Code: "AUTH_EXPIRED"}
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"client_id":     {profile.OAuth.ClientID},
	}
	refreshed, err := requestOAuthToken(ctx, profile.OAuth.TokenURL, form)
	if err != nil {
		return oauthToken{}, err
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = token.RefreshToken
	}
	content, err := json.Marshal(refreshed)
	if err != nil || len(content) > 2048 {
		return oauthToken{}, &mail.Error{Code: "OAUTH_TOKEN_TOO_LARGE"}
	}
	if err := s.credentials.Put(accountID, string(content)); err != nil {
		return oauthToken{}, &mail.Error{Code: "CREDENTIAL_STORE_UNAVAILABLE"}
	}
	return refreshed, nil
}

func requestOAuthToken(ctx context.Context, endpoint string, form url.Values) (oauthToken, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return oauthToken{}, &mail.Error{Code: "OAUTH_TOKEN_EXCHANGE_FAILED"}
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := client.Do(request)
	if err != nil {
		return oauthToken{}, &mail.Error{Code: "OAUTH_TOKEN_EXCHANGE_FAILED"}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return oauthToken{}, &mail.Error{Code: "OAUTH_TOKEN_EXCHANGE_FAILED"}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if err != nil || len(data) > 64*1024 {
		return oauthToken{}, &mail.Error{Code: "OAUTH_TOKEN_EXCHANGE_FAILED"}
	}
	var responseData struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if json.Unmarshal(data, &responseData) != nil || responseData.AccessToken == "" ||
		len(responseData.AccessToken) > 4096 || len(responseData.RefreshToken) > 4096 ||
		strings.ContainsAny(responseData.AccessToken+responseData.RefreshToken, "\r\n\x00") ||
		responseData.ExpiresIn < 0 || responseData.ExpiresIn > 366*24*60*60 {
		return oauthToken{}, &mail.Error{Code: "OAUTH_TOKEN_EXCHANGE_FAILED"}
	}
	if responseData.ExpiresIn == 0 {
		responseData.ExpiresIn = 3600
	}
	return oauthToken{
		AccessToken: responseData.AccessToken, RefreshToken: responseData.RefreshToken,
		ExpiresAt: time.Now().Add(time.Duration(responseData.ExpiresIn) * time.Second),
	}, nil
}

func validRedirectURI(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		parsed.Path != "/oauth/callback" {
		return false
	}
	if parsed.Scheme == "https" {
		return parsed.Host != ""
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	return parsed.Scheme == "http" && (host == "localhost" || (ip != nil && ip.IsLoopback()))
}
