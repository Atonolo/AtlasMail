package mail

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Atonolo/AtlasMail/internal/model"
	"github.com/Atonolo/AtlasMail/internal/store"
)

type Endpoint struct {
	Host    string         `json:"host"`
	Port    int            `json:"port"`
	TLSMode string         `json:"tlsMode"`
	RootCAs *x509.CertPool `json:"-"`
}

type OAuthSettings struct {
	AuthorizationURL string `json:"authorizationUrl"`
	TokenURL         string `json:"tokenUrl"`
	ClientID         string `json:"clientId"`
	Scope            string `json:"scope"`
}

type Profile struct {
	AccountID     string         `json:"accountId"`
	Username      string         `json:"username"`
	AuthMechanism string         `json:"authMechanism"`
	IMAP          Endpoint       `json:"imap"`
	SMTP          Endpoint       `json:"smtp"`
	JMAPURL       string         `json:"jmapUrl"`
	OAuth         *OAuthSettings `json:"oauth"`
}

type profilesDocument struct {
	Version  int       `json:"version"`
	Profiles []Profile `json:"profiles"`
}

type ProfileStore struct {
	state *store.StateStore
}

type Error struct {
	Code string
}

func (e *Error) Error() string { return e.Code }

var hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9.-]{0,251}[a-zA-Z0-9])?$`)

func NewProfileStore(state *store.StateStore) *ProfileStore {
	return &ProfileStore{state: state}
}

func (p *ProfileStore) Get(accountID string) (Profile, error) {
	profiles, err := p.read()
	if err != nil {
		return Profile{}, err
	}
	for _, profile := range profiles {
		if profile.AccountID == accountID {
			return profile, nil
		}
	}
	return Profile{}, &Error{Code: "ACCOUNT_NOT_CONFIGURED"}
}

func (p *ProfileStore) List() ([]Profile, error) {
	return p.read()
}

func (p *ProfileStore) Put(profile Profile) error {
	if err := ValidateProfile(profile); err != nil {
		return err
	}
	profiles, err := p.read()
	if err != nil {
		return err
	}
	for index := range profiles {
		if profiles[index].AccountID == profile.AccountID {
			profiles[index] = profile
			return p.write(profiles)
		}
	}
	profiles = append(profiles, profile)
	return p.write(profiles)
}

func (p *ProfileStore) Remove(accountID string) error {
	profiles, err := p.read()
	if err != nil {
		return err
	}
	for index := range profiles {
		if profiles[index].AccountID == accountID {
			profiles = append(profiles[:index], profiles[index+1:]...)
			return p.write(profiles)
		}
	}
	return nil
}

func (p *ProfileStore) read() ([]Profile, error) {
	content, err := p.state.ReadConnections()
	if err != nil {
		return nil, &Error{Code: "CONNECTIONS_READ_FAILED"}
	}
	if len(content) == 0 {
		return []Profile{}, nil
	}
	if err := model.ValidateJSON(content); err != nil {
		return nil, &Error{Code: "INVALID_CONNECTIONS"}
	}
	var document profilesDocument
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return nil, &Error{Code: "INVALID_CONNECTIONS"}
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, &Error{Code: "INVALID_CONNECTIONS"}
	}
	if document.Version != 1 || document.Profiles == nil {
		return nil, &Error{Code: "INVALID_CONNECTIONS"}
	}
	seen := make(map[string]bool, len(document.Profiles))
	for _, profile := range document.Profiles {
		if err := ValidateProfile(profile); err != nil || seen[profile.AccountID] {
			return nil, &Error{Code: "INVALID_CONNECTIONS"}
		}
		seen[profile.AccountID] = true
	}
	return document.Profiles, nil
}

func (p *ProfileStore) write(profiles []Profile) error {
	if len(profiles) == 0 {
		if err := p.state.WriteConnections(nil); err != nil {
			return &Error{Code: "CONNECTIONS_WRITE_FAILED"}
		}
		return nil
	}
	content, err := json.Marshal(profilesDocument{Version: 1, Profiles: profiles})
	if err != nil {
		return &Error{Code: "CONNECTIONS_WRITE_FAILED"}
	}
	if err := p.state.WriteConnections(append(content, '\n')); err != nil {
		return &Error{Code: "CONNECTIONS_WRITE_FAILED"}
	}
	return nil
}

func ValidateProfile(profile Profile) error {
	if !model.ValidIdentifier(profile.AccountID) || strings.TrimSpace(profile.Username) == "" ||
		len(profile.Username) > 254 || strings.ContainsFunc(profile.Username, func(char rune) bool {
		return char < 0x20 || char == 0x7f
	}) {
		return &Error{Code: "INVALID_ACCOUNT_CONFIGURATION"}
	}
	if profile.AuthMechanism != "password" && profile.AuthMechanism != "oauth2" {
		return &Error{Code: "INVALID_AUTH_MECHANISM"}
	}
	switch {
	case profile.JMAPURL != "":
		if profile.IMAP != (Endpoint{}) || profile.SMTP != (Endpoint{}) || !validJMAPURL(profile.JMAPURL) {
			return &Error{Code: "INVALID_JMAP_URL"}
		}
	case profile.IMAP != (Endpoint{}) || profile.SMTP != (Endpoint{}):
		if !validEndpoint(profile.IMAP) || !validEndpoint(profile.SMTP) || profile.JMAPURL != "" {
			return &Error{Code: "INVALID_MAIL_SERVER"}
		}
	default:
		return &Error{Code: "INVALID_MAIL_SERVER"}
	}
	if profile.AuthMechanism == "oauth2" {
		if profile.OAuth == nil || !validHTTPSURL(profile.OAuth.AuthorizationURL) ||
			!validHTTPSURL(profile.OAuth.TokenURL) || strings.TrimSpace(profile.OAuth.ClientID) == "" ||
			len(profile.OAuth.ClientID) > 512 || strings.TrimSpace(profile.OAuth.Scope) == "" ||
			len(profile.OAuth.Scope) > 2048 ||
			strings.ContainsAny(profile.OAuth.ClientID+profile.OAuth.Scope, "\r\n\x00") {
			return &Error{Code: "INVALID_OAUTH_CONFIGURATION"}
		}
	} else if profile.OAuth != nil {
		return &Error{Code: "INVALID_OAUTH_CONFIGURATION"}
	}
	return nil
}

func validEndpoint(endpoint Endpoint) bool {
	if endpoint.Port < 1 || endpoint.Port > 65535 ||
		(endpoint.TLSMode != "implicit" && endpoint.TLSMode != "starttls") {
		return false
	}
	if net.ParseIP(endpoint.Host) != nil {
		return true
	}
	return hostnamePattern.MatchString(endpoint.Host) && !strings.Contains(endpoint.Host, "..")
}

func validJMAPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	host, _, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		host = parsed.Hostname()
	}
	ip := net.ParseIP(host)
	return parsed.Scheme == "http" && (host == "localhost" || (ip != nil && ip.IsLoopback()))
}

func validHTTPSURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" &&
		parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}

func DefaultProfile(accountID, protocol, username string, imap, smtp Endpoint, jmapURL string) Profile {
	profile := Profile{
		AccountID: accountID, Username: username, AuthMechanism: "password",
		IMAP: imap, SMTP: smtp, JMAPURL: jmapURL,
	}
	if protocol == "imap-smtp" {
		if profile.IMAP.Port == 0 {
			profile.IMAP.Port, profile.IMAP.TLSMode = 993, "implicit"
		}
		if profile.SMTP.Port == 0 {
			profile.SMTP.Port, profile.SMTP.TLSMode = 465, "implicit"
		}
	}
	return profile
}

func ParsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, &Error{Code: "INVALID_PORT"}
	}
	return port, nil
}
