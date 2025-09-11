package common

import (
	"encoding/base64"
	"fmt"
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

// ExtractMessageBody extracts plain text body from a message
func ExtractMessageBody(msg *gmail.Message) string {
	return extractBody(msg.Payload)
}

// extractBody recursively extracts text from message parts
func extractBody(part *gmail.MessagePart) string {
	// Check if this part has a body
	if part.Body != nil && part.Body.Data != "" {
		if part.MimeType == "text/plain" {
			data, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				return ""
			}
			return string(data)
		}
	}

	// Recursively check parts
	for _, p := range part.Parts {
		if body := extractBody(p); body != "" {
			return body
		}
	}

	return ""
}

// ExtractHeaders extracts common headers from a message
func ExtractHeaders(msg *gmail.Message) map[string]string {
	headers := make(map[string]string)
	
	for _, header := range msg.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "from", "to", "subject", "date", "message-id", "in-reply-to", "references":
			headers[strings.ToLower(header.Name)] = header.Value
		}
	}
	
	return headers
}

// GetLabelNames returns human-readable label names
func GetLabelNames(labelIDs []string) []string {
	names := make([]string, len(labelIDs))
	for i, id := range labelIDs {
		names[i] = id
	}
	return names
}