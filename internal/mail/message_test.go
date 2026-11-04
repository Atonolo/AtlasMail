package mail

import (
	"bytes"
	"encoding/base64"
	"net/mail"
	"strings"
	"testing"

	"github.com/Atonolo/AtlasMail/internal/model"
)

func TestParseMIMEMessagePreservesUnicodeAlternativesAndAttachments(t *testing.T) {
	raw := strings.Join([]string{
		"From: =?ISO-8859-1?Q?M=FCller?= <sender@example.org>",
		"To: =?UTF-8?Q?Empf=C3=A4nger?= <person@example.org>",
		"Subject: =?UTF-8?Q?Gr=C3=BC=C3=9Fe?=",
		"Message-ID: <message-1@example.org>",
		"References: <older@example.org>",
		"MIME-Version: 1.0",
		"Content-Type: multipart/mixed; boundary=outer",
		"",
		"--outer",
		"Content-Type: multipart/alternative; boundary=inner",
		"",
		"--inner",
		"Content-Type: text/plain; charset=ISO-8859-1",
		"Content-Transfer-Encoding: quoted-printable",
		"",
		"Gr=FC=DFe aus K=F6ln",
		"--inner",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<p>Grüße <img src=\"https://tracker.example/pixel\"></p>",
		"--inner--",
		"--outer",
		"Content-Type: application/octet-stream; name*=utf-8''bericht%20final.bin",
		"Content-Disposition: attachment; filename*=utf-8''bericht%20final.bin",
		"Content-Transfer-Encoding: base64",
		"",
		base64.StdEncoding.EncodeToString([]byte{0, 1, 2, 255}),
		"--outer--",
		"",
	}, "\r\n")
	message, err := ParseMessage([]byte(raw), "11111111-1111-4111-8111-111111111111", "55")
	if err != nil {
		t.Fatal(err)
	}
	if message.Subject != "Grüße" || !strings.Contains(message.From, "Müller") ||
		!strings.Contains(strings.Join(message.To, ","), "Empfänger") ||
		message.Text != "Grüße aus Köln" || !strings.Contains(message.HTML, "tracker.example") {
		t.Fatalf("MIME text or headers were changed: %#v", message)
	}
	if message.MessageID != "<message-1@example.org>" || message.References != "<older@example.org>" {
		t.Fatalf("reply headers were lost: %#v", message)
	}
	if len(message.Attachments) != 1 || message.Attachments[0].Name != "bericht final.bin" ||
		!bytes.Equal(message.Attachments[0].Data, []byte{0, 1, 2, 255}) {
		t.Fatalf("MIME attachment was changed: %#v", message.Attachments)
	}
}

func TestBuildMessageProducesSafeReplyAndMultipartAttachments(t *testing.T) {
	account := model.Account{
		ID: "11111111-1111-4111-8111-111111111111", Label: "Private",
		Address: "me@example.org", Protocol: "imap-smtp",
	}
	input := SendInput{
		AccountID: account.ID, To: []string{"Empfänger <person@example.org>"},
		Subject: "Grüße", Text: "Hallo aus Köln", HTML: "<p>Hallo aus Köln</p>",
		InReplyTo: "<previous@example.org>", References: "<root@example.org> <previous@example.org>",
		Attachments: []Attachment{{Name: "übung.txt", ContentType: "text/plain", Data: []byte("Anhang")}},
	}
	raw, err := BuildMessage(account, input)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseMessage(raw, account.ID, "sent")
	if err != nil {
		t.Fatal(err)
	}
	from, err := mail.ParseAddress(parsed.From)
	if err != nil || from.Address != account.Address {
		t.Fatalf("message sender is not the selected account: %q, %v", parsed.From, err)
	}
	if parsed.Subject != input.Subject || parsed.Text != input.Text || parsed.HTML != input.HTML ||
		parsed.MessageID == "" || parsed.References != input.References ||
		len(parsed.Attachments) != 1 || parsed.Attachments[0].Name != "übung.txt" ||
		string(parsed.Attachments[0].Data) != "Anhang" {
		t.Fatalf("outgoing MIME message did not round-trip: %#v", parsed)
	}
}

func TestBuildMessageRejectsHeaderInjectionAndInvalidRecipients(t *testing.T) {
	account := model.Account{
		ID: "11111111-1111-4111-8111-111111111111", Address: "me@example.org", Protocol: "imap-smtp",
	}
	for _, input := range []SendInput{
		{AccountID: account.ID, To: []string{"friend@example.org"}, Subject: "hello\r\nBcc: victim@example.org", Text: "body"},
		{AccountID: account.ID, To: []string{"friend@example.org\r\nBcc: victim@example.org"}, Text: "body"},
	} {
		if _, err := BuildMessage(account, input); err == nil {
			t.Fatalf("unsafe message was accepted: %#v", input)
		}
	}
}
