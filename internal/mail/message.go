package mail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Atonolo/AtlasMail/internal/model"
)

const (
	maxMessageBytes    = 15 * 1024 * 1024
	maxAttachmentBytes = 10 * 1024 * 1024
	maxAttachments     = 20
	maxInboxBytes      = 15 * 1024 * 1024
)

type Attachment struct {
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Data        []byte `json:"data"`
}

type Message struct {
	ID          string       `json:"id"`
	AccountID   string       `json:"accountId"`
	MessageID   string       `json:"messageId"`
	References  string       `json:"references"`
	From        string       `json:"from"`
	To          []string     `json:"to"`
	Subject     string       `json:"subject"`
	Date        string       `json:"date"`
	Text        string       `json:"text"`
	HTML        string       `json:"html"`
	Attachments []Attachment `json:"attachments"`
}

type SendInput struct {
	AccountID   string       `json:"accountId"`
	To          []string     `json:"to"`
	Subject     string       `json:"subject"`
	Text        string       `json:"text"`
	HTML        string       `json:"html"`
	Attachments []Attachment `json:"attachments"`
	InReplyTo   string       `json:"inReplyTo"`
	References  string       `json:"references"`
}

func ParseMessage(raw []byte, accountID, id string) (Message, error) {
	if len(raw) > maxMessageBytes {
		return Message{}, &Error{Code: "MESSAGE_TOO_LARGE"}
	}
	parsed, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return Message{}, &Error{Code: "INVALID_MESSAGE"}
	}
	decoder := mime.WordDecoder{CharsetReader: decodeCharset}
	message := Message{
		ID: id, AccountID: accountID, Text: "", HTML: "", Attachments: []Attachment{},
	}
	message.Subject, _ = decoder.DecodeHeader(parsed.Header.Get("Subject"))
	if from := parsed.Header.Get("From"); from != "" {
		addresses, err := mail.ParseAddressList(from)
		if err != nil {
			return Message{}, &Error{Code: "INVALID_MESSAGE"}
		}
		message.MessageID = parsed.Header.Get("Message-ID")
		message.References = parsed.Header.Get("References")
		for index := range addresses {
			addresses[index].Name, _ = decoder.DecodeHeader(addresses[index].Name)
		}
		message.From = strings.Join(formatAddresses(addresses), ", ")
	}
	if to := parsed.Header.Get("To"); to != "" {
		addresses, err := mail.ParseAddressList(to)
		if err != nil {
			return Message{}, &Error{Code: "INVALID_MESSAGE"}
		}
		for index := range addresses {
			addresses[index].Name, _ = decoder.DecodeHeader(addresses[index].Name)
		}
		message.To = formatAddresses(addresses)
	}
	if date, err := mail.ParseDate(parsed.Header.Get("Date")); err == nil {
		message.Date = date.UTC().Format(time.RFC3339)
	}
	if err := decodeMIMEPart(parsed.Header, parsed.Body, &message, 0); err != nil {
		return Message{}, err
	}
	return message, nil
}

func formatAddresses(addresses []*mail.Address) []string {
	formatted := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if address.Name == "" {
			formatted = append(formatted, address.Address)
		} else {
			formatted = append(formatted, address.Name+" <"+address.Address+">")
		}
	}
	return formatted
}

func formatWireAddresses(addresses []*mail.Address) []string {
	formatted := make([]string, 0, len(addresses))
	for _, address := range addresses {
		formatted = append(formatted, address.String())
	}
	return formatted
}
func decodeMIMEPart(header mail.Header, body io.Reader, result *Message, depth int) error {
	if depth > 12 {
		return &Error{Code: "MIME_NESTING_TOO_DEEP"}
	}
	contentType := header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return &Error{Code: "INVALID_MIME"}
	}
	disposition, dispositionParams, _ := mime.ParseMediaType(header.Get("Content-Disposition"))
	isAttachment := disposition == "attachment" || dispositionParams["filename"] != "" || params["name"] != ""
	if strings.HasPrefix(mediaType, "multipart/") && !isAttachment {
		reader := multipart.NewReader(body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return &Error{Code: "INVALID_MIME"}
			}
			partHeader := mail.Header(part.Header)
			if err := decodeMIMEPart(partHeader, part, result, depth+1); err != nil {
				part.Close()
				return err
			}
			part.Close()
		}
		return nil
	}
	if isAttachment || (!strings.HasPrefix(mediaType, "text/") && mediaType != "message/rfc822") {
		if len(result.Attachments) >= maxAttachments {
			return &Error{Code: "TOO_MANY_ATTACHMENTS"}
		}
		data, err := readTransferEncoded(body, header.Get("Content-Transfer-Encoding"), maxAttachmentBytes)
		if err != nil {
			return err
		}
		name := dispositionParams["filename"]
		if name == "" {
			name = params["name"]
		}
		if name == "" {
			name = "attachment-" + strconv.Itoa(len(result.Attachments)+1)
		}
		decoder := mime.WordDecoder{CharsetReader: decodeCharset}
		name, _ = decoder.DecodeHeader(name)
		result.Attachments = append(result.Attachments, Attachment{
			Name: name, ContentType: mediaType, Data: data,
		})
		return nil
	}
	if mediaType != "text/plain" && mediaType != "text/html" {
		return nil
	}
	data, err := readTransferEncoded(body, header.Get("Content-Transfer-Encoding"), maxMessageBytes)
	if err != nil {
		return err
	}
	data, err = decodeCharsetBytes(data, params["charset"])
	if err != nil {
		return &Error{Code: "UNSUPPORTED_CHARSET"}
	}
	if mediaType == "text/html" {
		result.HTML = string(data)
	} else {
		result.Text = string(data)
	}
	return nil
}

func readTransferEncoded(reader io.Reader, encoding string, limit int) ([]byte, error) {
	var decoded io.Reader = reader
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "7bit", "8bit", "binary":
	case "base64":
		decoded = base64.NewDecoder(base64.StdEncoding, reader)
	case "quoted-printable":
		decoded = quotedprintable.NewReader(reader)
	default:
		return nil, &Error{Code: "UNSUPPORTED_TRANSFER_ENCODING"}
	}
	data, err := io.ReadAll(io.LimitReader(decoded, int64(limit)+1))
	if err != nil || len(data) > limit {
		return nil, &Error{Code: "MIME_PART_TOO_LARGE"}
	}
	return data, nil
}

func decodeCharset(charset string, reader io.Reader) (io.Reader, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxMessageBytes+1))
	if err != nil || len(data) > maxMessageBytes {
		return nil, fmt.Errorf("MIME_PART_TOO_LARGE")
	}
	decoded, err := decodeCharsetBytes(data, charset)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(decoded), nil
}

func decodeCharsetBytes(data []byte, charset string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "", "utf-8", "utf8", "us-ascii":
		if !utf8.Valid(data) {
			return nil, errors.New("invalid UTF-8")
		}
		return data, nil
	case "iso-8859-1", "latin1", "windows-1252":
		result := make([]rune, 0, len(data))
		for _, value := range data {
			if value >= 0x80 && value <= 0x9f && strings.EqualFold(charset, "windows-1252") {
				result = append(result, windows1252[value-0x80])
			} else {
				result = append(result, rune(value))
			}
		}
		return []byte(string(result)), nil
	default:
		return nil, errors.New("unsupported charset")
	}
}

var windows1252 = [32]rune{
	'\u20ac', '\u0081', '\u201a', '\u0192', '\u201e', '\u2026', '\u2020', '\u2021',
	'\u02c6', '\u2030', '\u0160', '\u2039', '\u0152', '\u008d', '\u017d', '\u008f',
	'\u0090', '\u2018', '\u2019', '\u201c', '\u201d', '\u2022', '\u2013', '\u2014',
	'\u02dc', '\u2122', '\u0161', '\u203a', '\u0153', '\u009d', '\u017e', '\u0178',
}

func BuildMessage(account model.Account, input SendInput) ([]byte, error) {
	if input.AccountID != account.ID || len(input.To) == 0 || len(input.To) > 100 ||
		len(input.Subject) > 998 || len(input.Text) > maxMessageBytes ||
		len(input.HTML) > maxMessageBytes || len(input.Attachments) > maxAttachments ||
		(len(input.Text) == 0 && len(input.HTML) == 0) {
		return nil, &Error{Code: "INVALID_MESSAGE"}
	}
	toAddresses := make([]*mail.Address, 0, len(input.To))
	for _, recipient := range input.To {
		address, err := mail.ParseAddress(strings.TrimSpace(recipient))
		if err != nil || address.Address == "" || strings.ContainsAny(address.Address, "\r\n") {
			return nil, &Error{Code: "INVALID_RECIPIENT"}
		}
		toAddresses = append(toAddresses, address)
	}
	var body []byte
	contentType := ""
	if len(input.Attachments) > 0 {
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		partHeader := make(textproto.MIMEHeader)
		if input.Text != "" && input.HTML != "" {
			alternativeType, alternative, err := buildAlternative(input.Text, input.HTML)
			if err != nil {
				return nil, err
			}
			partHeader.Set("Content-Type", alternativeType)
			part, err := writer.CreatePart(partHeader)
			if err != nil {
				return nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
			}
			if _, err := part.Write(alternative); err != nil {
				return nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
			}
		} else {
			partHeader.Set("Content-Type", "text/plain; charset=utf-8")
			if input.HTML != "" {
				partHeader.Set("Content-Type", "text/html; charset=utf-8")
			}
			part, err := writer.CreatePart(partHeader)
			if err != nil {
				return nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
			}
			if _, err := io.WriteString(part, input.Text+input.HTML); err != nil {
				return nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
			}
		}
		for _, attachment := range input.Attachments {
			if len(attachment.Data) > maxAttachmentBytes || attachment.Name == "" ||
				len(attachment.Name) > 255 || strings.ContainsAny(attachment.Name, "\r\n/\\") ||
				strings.ContainsAny(attachment.ContentType, "\r\n") {
				return nil, &Error{Code: "INVALID_ATTACHMENT"}
			}
			mediaType := attachment.ContentType
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}
			if _, _, err := mime.ParseMediaType(mediaType); err != nil {
				return nil, &Error{Code: "INVALID_ATTACHMENT"}
			}
			partHeader := make(textproto.MIMEHeader)
			partHeader.Set("Content-Type", mediaType)
			partHeader.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Name}))
			partHeader.Set("Content-Transfer-Encoding", "base64")
			part, err := writer.CreatePart(partHeader)
			if err != nil {
				return nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
			}
			encoded := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Data)))
			base64.StdEncoding.Encode(encoded, attachment.Data)
			for index := 0; index < len(encoded); index += 76 {
				end := index + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				if _, err := part.Write(encoded[index:end]); err != nil {
					return nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
				}
				if _, err := io.WriteString(part, "\r\n"); err != nil {
					return nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
				}
			}
		}
		if err := writer.Close(); err != nil {
			return nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
		}
		body = buffer.Bytes()
		contentType = mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": writer.Boundary()})
	} else if input.Text != "" && input.HTML != "" {
		var err error
		contentType, body, err = buildAlternative(input.Text, input.HTML)
		if err != nil {
			return nil, err
		}
	} else if input.HTML != "" {
		contentType, body = "text/html; charset=utf-8", []byte(input.HTML)
	} else {
		contentType, body = "text/plain; charset=utf-8", []byte(input.Text)
	}

	headers := make(textproto.MIMEHeader)
	headers.Set("From", (&mail.Address{Address: account.Address}).String())
	headers.Set("To", strings.Join(formatWireAddresses(toAddresses), ", "))
	headers.Set("Date", time.Now().UTC().Format(time.RFC1123Z))
	headers.Set("Message-ID", "<"+account.ID+"."+strconv.FormatInt(time.Now().UnixNano(), 10)+"@atlas-mail.local>")
	headers.Set("MIME-Version", "1.0")
	if strings.ContainsAny(input.Subject, "\r\n") || strings.ContainsAny(input.InReplyTo, "\r\n") ||
		strings.ContainsAny(input.References, "\r\n") {
		return nil, &Error{Code: "INVALID_MESSAGE"}
	}
	headers.Set("Subject", mime.QEncoding.Encode("UTF-8", input.Subject))
	if input.InReplyTo != "" {
		headers.Set("In-Reply-To", input.InReplyTo)
	}
	if input.References != "" {
		headers.Set("References", input.References)
	}
	var result bytes.Buffer
	for name, values := range headers {
		for _, value := range values {
			fmt.Fprintf(&result, "%s: %s\r\n", name, value)
		}
	}
	fmt.Fprintf(&result, "Content-Type: %s\r\n", contentType)
	if len(input.Attachments) == 0 && !(input.Text != "" && input.HTML != "") {
		result.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	}
	result.WriteString("\r\n")
	result.Write(body)
	if result.Len() > maxMessageBytes {
		return nil, &Error{Code: "MESSAGE_TOO_LARGE"}
	}
	return result.Bytes(), nil
}

func buildAlternative(text, html string) (string, []byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, entry := range []struct {
		mediaType string
		content   string
	}{{"text/plain", text}, {"text/html", html}} {
		partHeader := make(textproto.MIMEHeader)
		partHeader.Set("Content-Type", entry.mediaType+"; charset=utf-8")
		partHeader.Set("Content-Transfer-Encoding", "8bit")
		part, err := writer.CreatePart(partHeader)
		if err != nil {
			return "", nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
		}
		if _, err := io.WriteString(part, entry.content); err != nil {
			return "", nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
		}
	}
	if err := writer.Close(); err != nil {
		return "", nil, &Error{Code: "MESSAGE_BUILD_FAILED"}
	}
	return mime.FormatMediaType("multipart/alternative", map[string]string{"boundary": writer.Boundary()}),
		body.Bytes(), nil
}
