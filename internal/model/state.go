package model

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

const SchemaVersion = 1

var (
	identifierPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	emailPattern      = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

type Error struct {
	Code string
}

func (e *Error) Error() string { return e.Code }

type AccountInput struct {
	Label    string `json:"label"`
	Address  string `json:"address"`
	Protocol string `json:"protocol"`
}

type Account struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Address  string `json:"address"`
	Protocol string `json:"protocol"`
}

type Message struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	RemoteID  string `json:"remoteId"`
	FolderID  string `json:"folderId"`
}

type Calendar struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	RemoteID  string `json:"remoteId"`
}

type Event struct {
	ID         string `json:"id"`
	CalendarID string `json:"calendarId"`
	RemoteID   string `json:"remoteId"`
}

type Contact struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	RemoteID  string `json:"remoteId"`
}

type Task struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	RemoteID  string `json:"remoteId"`
}

type CacheEntry struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	EntityID  string `json:"entityId"`
	Kind      string `json:"kind"`
	State     string `json:"state"`
}

type QueueEntry struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	EntityID  string `json:"entityId"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type State struct {
	SchemaVersion int          `json:"schemaVersion"`
	Accounts      []Account    `json:"accounts"`
	Messages      []Message    `json:"messages"`
	Calendars     []Calendar   `json:"calendars"`
	Events        []Event      `json:"events"`
	Contacts      []Contact    `json:"contacts"`
	Tasks         []Task       `json:"tasks"`
	Cache         []CacheEntry `json:"cache"`
	Queue         []QueueEntry `json:"queue"`
}

func EmptyState() State {
	return State{
		SchemaVersion: SchemaVersion,
		Accounts:      []Account{},
		Messages:      []Message{},
		Calendars:     []Calendar{},
		Events:        []Event{},
		Contacts:      []Contact{},
		Tasks:         []Task{},
		Cache:         []CacheEntry{},
		Queue:         []QueueEntry{},
	}
}

func NewIdentifier() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}

func MakeAccount(input AccountInput) (Account, error) {
	id, err := NewIdentifier()
	if err != nil {
		return Account{}, err
	}
	account := Account{
		ID:       id,
		Label:    strings.TrimSpace(input.Label),
		Address:  strings.TrimSpace(input.Address),
		Protocol: input.Protocol,
	}
	if err := ValidateAccount(account); err != nil {
		return Account{}, err
	}
	return account, nil
}

func ValidateAccount(account Account) error {
	if !ValidIdentifier(account.ID) {
		return &Error{Code: "INVALID_IDENTIFIER"}
	}
	if !validText(account.Label, 80, false) || !validText(account.Address, 254, false) ||
		!emailPattern.MatchString(account.Address) ||
		(account.Protocol != "imap-smtp" && account.Protocol != "jmap") {
		return &Error{Code: "INVALID_ACCOUNT"}
	}
	return nil
}

func ValidIdentifier(value string) bool { return identifierPattern.MatchString(value) }

func DecodeState(data []byte) (State, error) {
	var state State
	var version struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &version); err != nil {
		return State{}, &Error{Code: "INVALID_STORED_DATA"}
	}
	if version.SchemaVersion != SchemaVersion {
		return State{}, &Error{Code: "UNSUPPORTED_SCHEMA"}
	}
	if err := decodeExact(data, &state); err != nil {
		return State{}, &Error{Code: "INVALID_STORED_DATA"}
	}
	if err := ValidateState(state); err != nil {
		return State{}, err
	}
	return state, nil
}

func ValidateState(state State) error {
	if state.SchemaVersion != SchemaVersion {
		return &Error{Code: "UNSUPPORTED_SCHEMA"}
	}
	collections := []int{
		len(state.Accounts), len(state.Messages), len(state.Calendars), len(state.Events),
		len(state.Contacts), len(state.Tasks), len(state.Cache), len(state.Queue),
	}
	if state.Accounts == nil || state.Messages == nil || state.Calendars == nil || state.Events == nil ||
		state.Contacts == nil || state.Tasks == nil || state.Cache == nil || state.Queue == nil {
		return &Error{Code: "INVALID_DATA"}
	}
	for _, count := range collections {
		if count > 10_000 {
			return &Error{Code: "INVALID_DATA"}
		}
	}

	identifiers := make(map[string]bool)
	accounts := make(map[string]bool)
	calendars := make(map[string]string)
	register := func(id string) error {
		if !ValidIdentifier(id) {
			return &Error{Code: "INVALID_IDENTIFIER"}
		}
		if identifiers[id] {
			return &Error{Code: "DUPLICATE_IDENTIFIER"}
		}
		identifiers[id] = true
		return nil
	}
	requireAccount := func(id string) error {
		if !ValidIdentifier(id) || !accounts[id] {
			return &Error{Code: "INVALID_REFERENCE"}
		}
		return nil
	}
	for _, account := range state.Accounts {
		if err := ValidateAccount(account); err != nil {
			return err
		}
		if err := register(account.ID); err != nil {
			return err
		}
		accounts[account.ID] = true
	}
	for _, item := range state.Messages {
		if err := requireAccount(item.AccountID); err != nil {
			return err
		}
		if !validText(item.RemoteID, 1024, true) || !validText(item.FolderID, 1024, true) {
			return &Error{Code: "INVALID_DATA"}
		}
		if err := register(item.ID); err != nil {
			return err
		}
	}
	for _, item := range state.Calendars {
		if err := requireAccount(item.AccountID); err != nil {
			return err
		}
		if !validText(item.RemoteID, 1024, true) {
			return &Error{Code: "INVALID_DATA"}
		}
		if err := register(item.ID); err != nil {
			return err
		}
		calendars[item.ID] = item.AccountID
	}
	for _, item := range state.Events {
		_, exists := calendars[item.CalendarID]
		if !exists || !validText(item.RemoteID, 1024, true) {
			return &Error{Code: "INVALID_REFERENCE"}
		}
		if err := register(item.ID); err != nil {
			return err
		}
	}
	for _, item := range state.Contacts {
		if err := requireAccount(item.AccountID); err != nil {
			return err
		}
		if !validText(item.RemoteID, 1024, true) {
			return &Error{Code: "INVALID_DATA"}
		}
		if err := register(item.ID); err != nil {
			return err
		}
	}
	for _, item := range state.Tasks {
		if err := requireAccount(item.AccountID); err != nil {
			return err
		}
		if !validText(item.RemoteID, 1024, true) {
			return &Error{Code: "INVALID_DATA"}
		}
		if err := register(item.ID); err != nil {
			return err
		}
	}
	for _, item := range state.Cache {
		if err := validateReference(item.ID, item.AccountID, item.EntityID, item.Kind); err != nil {
			return err
		}
		if err := register(item.ID); err != nil {
			return err
		}
		if item.State != "metadata" {
			return &Error{Code: "INVALID_DATA"}
		}
	}
	for _, item := range state.Queue {
		if err := validateReference(item.ID, item.AccountID, item.EntityID, item.Kind); err != nil {
			return err
		}
		if err := register(item.ID); err != nil {
			return err
		}
		if item.Status != "pending" && item.Status != "in-progress" && item.Status != "failed" && item.Status != "complete" {
			return &Error{Code: "INVALID_DATA"}
		}
		createdAt, err := time.Parse("2006-01-02T15:04:05.000Z", item.CreatedAt)
		if err != nil || createdAt.UTC().Format("2006-01-02T15:04:05.000Z") != item.CreatedAt {
			return &Error{Code: "INVALID_DATA"}
		}
	}
	if err := ValidateReferences(state); err != nil {
		return err
	}
	for _, entry := range state.Cache {
		if !accounts[entry.AccountID] {
			return &Error{Code: "INVALID_REFERENCE"}
		}
	}
	for _, entry := range state.Queue {
		if !accounts[entry.AccountID] {
			return &Error{Code: "INVALID_REFERENCE"}
		}
	}
	return nil
}

type entity struct {
	kind  string
	owner string
}

func validateReference(id, accountID, entityID, kind string) error {
	if !ValidIdentifier(id) || !ValidIdentifier(accountID) || !ValidIdentifier(entityID) {
		return &Error{Code: "INVALID_IDENTIFIER"}
	}
	if kind != "message" && kind != "event" && kind != "contact" && kind != "task" {
		return &Error{Code: "INVALID_DATA"}
	}
	return nil
}

func validText(value string, maxLength int, allowEmpty bool) bool {
	if !utf8.ValidString(value) {
		return false
	}
	length := 0
	for _, char := range value {
		length += utf16.RuneLen(char)
	}
	if length > maxLength || (!allowEmpty && (value == "" || strings.TrimSpace(value) != value)) {
		return false
	}
	return true
}

func ValidateReferences(state State) error {
	entities := make(map[string]entity)
	calendars := make(map[string]string, len(state.Calendars))
	for _, calendar := range state.Calendars {
		calendars[calendar.ID] = calendar.AccountID
	}
	for _, item := range state.Messages {
		entities[item.ID] = entity{"message", item.AccountID}
	}
	for _, item := range state.Events {
		entities[item.ID] = entity{"event", calendars[item.CalendarID]}
	}
	for _, item := range state.Contacts {
		entities[item.ID] = entity{"contact", item.AccountID}
	}
	for _, item := range state.Tasks {
		entities[item.ID] = entity{"task", item.AccountID}
	}
	for _, reference := range state.Cache {
		if err := checkReference(reference.AccountID, reference.EntityID, reference.Kind, entities); err != nil {
			return err
		}
	}
	for _, reference := range state.Queue {
		if err := checkReference(reference.AccountID, reference.EntityID, reference.Kind, entities); err != nil {
			return err
		}
	}
	return nil
}

func checkReference(accountID, entityID, kind string, entities map[string]entity) error {
	target, exists := entities[entityID]
	if !exists || target.owner != accountID || target.kind != kind {
		return &Error{Code: "INVALID_REFERENCE"}
	}
	return nil
}

func (s State) Marshal() ([]byte, error) {
	if err := ValidateState(s); err != nil {
		return nil, err
	}
	result, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode state: %w", err)
	}
	return append(result, '\n'), nil
}
