package mail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJMAPReceiveSendAndAttachments(t *testing.T) {
	var server *httptest.Server
	var sentEmail map[string]any
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "jmap-login" || password != "secret" {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/session" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"apiUrl":          server.URL + "/api",
				"uploadUrl":       server.URL + "/upload/{accountId}",
				"downloadUrl":     server.URL + "/download/{accountId}/{blobId}/{name}?type={type}",
				"primaryAccounts": map[string]string{jmapMailCapability: "jmap-account"},
				"accounts":        map[string]any{"jmap-account": map[string]any{}},
			})
		case r.URL.Path == "/download/jmap-account/old-blob/attachment":
			if r.URL.Query().Get("type") != "application/octet-stream" {
				t.Errorf("unexpected JMAP download type: %s", r.URL.RawQuery)
			}
			w.Write([]byte("received file"))
		case r.URL.Path == "/upload/jmap-account":
			content, _ := io.ReadAll(r.Body)
			if string(content) != "outgoing file" {
				t.Errorf("unexpected uploaded attachment: %q", content)
			}
			json.NewEncoder(w).Encode(map[string]any{"blobId": "new-blob"})
		case r.URL.Path == "/api" && r.Method == http.MethodPost:
			var request struct {
				MethodCalls [][3]json.RawMessage `json:"methodCalls"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode method calls: %v", err)
				return
			}
			responses := make([][]any, 0, len(request.MethodCalls))
			for _, call := range request.MethodCalls {
				var method, callID string
				json.Unmarshal(call[0], &method)
				json.Unmarshal(call[2], &callID)
				var response any
				switch method {
				case "Mailbox/get":
					response = map[string]any{"list": []map[string]string{
						{"id": "inbox", "name": "Inbox", "role": "inbox"},
						{"id": "drafts", "name": "Drafts", "role": "drafts"},
					}}
				case "Email/query":
					response = map[string]any{"ids": []string{"email-1"}}
				case "Email/get":
					response = map[string]any{"list": []map[string]any{{
						"id": "email-1", "messageId": []string{"<jmap-message@example.org>"},
						"from":    []map[string]string{{"name": "Müller", "email": "sender@example.org"}},
						"to":      []map[string]string{{"name": "", "email": "me@example.org"}},
						"subject": "Grüße", "receivedAt": "2026-11-04T10:00:00Z",
						"textBody": []map[string]string{{"partId": "1"}},
						"htmlBody": []map[string]string{{"partId": "2"}},
						"bodyValues": map[string]map[string]string{
							"1": {"value": "Hallo"}, "2": {"value": "<p>Hallo</p>"},
						},
						"attachments": []map[string]any{{
							"blobId": "old-blob", "type": "application/octet-stream",
							"name": "in.bin", "size": len("received file"),
						}},
					}}}
				case "Identity/get":
					response = map[string]any{"list": []map[string]string{{
						"id": "identity-1", "email": "me@example.org",
					}}}
				case "Email/set":
					var args struct {
						Create map[string]map[string]any `json:"create"`
					}
					if err := json.Unmarshal(call[1], &args); err != nil {
						t.Errorf("decode Email/set: %v", err)
						return
					}
					sentEmail = args.Create["draft"]
					response = map[string]any{"created": map[string]map[string]string{
						"draft": {"id": "draft-1"},
					}}
				case "EmailSubmission/set":
					response = map[string]any{"created": map[string]map[string]string{
						"submission": {"id": "submission-1"},
					}}
				default:
					t.Errorf("unexpected JMAP call: %s", method)
					response = map[string]any{}
				}
				responses = append(responses, []any{method, response, callID})
			}
			json.NewEncoder(w).Encode(map[string]any{"methodResponses": responses})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	profile := Profile{
		AccountID: testAccountID, Username: "jmap-login", AuthMechanism: "password",
		JMAPURL: server.URL + "/session",
	}
	messages, err := ReceiveJMAP(context.Background(), profile, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Subject != "Grüße" ||
		messages[0].From != "Müller <sender@example.org>" ||
		messages[0].MessageID != "<jmap-message@example.org>" ||
		messages[0].Text != "Hallo" || len(messages[0].Attachments) != 1 ||
		string(messages[0].Attachments[0].Data) != "received file" {
		t.Fatalf("unexpected JMAP message: %#v", messages)
	}

	err = SendJMAP(context.Background(), profile, "secret", "me@example.org", SendInput{
		AccountID: testAccountID, To: []string{"friend@example.org"},
		Subject: "Antwort", Text: "Hallo zurück",
		Attachments: []Attachment{{Name: "out.bin", ContentType: "application/octet-stream", Data: []byte("outgoing file")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	from, ok := sentEmail["from"].([]any)
	if !ok || len(from) != 1 {
		t.Fatalf("JMAP message omitted explicit account sender: %#v", sentEmail["from"])
	}
	fromAddress, _ := from[0].(map[string]any)
	if fromAddress["email"] != "me@example.org" {
		t.Fatalf("JMAP sender was not the selected account: %#v", from)
	}
	bodyStructure, _ := sentEmail["bodyStructure"].(map[string]any)
	encodedStructure, _ := json.Marshal(bodyStructure)
	if !strings.Contains(string(encodedStructure), "new-blob") {
		t.Fatalf("uploaded attachment missing from JMAP body structure: %s", encodedStructure)
	}
}

func TestJMAPRejectsCrossOriginAPIEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"apiUrl":          "http://attacker.example/api",
			"primaryAccounts": map[string]string{jmapMailCapability: "account"},
		})
	}))
	defer server.Close()
	client := newJMAPClient(Profile{JMAPURL: server.URL, AuthMechanism: "password"}, "secret")
	if err := client.connect(context.Background()); err == nil {
		t.Fatal("cross-origin API URL was accepted")
	}
}

func TestJMAPMessageDoesNotRetryUncertainSubmission(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session" {
			json.NewEncoder(w).Encode(map[string]any{
				"apiUrl":          serverURL(r) + "/api",
				"primaryAccounts": map[string]string{jmapMailCapability: "account"},
			})
			return
		}
		var request struct {
			MethodCalls [][3]json.RawMessage `json:"methodCalls"`
		}
		json.NewDecoder(r.Body).Decode(&request)
		var method string
		json.Unmarshal(request.MethodCalls[0][0], &method)
		response := map[string]any{}
		switch method {
		case "Mailbox/get":
			response = map[string]any{"list": []map[string]string{{"id": "drafts", "role": "drafts"}}}
		case "Identity/get":
			response = map[string]any{"list": []map[string]string{{"id": "identity", "email": "me@example.org"}}}
		case "Email/set":
			response = map[string]any{"created": map[string]map[string]string{"draft": {"id": "draft"}}}
		case "EmailSubmission/set":
			attempts++
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		var callID string
		json.Unmarshal(request.MethodCalls[0][2], &callID)
		json.NewEncoder(w).Encode(map[string]any{"methodResponses": [][]any{{method, response, callID}}})
	}))
	defer server.Close()
	profile := Profile{AccountID: testAccountID, Username: "me@example.org", AuthMechanism: "password"}
	profile.JMAPURL = server.URL + "/session"
	err := SendJMAP(context.Background(), profile, "secret", "me@example.org", SendInput{
		AccountID: testAccountID, To: []string{"friend@example.org"}, Text: "message",
	})
	var mailErr *Error
	if !errors.As(err, &mailErr) || mailErr.Code != "SEND_STATUS_UNCERTAIN" || attempts != 1 {
		t.Fatalf("uncertain submission was retried or misreported: error=%v attempts=%d", err, attempts)
	}
}

func serverURL(r *http.Request) string {
	return fmt.Sprintf("http://%s", r.Host)
}
