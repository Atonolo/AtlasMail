package service

import (
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
	Accounts        []model.Account `json:"accounts"`
	Adapters        []AdapterStatus `json:"adapters"`
	CredentialStore string          `json:"credentialStore"`
}

type MailService struct {
	store *store.StateStore
}

func New(stateStore *store.StateStore) *MailService {
	return &MailService{store: stateStore}
}

func (s *MailService) Status() (Status, error) {
	state, err := s.store.Read()
	if err != nil {
		return Status{}, err
	}
	adapters := NewAdapters()
	capabilities := make([]AdapterStatus, 0, len(adapters))
	for _, protocol := range []string{"imap", "smtp", "jmap", "caldav", "carddav"} {
		capabilities = append(capabilities, adapters[protocol].Capabilities())
	}
	return Status{
		ContractVersion: 1,
		SchemaVersion:   model.SchemaVersion,
		Accounts:        state.Accounts,
		Adapters:        capabilities,
		CredentialStore: "native-keyring; disabled until account integration",
	}, nil
}

func (s *MailService) AddAccount(input model.AccountInput) (model.Account, error) {
	return s.store.AddAccount(input)
}
