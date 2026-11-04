package mail

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Atonolo/AtlasMail/internal/model"
)

var imapLiteral = regexp.MustCompile(`\{([0-9]+)\+?\}\r\n$`)
var imapLiteralMarker = regexp.MustCompile(`\{([0-9]+)\+?\}\r\n`)
var imapUID = regexp.MustCompile(`\bUID ([0-9]+)\b`)

type imapConnection struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	tag    int
}

func receiveIMAP(ctx context.Context, profile Profile, secret string) ([]Message, error) {
	connection, err := connectEndpoint(ctx, profile.IMAP)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	client := &imapConnection{conn: connection, reader: bufio.NewReader(connection), writer: bufio.NewWriter(connection)}
	if _, err := client.readResponse(); err != nil {
		return nil, &Error{Code: "IMAP_CONNECTION_FAILED"}
	}
	if profile.IMAP.TLSMode == "starttls" {
		if _, err := client.command("STARTTLS"); err != nil {
			return nil, &Error{Code: "TLS_REQUIRED"}
		}
		secured, err := upgradeTLS(ctx, connection, profile.IMAP.Host, profile.IMAP.RootCAs)
		if err != nil {
			return nil, err
		}
		connection = secured
		client.conn, client.reader, client.writer = connection, bufio.NewReader(connection), bufio.NewWriter(connection)
	}
	if profile.AuthMechanism == "oauth2" {
		if err := client.authenticateOAuth(profile.Username, secret); err != nil {
			return nil, err
		}
	} else if _, err := client.command("LOGIN " + imapQuote(profile.Username) + " " + imapQuote(secret)); err != nil {
		return nil, &Error{Code: "AUTH_FAILED"}
	}
	if _, err := client.command("SELECT INBOX"); err != nil {
		return nil, &Error{Code: "MAILBOX_OPEN_FAILED"}
	}
	responses, err := client.command("UID SEARCH ALL")
	if err != nil {
		return nil, &Error{Code: "MAIL_RECEIVE_FAILED"}
	}
	var ids []string
	for _, response := range responses {
		if !strings.HasPrefix(string(response), "* SEARCH ") {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(strings.TrimSuffix(string(response), "\r\n")))
		if len(fields) > 2 {
			ids = append(ids, fields[2:]...)
		}
	}
	if len(ids) > 50 {
		ids = ids[len(ids)-50:]
	}
	messages := make([]Message, 0, len(ids))
	totalBytes := 0
	for index := len(ids) - 1; index >= 0; index-- {
		uid := ids[index]
		if _, err := strconv.ParseUint(uid, 10, 32); err != nil {
			continue
		}
		responses, err := client.command("UID FETCH " + uid + " (UID BODY.PEEK[])")
		if err != nil {
			return nil, &Error{Code: "MAIL_RECEIVE_FAILED"}
		}
		for _, response := range responses {
			if !bytesContains(response, "FETCH") {
				continue
			}
			uidMatch := imapUID.FindSubmatch(response)
			if len(uidMatch) != 2 {
				continue
			}
			raw, ok := imapLiteralBytes(response)
			if !ok {
				continue
			}
			if totalBytes+len(raw) > maxInboxBytes {
				return messages, nil
			}
			totalBytes += len(raw)
			message, err := ParseMessage(raw, profile.AccountID, string(uidMatch[1]))
			if err != nil {
				continue
			}
			messages = append(messages, message)
		}
	}
	return messages, nil
}

func ReceiveIMAP(ctx context.Context, profile Profile, secret string) ([]Message, error) {
	return receiveIMAP(ctx, profile, secret)
}

func (c *imapConnection) authenticateOAuth(username, token string) error {
	tag, err := c.writeCommand("AUTHENTICATE XOAUTH2")
	if err != nil {
		return &Error{Code: "IMAP_AUTH_FAILED"}
	}
	response, err := c.readResponse()
	if err != nil || !strings.HasPrefix(string(response), "+") {
		return &Error{Code: "IMAP_AUTH_FAILED"}
	}
	payload := "user=" + username + "\x01auth=Bearer " + token + "\x01\x01"
	if _, err := c.writer.WriteString(base64.StdEncoding.EncodeToString([]byte(payload)) + "\r\n"); err != nil {
		return &Error{Code: "IMAP_AUTH_FAILED"}
	}
	if err := c.writer.Flush(); err != nil {
		return &Error{Code: "IMAP_AUTH_FAILED"}
	}
	for {
		response, err = c.readResponse()
		if err != nil {
			return &Error{Code: "IMAP_AUTH_FAILED"}
		}
		line := strings.TrimSpace(string(response))
		if strings.HasPrefix(line, tag+" OK ") || line == tag+" OK" {
			return nil
		}
		if strings.HasPrefix(line, tag+" ") {
			return &Error{Code: "AUTH_FAILED"}
		}
		if strings.HasPrefix(line, "+") {
			if _, err := c.writer.WriteString("\r\n"); err != nil {
				return &Error{Code: "IMAP_AUTH_FAILED"}
			}
			if err := c.writer.Flush(); err != nil {
				return &Error{Code: "IMAP_AUTH_FAILED"}
			}
		}
	}
}

func (c *imapConnection) command(command string) ([][]byte, error) {
	tag, err := c.writeCommand(command)
	if err != nil {
		return nil, err
	}
	var responses [][]byte
	for {
		response, err := c.readResponse()
		if err != nil {
			return responses, err
		}
		line := strings.TrimSpace(string(response))
		if strings.HasPrefix(line, tag+" ") {
			if !strings.HasPrefix(line, tag+" OK") {
				return responses, errors.New("IMAP command rejected")
			}
			return responses, nil
		}
		responses = append(responses, response)
	}
}

func (c *imapConnection) writeCommand(command string) (string, error) {
	c.tag++
	tag := fmt.Sprintf("A%04d", c.tag)
	if _, err := fmt.Fprintf(c.writer, "%s %s\r\n", tag, command); err != nil {
		return "", err
	}
	if err := c.writer.Flush(); err != nil {
		return "", err
	}
	return tag, nil
}

func (c *imapConnection) readResponse() ([]byte, error) {
	var response []byte
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return response, err
		}
		response = append(response, line...)
		match := imapLiteral.FindStringSubmatch(line)
		if len(match) == 0 {
			return response, nil
		}
		length, err := strconv.Atoi(match[1])
		if err != nil || length < 0 || length > maxMessageBytes {
			return nil, &Error{Code: "MESSAGE_TOO_LARGE"}
		}
		offset := len(response)
		response = append(response, make([]byte, length)...)
		if _, err := io.ReadFull(c.reader, response[offset:]); err != nil {
			return nil, err
		}
	}
}

func imapQuote(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}

func imapLiteralBytes(response []byte) ([]byte, bool) {
	match := imapLiteralMarker.FindIndex(response)
	if match == nil {
		return nil, false
	}
	var length int
	if _, err := fmt.Sscanf(string(response[match[0]:match[1]]), "{%d}", &length); err != nil {
		return nil, false
	}
	start := match[1]
	if length < 0 || start+length > len(response) {
		return nil, false
	}
	return response[start : start+length], true
}

func bytesContains(data []byte, value string) bool {
	return strings.Contains(string(data), value)
}

type xoauth2 struct {
	username string
	token    string
	secure   bool
}

func (a xoauth2) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !a.secure && !server.TLS {
		return "", nil, &Error{Code: "TLS_REQUIRED"}
	}
	return "XOAUTH2", []byte("user=" + a.username + "\x01auth=Bearer " + a.token + "\x01\x01"), nil
}

func (a xoauth2) Next(challenge []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	return nil, &Error{Code: "SMTP_AUTH_FAILED"}
}

type passwordAuth struct {
	username string
	password string
	secure   bool
}

func (a passwordAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !a.secure && !server.TLS {
		return "", nil, &Error{Code: "TLS_REQUIRED"}
	}
	return "PLAIN", []byte("\x00" + a.username + "\x00" + a.password), nil
}

func (passwordAuth) Next(_ []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	return nil, errors.New("unexpected SMTP authentication challenge")
}

func sendSMTP(ctx context.Context, profile Profile, secret string, accountAddress string, input SendInput) error {
	raw, err := BuildMessage(modelAccount(accountAddress, profile), input)
	if err != nil {
		return err
	}
	conn, err := connectEndpoint(ctx, profile.SMTP)
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, profile.SMTP.Host)
	if err != nil {
		return &Error{Code: "SMTP_CONNECTION_FAILED"}
	}
	defer client.Close()
	if err := client.Hello("atlas-mail.local"); err != nil {
		return &Error{Code: "SMTP_CONNECTION_FAILED"}
	}
	if profile.SMTP.TLSMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return &Error{Code: "TLS_REQUIRED"}
		}
		if err := client.StartTLS(&tls.Config{
			ServerName: profile.SMTP.Host, MinVersion: tls.VersionTLS12, RootCAs: profile.SMTP.RootCAs,
		}); err != nil {
			if isTLSCertificateError(err) || strings.Contains(strings.ToLower(err.Error()), "certificate") {
				return &Error{Code: "TLS_CERTIFICATE_INVALID"}
			}
			return &Error{Code: "TLS_REQUIRED"}
		}
	}
	var auth smtp.Auth
	_, hasTLS := client.TLSConnectionState()
	secure := profile.SMTP.TLSMode == "implicit" || hasTLS
	if profile.AuthMechanism == "oauth2" {
		auth = xoauth2{username: profile.Username, token: secret, secure: secure}
	} else {
		auth = passwordAuth{
			username: profile.Username, password: secret, secure: secure,
		}
	}
	if err := client.Auth(auth); err != nil {
		var mailErr *Error
		if errors.As(err, &mailErr) {
			return mailErr
		}
		return &Error{Code: "AUTH_FAILED"}
	}
	from, err := mail.ParseAddress(accountAddress)
	if err != nil {
		return &Error{Code: "INVALID_ACCOUNT"}
	}
	if err := client.Mail(from.Address); err != nil {
		return &Error{Code: "SMTP_SEND_FAILED"}
	}
	recipients, err := mail.ParseAddressList(strings.Join(input.To, ","))
	if err != nil {
		return &Error{Code: "INVALID_RECIPIENT"}
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient.Address); err != nil {
			return &Error{Code: "SMTP_SEND_FAILED"}
		}
	}
	dataWriter, err := client.Data()
	if err != nil {
		return &Error{Code: "SMTP_SEND_FAILED"}
	}
	if _, err := dataWriter.Write(raw); err != nil {
		_ = dataWriter.Close()
		return &Error{Code: "SEND_STATUS_UNCERTAIN"}
	}
	if err := dataWriter.Close(); err != nil {
		return &Error{Code: "SEND_STATUS_UNCERTAIN"}
	}
	_ = client.Quit()
	return nil
}

func SendSMTP(ctx context.Context, profile Profile, secret, accountAddress string, input SendInput) error {
	return sendSMTP(ctx, profile, secret, accountAddress, input)
}

func modelAccount(address string, profile Profile) model.Account {
	return model.Account{ID: profile.AccountID, Address: address, Label: address, Protocol: "imap-smtp"}
}

func connectEndpoint(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	raw, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(endpoint.Host, strconv.Itoa(endpoint.Port)))
	if err != nil {
		return nil, &Error{Code: "CONNECTION_FAILED"}
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = raw.SetDeadline(deadline)
	}
	if endpoint.TLSMode == "implicit" {
		secured, err := upgradeTLS(ctx, raw, endpoint.Host, endpoint.RootCAs)
		if err != nil {
			raw.Close()
			return nil, err
		}
		return secured, nil
	}
	return raw, nil
}

func upgradeTLS(ctx context.Context, connection net.Conn, serverName string, rootCAs *x509.CertPool) (net.Conn, error) {
	secured := tls.Client(connection, &tls.Config{
		ServerName: serverName, MinVersion: tls.VersionTLS12, RootCAs: rootCAs,
	})
	if err := secured.HandshakeContext(ctx); err != nil {
		if isTLSCertificateError(err) {
			return nil, &Error{Code: "TLS_CERTIFICATE_INVALID"}
		}
		return nil, &Error{Code: "TLS_REQUIRED"}
	}
	return secured, nil
}

func isTLSCertificateError(err error) bool {
	var verificationError *tls.CertificateVerificationError
	return errors.As(err, &verificationError)
}
