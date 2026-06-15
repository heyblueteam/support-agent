package common

import (
	"encoding/base64"
	"fmt"
	"html"
	"regexp"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// GmailClient wraps the Gmail service with helper methods
type GmailClient struct {
	Service *gmail.Service
	UserID  string
}

// NewGmailClient creates a new Gmail client
func NewGmailClient() (*GmailClient, error) {
	service, err := GetAuthenticatedClient()
	if err != nil {
		return nil, err
	}

	return &GmailClient{
		Service: service,
		UserID:  "me",
	}, nil
}

// ListMessages lists messages with optional query
func (c *GmailClient) ListMessages(query string, maxResults int64) ([]*gmail.Message, error) {
	call := c.Service.Users.Messages.List(c.UserID)
	
	if query != "" {
		call.Q(query)
	}
	
	if maxResults > 0 {
		call.MaxResults(maxResults)
	}

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve messages: %v", err)
	}

	return response.Messages, nil
}

// GetMessage retrieves a full message by ID
func (c *GmailClient) GetMessage(messageID string) (*gmail.Message, error) {
	msg, err := c.Service.Users.Messages.Get(c.UserID, messageID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve message: %v", err)
	}
	return msg, nil
}

// GetThread retrieves a full thread by ID
func (c *GmailClient) GetThread(threadID string) (*gmail.Thread, error) {
	thread, err := c.Service.Users.Threads.Get(c.UserID, threadID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve thread: %v", err)
	}
	return thread, nil
}

// SendMessage sends an email message
func (c *GmailClient) SendMessage(message *gmail.Message) (*gmail.Message, error) {
	msg, err := c.Service.Users.Messages.Send(c.UserID, message).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to send message: %v", err)
	}
	return msg, nil
}

// CreateDraft creates a Gmail draft — it does NOT send. The draft lands in the
// mailbox's Drafts, threaded via message.ThreadId, for a human to review and
// send. This is the only outbound-write primitive the unattended triage job is
// permitted to use; the Send button stays the human gate.
func (c *GmailClient) CreateDraft(message *gmail.Message) (*gmail.Draft, error) {
	draft, err := c.Service.Users.Drafts.Create(c.UserID, &gmail.Draft{Message: message}).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create draft: %v", err)
	}
	return draft, nil
}

// ModifyMessage modifies labels on a message
func (c *GmailClient) ModifyMessage(messageID string, addLabels, removeLabels []string) (*gmail.Message, error) {
	modReq := &gmail.ModifyMessageRequest{
		AddLabelIds:    addLabels,
		RemoveLabelIds: removeLabels,
	}
	
	msg, err := c.Service.Users.Messages.Modify(c.UserID, messageID, modReq).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to modify message: %v", err)
	}
	return msg, nil
}

// GetAttachment retrieves an attachment by message ID and attachment ID
func (c *GmailClient) GetAttachment(messageID, attachmentID string) (*gmail.MessagePartBody, error) {
	attachment, err := c.Service.Users.Messages.Attachments.Get(c.UserID, messageID, attachmentID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve attachment: %v", err)
	}
	return attachment, nil
}

// ModifyThread modifies labels on all messages in a thread
func (c *GmailClient) ModifyThread(threadID string, addLabels, removeLabels []string) (*gmail.Thread, error) {
	modReq := &gmail.ModifyThreadRequest{
		AddLabelIds:    addLabels,
		RemoveLabelIds: removeLabels,
	}
	
	thread, err := c.Service.Users.Threads.Modify(c.UserID, threadID, modReq).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to modify thread: %v", err)
	}
	return thread, nil
}

// ExtractMessageBody extracts the message body, preferring text/plain and
// falling back to text/html (tags stripped) when no plain-text part exists.
// Many automated senders (e.g. the Blue feedback form) ship HTML-only email,
// so a plain-text-only extractor returns an empty body for them.
func ExtractMessageBody(msg *gmail.Message) string {
	if body := extractBody(msg.Payload, "text/plain"); body != "" {
		return body
	}
	if html := extractBody(msg.Payload, "text/html"); html != "" {
		return htmlToText(html)
	}
	return ""
}

// extractBody recursively extracts the decoded body of the first part matching
// mimeType.
func extractBody(part *gmail.MessagePart, mimeType string) string {
	if part.Body != nil && part.Body.Data != "" && part.MimeType == mimeType {
		if data, err := decodeBase64URL(part.Body.Data); err == nil {
			return string(data)
		}
	}

	for _, p := range part.Parts {
		if body := extractBody(p, mimeType); body != "" {
			return body
		}
	}

	return ""
}

// decodeBase64URL decodes Gmail body data, tolerating missing padding.
func decodeBase64URL(s string) ([]byte, error) {
	if data, err := base64.URLEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}

// htmlToText reduces an HTML body to readable plain text: it drops script/style
// blocks, turns block-level tags into newlines, strips remaining tags, and
// unescapes HTML entities. Good enough for reading support email, not a full
// HTML renderer.
func htmlToText(s string) string {
	reStyle := regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	s = reStyle.ReplaceAllString(s, "")

	reBreak := regexp.MustCompile(`(?i)<(br|/p|/div|/tr|/li|/h[1-6])\s*/?>`)
	s = reBreak.ReplaceAllString(s, "\n")

	reTag := regexp.MustCompile(`(?s)<[^>]+>`)
	s = reTag.ReplaceAllString(s, "")

	s = html.UnescapeString(s)

	reBlankLines := regexp.MustCompile(`\n[ \t]*\n[ \t]*(\n[ \t]*)+`)
	s = reBlankLines.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}

// InternalDomain is the email domain considered internal to Blue.
// Addresses on this domain are treated as support/staff, not customers.
const InternalDomain = "blue.cc"

// ExtractHeaders extracts common headers from a message
func ExtractHeaders(msg *gmail.Message) map[string]string {
	headers := make(map[string]string)

	for _, header := range msg.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "from", "to", "cc", "bcc", "reply-to", "delivered-to",
			"subject", "date", "message-id", "in-reply-to", "references":
			headers[strings.ToLower(header.Name)] = header.Value
		}
	}

	return headers
}

// IsInternalAddress reports whether an RFC 5322 address (e.g. "Name <x@blue.cc>")
// belongs to the internal Blue domain.
func IsInternalAddress(addr string) bool {
	return strings.Contains(strings.ToLower(addr), "@"+InternalDomain)
}

// GetLabelNames returns human-readable label names
func GetLabelNames(labelIDs []string) []string {
	names := make([]string, len(labelIDs))
	for i, id := range labelIDs {
		names[i] = id
	}
	return names
}

// ListLabels returns all labels (system + user) in the mailbox.
func (c *GmailClient) ListLabels() ([]*gmail.Label, error) {
	resp, err := c.Service.Users.Labels.List(c.UserID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list labels: %v", err)
	}
	return resp.Labels, nil
}

// CreateLabel creates a new user label with sensible default visibility.
func (c *GmailClient) CreateLabel(name string) (*gmail.Label, error) {
	label := &gmail.Label{
		Name:                  name,
		LabelListVisibility:   "labelShow",
		MessageListVisibility: "show",
	}
	created, err := c.Service.Users.Labels.Create(c.UserID, label).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create label %q: %v", name, err)
	}
	return created, nil
}

// systemLabelIDs is the closed set of label names whose Gmail ID equals the
// name itself. Anything else is a user label and must be resolved via
// labels.list to get the opaque Label_NNN id required by the modify API.
var systemLabelIDs = map[string]bool{
	"INBOX":                 true,
	"SENT":                  true,
	"DRAFT":                 true,
	"SPAM":                  true,
	"TRASH":                 true,
	"UNREAD":                true,
	"STARRED":               true,
	"IMPORTANT":             true,
	"CHAT":                  true,
	"CATEGORY_PERSONAL":     true,
	"CATEGORY_SOCIAL":       true,
	"CATEGORY_PROMOTIONS":   true,
	"CATEGORY_UPDATES":      true,
	"CATEGORY_FORUMS":       true,
	"CATEGORY_RESERVATIONS": true,
	"CATEGORY_PURCHASES":    true,
}

// ResolveLabelNames translates a list of label names or IDs into Gmail label
// IDs suitable for the modify API. System labels pass through as-is. User
// labels are looked up by case-insensitive exact name match. Inputs that
// already look like a user label ID ("Label_…") pass through unchanged.
// Unknown user labels error unless createIfMissing is true, in which case
// they are created on the fly.
func (c *GmailClient) ResolveLabelNames(names []string, createIfMissing bool) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	// Lazy fetch of the label list; only needed for non-system inputs.
	var labels []*gmail.Label
	loadLabels := func() error {
		if labels != nil {
			return nil
		}
		ls, err := c.ListLabels()
		if err != nil {
			return err
		}
		labels = ls
		return nil
	}

	resolved := make([]string, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}

		// Pass-through for system labels and raw user label IDs.
		if systemLabelIDs[strings.ToUpper(name)] {
			resolved = append(resolved, strings.ToUpper(name))
			continue
		}
		if strings.HasPrefix(name, "Label_") {
			resolved = append(resolved, name)
			continue
		}

		if err := loadLabels(); err != nil {
			return nil, err
		}

		var matchID string
		for _, l := range labels {
			if strings.EqualFold(l.Name, name) {
				matchID = l.Id
				break
			}
		}
		if matchID != "" {
			resolved = append(resolved, matchID)
			continue
		}

		if !createIfMissing {
			available := make([]string, 0, len(labels))
			for _, l := range labels {
				if l.Type == "user" {
					available = append(available, l.Name)
				}
			}
			return nil, fmt.Errorf("label %q not found (available user labels: %s) — pass --create-if-missing to create it", name, strings.Join(available, ", "))
		}

		created, err := c.CreateLabel(name)
		if err != nil {
			return nil, err
		}
		labels = append(labels, created)
		resolved = append(resolved, created.Id)
	}

	return resolved, nil
}