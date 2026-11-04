package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Atonolo/AtlasMail/internal/model"
)

const (
	maxFileBytes       = 10 * 1024 * 1024
	dataFilename       = "state-v1.json"
	connectionFilename = "connections-v1.json"
)

type Error struct {
	Code  string
	Cause error
}

func (e *Error) Error() string { return e.Code }
func (e *Error) Unwrap() error { return e.Cause }

type StateStore struct {
	directory string
	replace   func(string, string) error
}

func New(directory string) *StateStore {
	return &StateStore{directory: directory, replace: replaceFile}
}

func (s *StateStore) Directory() string { return s.directory }

func (s *StateStore) ReadConnections() ([]byte, error) {
	filename := filepath.Join(s.directory, connectionFilename)
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &Error{Code: "CONNECTIONS_READ_FAILED", Cause: err}
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if err != nil {
		return nil, &Error{Code: "CONNECTIONS_READ_FAILED", Cause: err}
	}
	if len(content) > 1024*1024 {
		return nil, &Error{Code: "CONNECTIONS_TOO_LARGE"}
	}
	return content, nil
}

func (s *StateStore) WriteConnections(content []byte) error {
	if len(content) > 1024*1024 {
		return &Error{Code: "CONNECTIONS_TOO_LARGE"}
	}
	filename := filepath.Join(s.directory, connectionFilename)
	stateFilename := filepath.Join(s.directory, dataFilename)
	return withLock(stateFilename, func() error {
		if len(content) == 0 {
			if err := os.Remove(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
				return &Error{Code: "CONNECTIONS_WRITE_FAILED", Cause: err}
			}
			return nil
		}
		if err := atomicWrite(filename, content, s.replace); err != nil {
			return &Error{Code: "CONNECTIONS_WRITE_FAILED", Cause: err}
		}
		return nil
	})
}

func DataDirectory() (string, error) {
	if override := os.Getenv("ATLAS_MAIL_DATA_DIR"); override != "" {
		if !filepath.IsAbs(override) {
			return "", &Error{Code: "INVALID_DATA_DIRECTORY"}
		}
		return override, nil
	}
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("APPDATA")
		if base == "" {
			var err error
			base, err = os.UserConfigDir()
			if err != nil {
				return "", &Error{Code: "STORAGE_UNAVAILABLE", Cause: err}
			}
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", &Error{Code: "STORAGE_UNAVAILABLE", Cause: err}
		}
		base = filepath.Join(home, "Library", "Application Support")
	case "linux":
		base = os.Getenv("XDG_DATA_HOME")
		if base != "" && !filepath.IsAbs(base) {
			return "", &Error{Code: "INVALID_DATA_DIRECTORY"}
		}
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", &Error{Code: "STORAGE_UNAVAILABLE", Cause: err}
			}
			base = filepath.Join(home, ".local", "share")
		}
	default:
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", &Error{Code: "STORAGE_UNAVAILABLE", Cause: err}
		}
	}
	return filepath.Join(base, "AtlasMail"), nil
}

func (s *StateStore) Read() (model.State, error) {
	filename := filepath.Join(s.directory, dataFilename)
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		return model.EmptyState(), nil
	}
	if err != nil {
		return model.State{}, &Error{Code: "READ_FAILED", Cause: err}
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil {
		return model.State{}, &Error{Code: "READ_FAILED", Cause: err}
	}
	if len(content) > maxFileBytes {
		return model.State{}, &Error{Code: "DATA_TOO_LARGE"}
	}
	state, err := model.DecodeState(content)
	if err != nil {
		return model.State{}, &Error{Code: errorCode(err), Cause: err}
	}
	return state, nil
}

func (s *StateStore) AddAccount(input model.AccountInput) (model.Account, error) {
	account, err := model.MakeAccount(input)
	if err != nil {
		return model.Account{}, err
	}
	err = s.Update(func(state *model.State) error {
		state.Accounts = append(state.Accounts, account)
		return nil
	})
	if err != nil {
		return model.Account{}, err
	}
	return account, nil
}

func (s *StateStore) RemoveAccount(id string) error {
	if !model.ValidIdentifier(id) {
		return &model.Error{Code: "INVALID_IDENTIFIER"}
	}
	return s.Update(func(state *model.State) error {
		accountIndex := -1
		for index, account := range state.Accounts {
			if account.ID == id {
				accountIndex = index
				break
			}
		}
		if accountIndex < 0 {
			return &model.Error{Code: "ACCOUNT_NOT_FOUND"}
		}
		state.Accounts = append(state.Accounts[:accountIndex], state.Accounts[accountIndex+1:]...)
		calendarIDs := make(map[string]bool)
		keptCalendars := state.Calendars[:0]
		for _, calendar := range state.Calendars {
			if calendar.AccountID == id {
				calendarIDs[calendar.ID] = true
				continue
			}
			keptCalendars = append(keptCalendars, calendar)
		}
		state.Calendars = keptCalendars
		eventIDs := make(map[string]bool)
		keptEvents := state.Events[:0]
		for _, event := range state.Events {
			if calendarIDs[event.CalendarID] {
				eventIDs[event.ID] = true
				continue
			}
			keptEvents = append(keptEvents, event)
		}
		state.Events = keptEvents
		messageIDs, contactIDs, taskIDs := make(map[string]bool), make(map[string]bool), make(map[string]bool)
		keptMessages := state.Messages[:0]
		for _, message := range state.Messages {
			if message.AccountID == id {
				messageIDs[message.ID] = true
				continue
			}
			keptMessages = append(keptMessages, message)
		}
		state.Messages = keptMessages
		keptContacts := state.Contacts[:0]
		for _, contact := range state.Contacts {
			if contact.AccountID == id {
				contactIDs[contact.ID] = true
				continue
			}
			keptContacts = append(keptContacts, contact)
		}
		state.Contacts = keptContacts
		keptTasks := state.Tasks[:0]
		for _, task := range state.Tasks {
			if task.AccountID == id {
				taskIDs[task.ID] = true
				continue
			}
			keptTasks = append(keptTasks, task)
		}
		state.Tasks = keptTasks
		removedEntities := make(map[string]bool)
		for entityID := range messageIDs {
			removedEntities[entityID] = true
		}
		for entityID := range eventIDs {
			removedEntities[entityID] = true
		}
		for entityID := range contactIDs {
			removedEntities[entityID] = true
		}
		for entityID := range taskIDs {
			removedEntities[entityID] = true
		}
		keptCache := state.Cache[:0]
		for _, entry := range state.Cache {
			if entry.AccountID != id && !removedEntities[entry.EntityID] {
				keptCache = append(keptCache, entry)
			}
		}
		state.Cache = keptCache
		keptQueue := state.Queue[:0]
		for _, entry := range state.Queue {
			if entry.AccountID != id && !removedEntities[entry.EntityID] {
				keptQueue = append(keptQueue, entry)
			}
		}
		state.Queue = keptQueue
		return nil
	})
}

func (s *StateStore) Update(change func(*model.State) error) error {
	filename := filepath.Join(s.directory, dataFilename)
	return withLock(filename, func() error {
		state, err := s.Read()
		if err != nil {
			return err
		}
		if err := change(&state); err != nil {
			return err
		}
		content, err := state.Marshal()
		if err != nil {
			return err
		}
		if len(content) > maxFileBytes {
			return &Error{Code: "DATA_TOO_LARGE"}
		}
		if err := atomicWrite(filename, content, s.replace); err != nil {
			return err
		}
		return nil
	})
}

func (s *StateStore) ExportBackup() ([]byte, error) {
	state, err := s.Read()
	if err != nil {
		return nil, err
	}
	return state.Marshal()
}

func (s *StateStore) RestoreBackup(content []byte) error {
	if len(content) > maxFileBytes {
		return &Error{Code: "DATA_TOO_LARGE"}
	}
	state, err := model.DecodeState(content)
	if err != nil {
		return &Error{Code: errorCode(err), Cause: err}
	}
	return s.Update(func(current *model.State) error {
		*current = state
		return nil
	})
}

func withLock(filename string, operation func() error) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		return &Error{Code: "STORAGE_UNAVAILABLE", Cause: err}
	}
	lockName := filename + ".lock"
	lock, err := os.OpenFile(lockName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return &Error{Code: "STORE_LOCKED", Cause: err}
		}
		return &Error{Code: "STORAGE_UNAVAILABLE", Cause: err}
	}
	_, writeErr := fmt.Fprintf(lock, "%d\n", os.Getpid())
	if writeErr == nil {
		writeErr = lock.Sync()
	}
	if writeErr != nil {
		closeErr := lock.Close()
		removeErr := os.Remove(lockName)
		if closeErr != nil || removeErr != nil {
			return &Error{Code: "LOCK_CLEANUP_FAILED", Cause: errors.Join(writeErr, closeErr, removeErr)}
		}
		return &Error{Code: "STORAGE_UNAVAILABLE", Cause: writeErr}
	}
	operationErr := operation()
	closeErr := lock.Close()
	removeErr := os.Remove(lockName)
	if closeErr != nil || removeErr != nil {
		return &Error{Code: "COMMIT_UNCERTAIN", Cause: errors.Join(operationErr, closeErr, removeErr)}
	}
	return operationErr
}

func atomicWrite(filename string, content []byte, replace func(string, string) error) (result error) {
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		return &Error{Code: "STORAGE_UNAVAILABLE", Cause: err}
	}
	temporary, err := os.CreateTemp(filepath.Dir(filename), filepath.Base(filename)+".*.tmp")
	if err != nil {
		return &Error{Code: "WRITE_FAILED", Cause: err}
	}
	name := temporary.Name()
	closed := false
	committed := false
	defer func() {
		var closeErr, removeErr error
		if !closed {
			closeErr = temporary.Close()
		}
		if !committed {
			removeErr = os.Remove(name)
		}
		if closeErr != nil || removeErr != nil {
			result = &Error{Code: "WRITE_FAILED", Cause: errors.Join(result, closeErr, removeErr)}
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return &Error{Code: "WRITE_FAILED", Cause: err}
	}
	if _, err := temporary.Write(content); err != nil {
		return &Error{Code: "WRITE_FAILED", Cause: err}
	}
	if err := temporary.Sync(); err != nil {
		return &Error{Code: "WRITE_FAILED", Cause: err}
	}
	err = temporary.Close()
	closed = true
	if err != nil {
		return &Error{Code: "WRITE_FAILED", Cause: err}
	}
	if err := replace(name, filename); err != nil {
		return &Error{Code: "WRITE_FAILED", Cause: err}
	}
	committed = true
	return nil
}

func ParseState(content []byte) (model.State, error) {
	if len(content) > maxFileBytes {
		return model.State{}, &Error{Code: "DATA_TOO_LARGE"}
	}
	return model.DecodeState(content)
}

func errorCode(err error) string {
	var modelErr *model.Error
	if errors.As(err, &modelErr) {
		return modelErr.Code
	}
	var storeErr *Error
	if errors.As(err, &storeErr) {
		return storeErr.Code
	}
	return "INVALID_STORED_DATA"
}
