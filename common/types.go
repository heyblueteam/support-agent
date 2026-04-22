package common

import "time"

// MessageInfo represents simplified message data for output
type MessageInfo struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Cc        string    `json:"cc,omitempty"`
	Bcc       string    `json:"bcc,omitempty"`
	ReplyTo   string    `json:"reply_to,omitempty"`
	Subject   string    `json:"subject"`
	Date      string    `json:"date"`
	Snippet   string    `json:"snippet,omitempty"`
	Body      string    `json:"body,omitempty"`
	Labels    []string  `json:"labels"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// ThreadInfo represents simplified thread data for output
type ThreadInfo struct {
	ID           string        `json:"id"`
	Subject      string        `json:"subject"`
	Participants []string      `json:"participants"`
	MessageCount int           `json:"message_count"`
	LastMessage  time.Time     `json:"last_message"`
	Messages     []MessageInfo `json:"messages,omitempty"`
	Labels       []string      `json:"labels,omitempty"`
}

// OutputFormat represents the output format type
type OutputFormat string

const (
	OutputSimple   OutputFormat = "simple"
	OutputDetailed OutputFormat = "detailed"
	OutputJSON     OutputFormat = "json"
)

// Config holds application configuration
type Config struct {
	CredentialsPath string
	TokenDir        string
	UserEmail       string
}