package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"unicode/utf8"
)

// decodeExact preserves the v1 contract: required, case-sensitive fields and
// no nulls or duplicate keys, including records with valid empty strings.
func decodeExact(data []byte, target any) error {
	if !utf8.Valid(data) {
		return errors.New("invalid UTF-8")
	}
	if err := checkEscapedUnicode(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := checkKeys(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	if err := checkShape(data, reflect.TypeOf(target).Elem()); err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func ValidateJSON(data []byte) error {
	if !utf8.Valid(data) {
		return errors.New("invalid UTF-8")
	}
	if err := checkEscapedUnicode(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := checkKeys(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

// encoding/json otherwise replaces unpaired UTF-16 escapes with U+FFFD.
func checkEscapedUnicode(data []byte) error {
	for index := 0; index < len(data); index++ {
		if data[index] != '\\' {
			continue
		}
		index++
		if index >= len(data) || data[index] != 'u' {
			continue
		}
		if index+4 >= len(data) {
			return errors.New("incomplete Unicode escape")
		}
		value, err := strconv.ParseUint(string(data[index+1:index+5]), 16, 16)
		if err != nil {
			return errors.New("invalid Unicode escape")
		}
		index += 4
		if value >= 0xdc00 && value <= 0xdfff {
			return errors.New("unpaired low surrogate")
		}
		if value < 0xd800 || value > 0xdbff {
			continue
		}
		if index+6 >= len(data) || data[index+1] != '\\' || data[index+2] != 'u' {
			return errors.New("unpaired high surrogate")
		}
		low, err := strconv.ParseUint(string(data[index+3:index+7]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return errors.New("invalid surrogate pair")
		}
		index += 6
	}
	return nil
}

func checkKeys(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	switch token {
	case json.Delim('{'):
		seen := make(map[string]bool)
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return errors.New("duplicate JSON field")
			}
			seen[name] = true
			if err := checkKeys(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case json.Delim('['):
		for decoder.More() {
			if err := checkKeys(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	}
	return nil
}

func checkShape(data []byte, expected reflect.Type) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null JSON value")
	}
	switch expected.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return err
		}
		if len(fields) != expected.NumField() {
			return errors.New("missing or unknown JSON field")
		}
		for index := 0; index < expected.NumField(); index++ {
			field := expected.Field(index)
			value, exists := fields[field.Tag.Get("json")]
			if !exists {
				return errors.New("missing JSON field")
			}
			if err := checkShape(value, field.Type); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var entries []json.RawMessage
		if err := json.Unmarshal(data, &entries); err != nil {
			return err
		}
		for _, entry := range entries {
			if err := checkShape(entry, expected.Elem()); err != nil {
				return err
			}
		}
	default:
		return json.Unmarshal(data, reflect.New(expected).Interface())
	}
	return nil
}

func DecodeAccountInput(data []byte) (AccountInput, error) {
	var input AccountInput
	if err := decodeExact(data, &input); err != nil {
		return AccountInput{}, &Error{Code: "INVALID_JSON"}
	}
	return input, nil
}
