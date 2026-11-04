package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const jmapMailCapability = "urn:ietf:params:jmap:mail"

type jmapSession struct {
	APIURL          string            `json:"apiUrl"`
	UploadURL       string            `json:"uploadUrl"`
	DownloadURL     string            `json:"downloadUrl"`
	PrimaryAccounts map[string]string `json:"primaryAccounts"`
	Accounts        map[string]struct {
		AccountCapabilities map[string]json.RawMessage `json:"accountCapabilities"`
	} `json:"accounts"`
}

type jmapClient struct {
	httpClient *http.Client
	profile    Profile
	account    string
	secret     string
	session    jmapSession
	baseURL    *url.URL
}

func newJMAPClient(profile Profile, secret string) *jmapClient {
	return &jmapClient{
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		profile: profile,
		secret:  secret,
	}
}

func (c *jmapClient) connect(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.profile.JMAPURL, nil)
	if err != nil {
		return &Error{Code: "INVALID_JMAP_URL"}
	}
	c.authorize(request)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return &Error{Code: "JMAP_CONNECTION_FAILED"}
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return &Error{Code: "AUTH_FAILED"}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return &Error{Code: "JMAP_CONNECTION_FAILED"}
	}
	if err := decodeLimitedJSON(response.Body, maxMessageBytes, &c.session); err != nil {
		return &Error{Code: "INVALID_JMAP_SESSION"}
	}
	c.baseURL, err = url.Parse(c.session.APIURL)
	if err != nil || c.baseURL.Host == "" || c.baseURL.User != nil ||
		!sameOrigin(c.profile.JMAPURL, c.session.APIURL) {
		return &Error{Code: "INVALID_JMAP_SESSION"}
	}
	if c.session.PrimaryAccounts == nil || c.session.PrimaryAccounts[jmapMailCapability] == "" {
		return &Error{Code: "JMAP_MAIL_UNAVAILABLE"}
	}
	c.account = c.session.PrimaryAccounts[jmapMailCapability]
	return nil
}

func (c *jmapClient) authorize(request *http.Request) {
	if c.profile.AuthMechanism == "oauth2" {
		request.Header.Set("Authorization", "Bearer "+c.secret)
		return
	}
	request.SetBasicAuth(c.profile.Username, c.secret)
}

func (c *jmapClient) call(ctx context.Context, calls [][3]any) (map[string]json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{
		"using":       []string{jmapMailCapability, "urn:ietf:params:jmap:core"},
		"methodCalls": calls,
	})
	if err != nil {
		return nil, &Error{Code: "JMAP_REQUEST_FAILED"}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.session.APIURL, bytes.NewReader(body))
	if err != nil {
		return nil, &Error{Code: "JMAP_REQUEST_FAILED"}
	}
	request.Header.Set("Content-Type", "application/json")
	c.authorize(request)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, &Error{Code: "JMAP_CONNECTION_FAILED"}
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, &Error{Code: "AUTH_FAILED"}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &Error{Code: "JMAP_REQUEST_FAILED"}
	}
	var result struct {
		MethodResponses [][3]json.RawMessage `json:"methodResponses"`
	}
	if err := decodeLimitedJSON(response.Body, maxMessageBytes, &result); err != nil {
		return nil, &Error{Code: "INVALID_JMAP_RESPONSE"}
	}
	values := make(map[string]json.RawMessage, len(result.MethodResponses))
	for _, response := range result.MethodResponses {
		var method, callID string
		if err := json.Unmarshal(response[0], &method); err != nil || json.Unmarshal(response[2], &callID) != nil {
			continue
		}
		if strings.HasSuffix(method, "/error") {
			return nil, &Error{Code: "JMAP_REQUEST_FAILED"}
		}
		values[callID] = response[1]
	}
	return values, nil
}

func (c *jmapClient) mailboxes(ctx context.Context) (map[string]string, error) {
	values, err := c.call(ctx, [][3]any{{"Mailbox/get", map[string]any{
		"accountId": c.account, "properties": []string{"id", "name", "role"},
	}, "mailboxes"}})
	if err != nil {
		return nil, err
	}
	var result struct {
		List []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"list"`
	}
	if err := json.Unmarshal(values["mailboxes"], &result); err != nil {
		return nil, &Error{Code: "INVALID_JMAP_RESPONSE"}
	}
	mailboxes := make(map[string]string, len(result.List))
	for _, mailbox := range result.List {
		mailboxes[mailbox.Role] = mailbox.ID
	}
	return mailboxes, nil
}

func receiveJMAP(ctx context.Context, profile Profile, secret string) ([]Message, error) {
	client := newJMAPClient(profile, secret)
	if err := client.connect(ctx); err != nil {
		return nil, err
	}
	mailboxes, err := client.mailboxes(ctx)
	if err != nil {
		return nil, err
	}
	inboxID := mailboxes["inbox"]
	if inboxID == "" {
		return []Message{}, nil
	}
	values, err := client.call(ctx, [][3]any{
		{"Email/query", map[string]any{
			"accountId": client.account,
			"filter":    map[string]any{"inMailbox": inboxID},
			"sort":      []map[string]any{{"property": "receivedAt", "isAscending": false}},
			"limit":     50,
		}, "query"},
		{"Email/get", map[string]any{
			"accountId": client.account,
			"ids":       map[string]any{"resultOf": "query", "name": "Email/query", "path": "/ids"},
			"properties": []string{"id", "messageId", "from", "to", "subject", "receivedAt",
				"textBody", "htmlBody", "bodyValues", "attachments", "inReplyTo", "references"},
			"fetchTextBodyValues": true, "fetchHTMLBodyValues": true,
		}, "emails"},
	})
	if err != nil {
		return nil, err
	}
	var response struct {
		List []struct {
			ID        string   `json:"id"`
			MessageID []string `json:"messageId"`
			From      []struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"from"`
			To []struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"to"`
			Subject    string   `json:"subject"`
			Received   string   `json:"receivedAt"`
			InReplyTo  []string `json:"inReplyTo"`
			References []string `json:"references"`
			TextBody   []struct {
				PartID string `json:"partId"`
			} `json:"textBody"`
			HTMLBody []struct {
				PartID string `json:"partId"`
			} `json:"htmlBody"`
			BodyValues map[string]struct {
				Value string `json:"value"`
			} `json:"bodyValues"`
			Attachments []struct {
				BlobID string `json:"blobId"`
				Type   string `json:"type"`
				Name   string `json:"name"`
				Size   int    `json:"size"`
			} `json:"attachments"`
		} `json:"list"`
	}
	if err := json.Unmarshal(values["emails"], &response); err != nil {
		return nil, &Error{Code: "INVALID_JMAP_RESPONSE"}
	}
	messages := make([]Message, 0, len(response.List))
	totalBytes := 0
	for _, item := range response.List {
		message := Message{
			ID: item.ID, AccountID: profile.AccountID, Subject: item.Subject,
			Date: item.Received, Attachments: []Attachment{},
		}
		message.From = strings.Join(formatJMAPAddresses(item.From), ", ")
		if len(item.MessageID) > 0 {
			message.MessageID = item.MessageID[0]
		}
		message.References = strings.Join(item.References, " ")
		if len(item.InReplyTo) > 0 {
			message.References = strings.TrimSpace(message.References + " " + strings.Join(item.InReplyTo, " "))
		}
		message.To = formatJMAPAddresses(item.To)
		for _, part := range item.TextBody {
			message.Text += item.BodyValues[part.PartID].Value
		}
		for _, part := range item.HTMLBody {
			message.HTML += item.BodyValues[part.PartID].Value
		}
		for _, attachment := range item.Attachments {
			if len(message.Attachments) >= maxAttachments || attachment.Size < 0 ||
				attachment.Size > maxAttachmentBytes {
				continue
			}
			data, err := client.download(ctx, attachment.BlobID)
			if err != nil {
				return nil, err
			}
			if totalBytes+len(data) > maxInboxBytes {
				return nil, &Error{Code: "MAIL_RECEIVE_TOO_LARGE"}
			}
			totalBytes += len(data)
			message.Attachments = append(message.Attachments, Attachment{
				Name: attachment.Name, ContentType: attachment.Type, Data: data,
			})
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func ReceiveJMAP(ctx context.Context, profile Profile, secret string) ([]Message, error) {
	return receiveJMAP(ctx, profile, secret)
}

func formatJMAPAddresses(addresses []struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}) []string {
	result := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if parsed, err := mail.ParseAddress(address.Email); err == nil {
			parsed.Name = address.Name
			result = append(result, formatAddresses([]*mail.Address{parsed})[0])
		}
	}
	return result
}

func (c *jmapClient) download(ctx context.Context, blobID string) ([]byte, error) {
	target := strings.NewReplacer("{accountId}", url.PathEscape(c.account), "{blobId}", url.PathEscape(blobID),
		"{name}", "attachment", "{type}", "application%2Foctet-stream").Replace(c.session.DownloadURL)
	parsed, err := url.Parse(target)
	if err != nil || !sameOrigin(c.session.APIURL, target) {
		return nil, &Error{Code: "INVALID_JMAP_SESSION"}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, &Error{Code: "JMAP_ATTACHMENT_FAILED"}
	}
	c.authorize(request)
	response, err := c.httpClient.Do(request)
	if err != nil || response.StatusCode < 200 || response.StatusCode >= 300 {
		if response != nil {
			response.Body.Close()
		}
		return nil, &Error{Code: "JMAP_ATTACHMENT_FAILED"}
	}
	defer response.Body.Close()
	content, err := io.ReadAll(io.LimitReader(response.Body, maxAttachmentBytes+1))
	if err != nil || len(content) > maxAttachmentBytes {
		return nil, &Error{Code: "JMAP_ATTACHMENT_TOO_LARGE"}
	}
	return content, nil
}

func sendJMAP(ctx context.Context, profile Profile, secret, accountAddress string, input SendInput) error {
	client := newJMAPClient(profile, secret)
	if err := client.connect(ctx); err != nil {
		return err
	}
	mailboxes, err := client.mailboxes(ctx)
	if err != nil {
		return err
	}
	draftsID := mailboxes["drafts"]
	if draftsID == "" {
		return &Error{Code: "JMAP_DRAFTS_UNAVAILABLE"}
	}
	identities, err := client.call(ctx, [][3]any{{"Identity/get", map[string]any{
		"accountId": client.account, "properties": []string{"id", "email"},
	}, "identities"}})
	if err != nil {
		return err
	}
	var identityResult struct {
		List []struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"list"`
	}
	if err := json.Unmarshal(identities["identities"], &identityResult); err != nil {
		return &Error{Code: "INVALID_JMAP_RESPONSE"}
	}
	identityID := ""
	for _, identity := range identityResult.List {
		if strings.EqualFold(identity.Email, accountAddress) {
			identityID = identity.ID
			break
		}
	}
	if identityID == "" && len(identityResult.List) > 0 {
		identityID = identityResult.List[0].ID
	}
	if identityID == "" {
		return &Error{Code: "JMAP_IDENTITY_UNAVAILABLE"}
	}
	bodyStructure, bodyValues, err := client.buildBody(ctx, input)
	if err != nil {
		return err
	}
	to := make([]map[string]string, 0, len(input.To))
	for _, recipient := range input.To {
		address, err := mail.ParseAddress(recipient)
		if err != nil {
			return &Error{Code: "INVALID_RECIPIENT"}
		}
		to = append(to, map[string]string{"email": address.Address, "name": address.Name})
	}
	email := map[string]any{
		"mailboxIds": map[string]bool{draftsID: true}, "keywords": map[string]bool{"$draft": true},
		"from": []map[string]string{{"email": accountAddress}},
		"to":   to, "subject": input.Subject, "bodyStructure": bodyStructure, "bodyValues": bodyValues,
	}
	if input.InReplyTo != "" {
		email["inReplyTo"] = []string{input.InReplyTo}
	}
	if input.References != "" {
		email["references"] = strings.Fields(input.References)
	}
	created, err := client.call(ctx, [][3]any{{"Email/set", map[string]any{
		"accountId": client.account, "create": map[string]any{"draft": email},
	}, "email"}})
	if err != nil {
		return err
	}
	var createResponse struct {
		Created map[string]struct {
			ID string `json:"id"`
		} `json:"created"`
	}
	if err := json.Unmarshal(created["email"], &createResponse); err != nil {
		return &Error{Code: "INVALID_JMAP_RESPONSE"}
	}
	draftID := createResponse.Created["draft"].ID
	if draftID == "" {
		return &Error{Code: "JMAP_SEND_FAILED"}
	}
	submission, err := client.call(ctx, [][3]any{{"EmailSubmission/set", map[string]any{
		"accountId": client.account,
		"create": map[string]any{"submission": map[string]any{
			"emailId": draftID, "identityId": identityID,
		}},
	}, "submission"}})
	if err != nil {
		return &Error{Code: "SEND_STATUS_UNCERTAIN"}
	}
	var submissionResponse struct {
		Created map[string]json.RawMessage `json:"created"`
	}
	if err := json.Unmarshal(submission["submission"], &submissionResponse); err != nil {
		return &Error{Code: "SEND_STATUS_UNCERTAIN"}
	}
	if len(submissionResponse.Created) == 0 {
		return &Error{Code: "SEND_STATUS_UNCERTAIN"}
	}
	return nil
}

func SendJMAP(ctx context.Context, profile Profile, secret, accountAddress string, input SendInput) error {
	if input.AccountID != profile.AccountID || len(input.To) == 0 || (input.Text == "" && input.HTML == "") ||
		len(input.Attachments) > maxAttachments {
		return &Error{Code: "INVALID_MESSAGE"}
	}
	return sendJMAP(ctx, profile, secret, accountAddress, input)
}

func (c *jmapClient) buildBody(ctx context.Context, input SendInput) (map[string]any, map[string]any, error) {
	bodyValues := make(map[string]any)
	parts := make([]map[string]any, 0, 3+len(input.Attachments))
	partID := 0
	if input.Text != "" {
		partID++
		key := strconv.Itoa(partID)
		parts = append(parts, map[string]any{"type": "text/plain", "partId": key})
		bodyValues[key] = map[string]string{"value": input.Text}
	}
	if input.HTML != "" {
		partID++
		key := strconv.Itoa(partID)
		parts = append(parts, map[string]any{"type": "text/html", "partId": key})
		bodyValues[key] = map[string]string{"value": input.HTML}
	}
	if len(parts) == 2 {
		parts = []map[string]any{{"type": "multipart/alternative", "subParts": parts}}
	}
	for _, attachment := range input.Attachments {
		blobID, err := c.upload(ctx, attachment)
		if err != nil {
			return nil, nil, err
		}
		mediaType := attachment.ContentType
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		parts = append(parts, map[string]any{
			"type": mediaType, "blobId": blobID, "name": attachment.Name,
			"disposition": "attachment",
		})
	}
	if len(parts) == 0 {
		return nil, nil, &Error{Code: "INVALID_MESSAGE"}
	}
	if len(parts) == 1 {
		return parts[0], bodyValues, nil
	}
	return map[string]any{"type": "multipart/mixed", "subParts": parts}, bodyValues, nil
}

func (c *jmapClient) upload(ctx context.Context, attachment Attachment) (string, error) {
	target := strings.NewReplacer("{accountId}", url.PathEscape(c.account)).Replace(c.session.UploadURL)
	if !sameOrigin(c.session.APIURL, target) {
		return "", &Error{Code: "INVALID_JMAP_SESSION"}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(attachment.Data))
	if err != nil {
		return "", &Error{Code: "JMAP_ATTACHMENT_FAILED"}
	}
	mediaType := attachment.ContentType
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	request.Header.Set("Content-Type", mediaType)
	c.authorize(request)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", &Error{Code: "JMAP_ATTACHMENT_FAILED"}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", &Error{Code: "JMAP_ATTACHMENT_FAILED"}
	}
	var result struct {
		BlobID string `json:"blobId"`
	}
	if err := decodeLimitedJSON(response.Body, 64*1024, &result); err != nil || result.BlobID == "" {
		return "", &Error{Code: "JMAP_ATTACHMENT_FAILED"}
	}
	return result.BlobID, nil
}

func sameOrigin(left, right string) bool {
	a, errA := url.Parse(left)
	b, errB := url.Parse(right)
	return errA == nil && errB == nil && strings.EqualFold(a.Scheme, b.Scheme) &&
		strings.EqualFold(a.Host, b.Host)
}

func decodeLimitedJSON(reader io.Reader, limit int64, target any) error {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil || int64(len(data)) > limit {
		return errors.New("response too large")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("invalid JSON response")
	}
	return nil
}
