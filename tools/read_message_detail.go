package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

// RunReadMessageDetail gets complete message details
func RunReadMessageDetail(args []string) error {
	fs := flag.NewFlagSet("read-message-detail", flag.ExitOnError)
	
	// Define flags
	messageID := fs.String("message-id", "", "Message ID to retrieve (required)")
	output := fs.String("output", "detailed", "Output format: simple, detailed, or json")
	
	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate
	if *messageID == "" {
		fmt.Println("Error: message-id is required")
		fmt.Println("\nUsage: read-message-detail --message-id MESSAGE_ID [--output FORMAT]")
		return fmt.Errorf("message-id is required")
	}

	// Create client
	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	// Get message
	msg, err := client.GetMessage(*messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %v", err)
	}

	// Extract message info
	headers := common.ExtractHeaders(msg)
	body := common.ExtractMessageBody(msg)
	
	// Check for attachments
	var attachments []string
	if msg.Payload != nil {
		attachments = extractAttachments(msg.Payload)
	}

	msgInfo := common.MessageInfo{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		From:     headers["from"],
		To:       headers["to"],
		Subject:  headers["subject"],
		Date:     headers["date"],
		Snippet:  msg.Snippet,
		Body:     body,
		Labels:   common.GetLabelNames(msg.LabelIds),
	}

	// Output results
	switch *output {
	case "json":
		// Add attachments to JSON output
		type MessageWithAttachments struct {
			common.MessageInfo
			Attachments []string `json:"attachments,omitempty"`
		}
		
		msgWithAttach := MessageWithAttachments{
			MessageInfo: msgInfo,
			Attachments: attachments,
		}
		
		jsonData, err := json.MarshalIndent(msgWithAttach, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		
	case "simple":
		fmt.Printf("ID: %s\n", msgInfo.ID)
		fmt.Printf("From: %s\n", msgInfo.From)
		fmt.Printf("Subject: %s\n", msgInfo.Subject)
		fmt.Printf("Date: %s\n", msgInfo.Date)
		if len(attachments) > 0 {
			fmt.Printf("Attachments: %d\n", len(attachments))
		}
		
	default: // detailed
		fmt.Println("=== Message Details ===")
		fmt.Printf("ID: %s\n", msgInfo.ID)
		fmt.Printf("Thread ID: %s\n", msgInfo.ThreadID)
		fmt.Printf("From: %s\n", msgInfo.From)
		fmt.Printf("To: %s\n", msgInfo.To)
		fmt.Printf("Subject: %s\n", msgInfo.Subject)
		fmt.Printf("Date: %s\n", msgInfo.Date)
		fmt.Printf("Labels: %s\n", strings.Join(msgInfo.Labels, ", "))
		
		if len(attachments) > 0 {
			fmt.Printf("\nAttachments (%d):\n", len(attachments))
			for _, att := range attachments {
				fmt.Printf("  - %s\n", att)
			}
		}
		
		fmt.Printf("\nBody:\n%s\n", body)
	}

	return nil
}

// extractAttachments finds attachment filenames in message parts
func extractAttachments(part *gmail.MessagePart) []string {
	var attachments []string
	
	// Check if this part is an attachment
	if part.Filename != "" {
		attachments = append(attachments, part.Filename)
	}
	
	// Recursively check parts
	for _, p := range part.Parts {
		attachments = append(attachments, extractAttachments(p)...)
	}
	
	return attachments
}