package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Atonolo/AtlasMail/internal/credentials"
	"github.com/Atonolo/AtlasMail/internal/mail"
	"github.com/Atonolo/AtlasMail/internal/model"
	"github.com/Atonolo/AtlasMail/internal/store"
)

type AdapterStatus struct {
	ContractVersion int    `json:"contractVersion"`
	Protocol        string `json:"protocol"`
	Available       bool   `json:"available"`
}

type Status struct {
	ContractVersion int             `json:"contractVersion"`
	SchemaVersion   int             `json:"schemaVersion"`
	Accounts        []AccountStatus `json:"accounts"`
	Adapters        []AdapterStatus `json:"adapters"`
	CredentialStore string          `json:"credentialStore"`
}

type AccountStatus struct {
	model.Account
	Configured bool `json:"configured"`
}

type MailService struct {
	store        *store.StateStore
	profiles     *mail.ProfileStore
	credentials  *credentials.Store
	oauthMu      sync.Mutex
	pendingOAuth map[string]pendingOAuth
}

func New(stateStore *store.StateStore) *MailService {
	return NewWithCredentials(stateStore, credentials.New())
}

func NewWithCredentials(stateStore *store.StateStore, credentialStore *credentials.Store) *MailService {
	return &MailService{
		store: stateStore, profiles: mail.NewProfileStore(stateStore), credentials: credentialStore,
	}
}

func (s *MailService) Status() (Status, error) {
	state, err := s.store.Read()
	if err != nil {
		return Status{}, err
	}
	profiles, err := s.profiles.List()
	if err != nil {
		return Status{}, err
	}
	configured := make(map[string]bool, len(profiles))
	for _, profile := range profiles {
		configured[profile.AccountID] = true
	}
	accounts := make([]AccountStatus, 0, len(state.Accounts))
	for _, account := range state.Accounts {
		accounts = append(accounts, AccountStatus{Account: account, Configured: configured[account.ID]})
	}
	return Status{
		ContractVersion: 1,
		SchemaVersion:   model.SchemaVersion,
		Accounts:        accounts,
		Adapters:        AdapterStatuses(),
		CredentialStore: "native-keyring",
	}, nil
}

func (s *MailService) AddAccount(input model.AccountInput) (model.Account, error) {
	return s.store.AddAccount(input)
}

func (s *MailService) ConfigureAccount(input model.AccountInput, profile mail.Profile, secret string) (model.Account, error) {
	account, err := model.MakeAccount(input)
	if err != nil {
		return model.Account{}, err
	}
	profile.AccountID = account.ID
	if profile.Username == "" {
		profile.Username = account.Address
	}
	if (account.Protocol == "jmap") != (profile.JMAPURL != "") {
		return model.Account{}, &mail.Error{Code: "INVALID_ACCOUNT_CONFIGURATION"}
	}
	if err := mail.ValidateProfile(profile); err != nil {
		return model.Account{}, err
	}
	if len(secret) > 2048 || strings.ContainsFunc(secret, func(char rune) bool {
		return char < 0x20 || char == 0x7f
	}) || (profile.AuthMechanism == "password" && strings.TrimSpace(secret) == "") ||
		(profile.AuthMechanism == "oauth2" && secret != "") {
		return model.Account{}, &mail.Error{Code: "INVALID_CREDENTIAL"}
	}
	if err := s.store.Update(func(state *model.State) error {
		state.Accounts = append(state.Accounts, account)
		return nil
	}); err != nil {
		return model.Account{}, err
	}
	if secret != "" {
		if err := s.credentials.Put(account.ID, secret); err != nil {
			return model.Account{}, errors.Join(&mail.Error{Code: "CREDENTIAL_STORE_UNAVAILABLE"}, s.store.RemoveAccount(account.ID))
		}
	}
	if err := s.profiles.Put(profile); err != nil {
		credentialErr := s.credentials.Delete(account.ID)
		removeErr := s.store.RemoveAccount(account.ID)
		return model.Account{}, errors.Join(err, credentialErr, removeErr)
	}
	return account, nil
}

func (s *MailService) RemoveAccount(accountID string) error {
	if _, err := s.account(accountID); err != nil {
		return err
	}
	if err := s.credentials.Delete(accountID); err != nil {
		return &mail.Error{Code: "CREDENTIAL_REVOKE_FAILED"}
	}
	if err := s.profiles.Remove(accountID); err != nil {
		return err
	}
	return s.store.RemoveAccount(accountID)
}

func (s *MailService) Receive(ctx context.Context, accountID string) ([]mail.Message, error) {
	account, profile, secret, err := s.connection(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.Protocol == "jmap" {
		return mail.ReceiveJMAP(ctx, profile, secret)
	}
	return mail.ReceiveIMAP(ctx, profile, secret)
}

func (s *MailService) Send(ctx context.Context, input mail.SendInput) error {
	account, profile, secret, err := s.connection(ctx, input.AccountID)
	if err != nil {
		return err
	}
	if _, err := mail.BuildMessage(account, input); err != nil {
		return err
	}
	if account.Protocol == "jmap" {
		return mail.SendJMAP(ctx, profile, secret, account.Address, input)
	}
	return mail.SendSMTP(ctx, profile, secret, account.Address, input)
}

func (s *MailService) account(accountID string) (model.Account, error) {
	if !model.ValidIdentifier(accountID) {
		return model.Account{}, &model.Error{Code: "INVALID_IDENTIFIER"}
	}
	state, err := s.store.Read()
	if err != nil {
		return model.Account{}, err
	}
	for _, account := range state.Accounts {
		if account.ID == accountID {
			return account, nil
		}
	}
	return model.Account{}, &mail.Error{Code: "ACCOUNT_NOT_FOUND"}
}

func (s *MailService) connection(ctx context.Context, accountID string) (model.Account, mail.Profile, string, error) {
	account, err := s.account(accountID)
	if err != nil {
		return model.Account{}, mail.Profile{}, "", err
	}
	profile, err := s.profiles.Get(accountID)
	if err != nil {
		return model.Account{}, mail.Profile{}, "", err
	}
	secret, err := s.credentials.Get(accountID)
	if err != nil {
		return model.Account{}, mail.Profile{}, "", &mail.Error{Code: "CREDENTIAL_READ_FAILED"}
	}
	if secret == "" {
		return model.Account{}, mail.Profile{}, "", &mail.Error{Code: "AUTH_REQUIRED"}
	}
	if profile.AuthMechanism == "oauth2" {
		var token oauthToken
		if json.Unmarshal([]byte(secret), &token) != nil || token.AccessToken == "" {
			return model.Account{}, mail.Profile{}, "", &mail.Error{Code: "AUTH_REQUIRED"}
		}
		if !token.ExpiresAt.After(time.Now().Add(30 * time.Second)) {
			token, err = s.refreshOAuth(ctx, accountID, profile, token)
			if err != nil {
				return model.Account{}, mail.Profile{}, "", err
			}
		}
		secret = token.AccessToken
	}
	return account, profile, secret, nil
}
