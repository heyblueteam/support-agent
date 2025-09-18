package tools

import (
	"flag"
	"fmt"
	"strings"

	"github.com/blue/support-agent/common"
)

// RunDraftReply creates and displays a draft reply without sending it
func RunDraftReply(args []string) error {
	fs := flag.NewFlagSet("draft-reply", flag.ExitOnError)

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
		fmt.Println("\nUsage: draft-reply --message-id MESSAGE_ID --body \"Reply text\" [--thread-id THREAD_ID]")
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

	// Display the draft
	fmt.Println("=== DRAFT REPLY ===")
	fmt.Printf("From: %s\n", from)
	fmt.Printf("To: %s\n", to)
	fmt.Printf("Subject: %s\n", subject)
	fmt.Printf("Thread ID: %s\n", tid)
	fmt.Printf("In-Reply-To: %s\n", originalMessageID)
	fmt.Println()
	fmt.Println("--- Message Body ---")
	fmt.Println(*body)
	fmt.Println()
	fmt.Println("=== END DRAFT ===")
	fmt.Println()
	fmt.Printf("To send this reply, use: support-agent reply-message --message-id %s --body \"%s\"\n", *messageID, *body)

	return nil
}