package tools

import (
	"encoding/base64"
	"flag"
	"fmt"
	"strings"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

// RunReplyMessage sends a reply to a message
func RunReplyMessage(args []string) error {
	fs := flag.NewFlagSet("reply-message", flag.ExitOnError)
	
	// Define flags
	messageID := fs.String("message-id", "", "Message ID to reply to (required)")
	body := fs.String("body", "", "Reply body text (required)")
	threadID := fs.String("thread-id", "", "Thread ID (optional, will be fetched if not provided)")
	
	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate
	if *messageID == "" || *body == "" {
		fmt.Println("Error: message-id and body are required")
		fmt.Println("\nUsage: reply-message --message-id MESSAGE_ID --body \"Reply text\" [--thread-id THREAD_ID]")
		return fmt.Errorf("message-id and body are required")
	}

	// Create client
	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	// Get original message for context
	originalMsg, err := client.GetMessage(*messageID)
	if err != nil {
		return fmt.Errorf("failed to get original message: %v", err)
	}

	// Extract headers from original
	headers := common.ExtractHeaders(originalMsg)
	
	// Use provided thread ID or get from original message
	tid := *threadID
	if tid == "" {
		tid = originalMsg.ThreadId
	}

	// Build reply headers
	from := "me" // Will use authenticated user's email
	to := headers["from"] // Reply to sender
	subject := headers["subject"]
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}
	
	// Get message ID for threading
	originalMessageID := headers["message-id"]
	references := headers["references"]
	if references != "" {
		references += " " + originalMessageID
	} else {
		references = originalMessageID
	}

	// Create email message
	emailContent := fmt.Sprintf(
		"From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"In-Reply-To: %s\r\n"+
		"References: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s",
		from, to, subject, originalMessageID, references, *body)

	// Encode message
	encodedMessage := base64.URLEncoding.EncodeToString([]byte(emailContent))
	
	// Create Gmail message
	message := &gmail.Message{
		Raw:      encodedMessage,
		ThreadId: tid,
	}

	// Send message
	sentMsg, err := client.SendMessage(message)
	if err != nil {
		return fmt.Errorf("failed to send reply: %v", err)
	}

	fmt.Printf("Reply sent successfully!\n")
	fmt.Printf("Message ID: %s\n", sentMsg.Id)
	fmt.Printf("Thread ID: %s\n", sentMsg.ThreadId)
	fmt.Printf("To: %s\n", to)
	fmt.Printf("Subject: %s\n", subject)

	return nil
}