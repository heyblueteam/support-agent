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