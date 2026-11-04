package mail

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testAccountID = "11111111-1111-4111-8111-111111111111"

func TestReceiveIMAPOverVerifiedTLS(t *testing.T) {
	listener, roots := newMailTLSListener(t)
	defer listener.Close()
	rawMessage := []byte("From: sender@example.org\r\nSubject: Grüße\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHallo\r\n")
	serverErr := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()
		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)
		fmt.Fprint(writer, "* OK IMAP ready\r\n")
		writer.Flush()
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				serverErr <- err
				return
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				serverErr <- fmt.Errorf("invalid IMAP command")
				return
			}
			switch fields[1] {
			case "LOGIN":
				fmt.Fprintf(writer, "%s OK authenticated\r\n", fields[0])
			case "SELECT":
				fmt.Fprintf(writer, "* 1 EXISTS\r\n%s OK selected\r\n", fields[0])
			case "UID":
				if strings.Contains(line, "SEARCH") {
					fmt.Fprintf(writer, "* SEARCH 1\r\n%s OK searched\r\n", fields[0])
				} else {
					fmt.Fprintf(writer, "* 1 FETCH (UID 1 BODY[] {%d}\r\n", len(rawMessage))
					writer.Flush()
					writer.Write(rawMessage)
					fmt.Fprintf(writer, ")\r\n%s OK fetched\r\n", fields[0])
				}
			default:
				fmt.Fprintf(writer, "%s BAD unexpected command\r\n", fields[0])
			}
			if err := writer.Flush(); err != nil {
				serverErr <- err
				return
			}
		}
	}()
	profile := Profile{
		AccountID: testAccountID, Username: "person@example.org", AuthMechanism: "password",
		IMAP: Endpoint{Host: "localhost", Port: listener.Addr().(*net.TCPAddr).Port, TLSMode: "implicit", RootCAs: roots},
	}
	messages, err := ReceiveIMAP(context.Background(), profile, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Subject != "Grüße" || strings.TrimSpace(messages[0].Text) != "Hallo" {
		t.Fatalf("unexpected IMAP messages: %#v", messages)
	}
}

func TestIMAPRejectsUntrustedTLSCertificate(t *testing.T) {
	listener, _ := newMailTLSListener(t)
	defer listener.Close()
	go func() {
		connection, err := listener.Accept()
		if err == nil {
			_ = connection.(*tls.Conn).Handshake()
			connection.Close()
		}
	}()
	profile := Profile{
		AccountID: testAccountID, Username: "person@example.org", AuthMechanism: "password",
		IMAP: Endpoint{Host: "localhost", Port: listener.Addr().(*net.TCPAddr).Port, TLSMode: "implicit"},
	}
	_, err := ReceiveIMAP(context.Background(), profile, "secret")
	var mailErr *Error
	if !errors.As(err, &mailErr) || mailErr.Code != "TLS_CERTIFICATE_INVALID" {
		t.Fatalf("untrusted certificate was not rejected explicitly: %v", err)
	}
}

func TestSendSMTPOverTLSUsesSelectedSenderAndDoesNotRetry(t *testing.T) {
	listener, roots := newMailTLSListener(t)
	defer listener.Close()
	serverErr := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()
		writer := bufio.NewWriter(connection)
		reader := bufio.NewReader(connection)
		fmt.Fprint(writer, "220 mail.example.org ESMTP\r\n")
		writer.Flush()
		var data strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				serverErr <- err
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO "):
				fmt.Fprint(writer, "250-mail.example.org\r\n250-AUTH PLAIN\r\n250 OK\r\n")
			case strings.HasPrefix(line, "AUTH PLAIN "):
				payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(line, "AUTH PLAIN ")))
				if err != nil || string(payload) != "\x00person@example.org\x00secret" {
					serverErr <- fmt.Errorf("unexpected SMTP credentials")
					return
				}
				fmt.Fprint(writer, "235 authenticated\r\n")
			case strings.HasPrefix(line, "MAIL FROM:"):
				if !strings.Contains(line, "selected@example.org") {
					serverErr <- fmt.Errorf("wrong sender: %s", line)
					return
				}
				fmt.Fprint(writer, "250 sender accepted\r\n")
			case strings.HasPrefix(line, "RCPT TO:"):
				fmt.Fprint(writer, "250 recipient accepted\r\n")
			case strings.TrimSpace(line) == "DATA":
				fmt.Fprint(writer, "354 send data\r\n")
				writer.Flush()
				for {
					part, err := reader.ReadString('\n')
					if err != nil {
						serverErr <- err
						return
					}
					if part == ".\r\n" {
						break
					}
					data.WriteString(part)
				}
				if !strings.Contains(data.String(), "From: <selected@example.org>") ||
					!strings.Contains(data.String(), "Subject:") {
					serverErr <- fmt.Errorf("message headers missing selected sender: %q", data.String())
					return
				}
				fmt.Fprint(writer, "250 message accepted\r\n")
			case strings.TrimSpace(line) == "QUIT":
				fmt.Fprint(writer, "221 bye\r\n")
				writer.Flush()
				serverErr <- nil
				return
			default:
				fmt.Fprintf(writer, "250 ok\r\n")
			}
			if err := writer.Flush(); err != nil {
				serverErr <- err
				return
			}
		}
	}()
	profile := Profile{
		AccountID: testAccountID, Username: "person@example.org", AuthMechanism: "password",
		SMTP: Endpoint{Host: "localhost", Port: listener.Addr().(*net.TCPAddr).Port, TLSMode: "implicit", RootCAs: roots},
	}
	err := SendSMTP(context.Background(), profile, "secret", "selected@example.org", SendInput{
		AccountID: testAccountID, To: []string{"friend@example.org"}, Subject: "hello", Text: "body",
	})
	if err != nil {
		select {
		case serverError := <-serverErr:
			t.Fatalf("SMTP client failed: %v; server: %v", err, serverError)
		default:
			t.Fatal(err)
		}
	}
	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("SMTP session did not finish")
	}
}

func TestSendSMTPReportsUncertainStatusAfterDataWithoutRetry(t *testing.T) {
	listener, roots := newMailTLSListener(t)
	defer listener.Close()
	var sessions atomic.Int32
	var deliveries atomic.Int32
	serverErr := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()
		sessions.Add(1)
		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)
		fmt.Fprint(writer, "220 mail.example.org ESMTP\r\n")
		if err := writer.Flush(); err != nil {
			serverErr <- err
			return
		}
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				serverErr <- err
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO "):
				fmt.Fprint(writer, "250-mail.example.org\r\n250-AUTH PLAIN\r\n250 OK\r\n")
			case strings.HasPrefix(line, "AUTH PLAIN "):
				fmt.Fprint(writer, "235 authenticated\r\n")
			case strings.HasPrefix(line, "MAIL FROM:"):
				fmt.Fprint(writer, "250 sender accepted\r\n")
			case strings.HasPrefix(line, "RCPT TO:"):
				fmt.Fprint(writer, "250 recipient accepted\r\n")
			case strings.TrimSpace(line) == "DATA":
				fmt.Fprint(writer, "354 send data\r\n")
				if err := writer.Flush(); err != nil {
					serverErr <- err
					return
				}
				for {
					part, err := reader.ReadString('\n')
					if err != nil {
						serverErr <- err
						return
					}
					if part == ".\r\n" {
						deliveries.Add(1)
						serverErr <- nil
						return
					}
				}
			default:
				fmt.Fprintf(writer, "250 ok\r\n")
			}
			if err := writer.Flush(); err != nil {
				serverErr <- err
				return
			}
		}
	}()
	profile := Profile{
		AccountID: testAccountID, Username: "person@example.org", AuthMechanism: "password",
		SMTP: Endpoint{Host: "localhost", Port: listener.Addr().(*net.TCPAddr).Port, TLSMode: "implicit", RootCAs: roots},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := SendSMTP(ctx, profile, "secret", "selected@example.org", SendInput{
		AccountID: testAccountID, To: []string{"friend@example.org"}, Subject: "hello", Text: "body",
	})
	var mailErr *Error
	if !errors.As(err, &mailErr) || mailErr.Code != "SEND_STATUS_UNCERTAIN" {
		t.Fatalf("missing final SMTP response should be uncertain, got %v", err)
	}
	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("SMTP server did not observe the message body")
	}
	if sessions.Load() != 1 || deliveries.Load() != 1 {
		t.Fatalf("uncertain send was retried: sessions=%d deliveries=%d", sessions.Load(), deliveries.Load())
	}
}

func newMailTLSListener(t *testing.T) (net.Listener, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKey}),
	)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		t.Fatal(err)
	}
	certificate.Leaf = leaf
	roots := x509.NewCertPool()
	roots.AddCert(leaf)
	listener, err := tls.Listen("tcp", net.JoinHostPort("127.0.0.1", "0"), &tls.Config{
		Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		t.Fatal(err)
	}
	if listener.Addr().(*net.TCPAddr).Port < 1 {
		t.Fatal("test server did not allocate a port")
	}
	return listener, roots
}
