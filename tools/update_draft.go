package tools

import (
	"encoding/base64"
	"flag"
	"fmt"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

// RunUpdateDraft updates an existing Gmail draft message
func RunUpdateDraft(args []string) error {
	fs := flag.NewFlagSet("update-draft", flag.ExitOnError)

	// Define flags
	draftID := fs.String("draft-id", "", "Draft ID to update (required)")
	body := fs.String("body", "", "New reply body text (required)")
	subject := fs.String("subject", "", "Subject (optional, keeps original if not provided)")

	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate
	if *draftID == "" || *body == "" {
		fmt.Println("Error: draft-id and body are required")
		fmt.Println("\nUsage: update-draft --draft-id DRAFT_ID --body \"Updated reply text\" [--subject \"New Subject\"]")
		return fmt.Errorf("draft-id and body are required")
	}

	// Create client
	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	// Get existing draft to preserve headers
	existingDraft, err := client.GetDraft(*draftID)
	if err != nil {
		return fmt.Errorf("failed to get existing draft: %v", err)
	}

	// Extract headers from existing draft message
	headers := common.ExtractHeaders(existingDraft.Message)

	// Use existing headers but allow subject override
	from := "me"
	to := headers["to"]
	sub := headers["subject"]
	if *subject != "" {
		sub = *subject
	}
	originalMessageID := headers["in-reply-to"]
	references := headers["references"]
	threadID := existingDraft.Message.ThreadId

	// Create updated email message
	emailContent := fmt.Sprintf(
		"From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n",
		from, to, sub)

	// Add threading headers if they exist
	if originalMessageID != "" {
		emailContent += fmt.Sprintf("In-Reply-To: %s\r\n", originalMessageID)
	}
	if references != "" {
		emailContent += fmt.Sprintf("References: %s\r\n", references)
	}

	emailContent += "Content-Type: text/plain; charset=UTF-8\r\n\r\n" + *body

	// Encode message
	encodedMessage := base64.URLEncoding.EncodeToString([]byte(emailContent))

	// Create updated Gmail message
	message := &gmail.Message{
		Raw:      encodedMessage,
		ThreadId: threadID,
	}

	// Update draft
	updatedDraft, err := client.UpdateDraft(*draftID, message)
	if err != nil {
		return fmt.Errorf("failed to update draft: %v", err)
	}

	fmt.Printf("Draft updated successfully!\n")
	fmt.Printf("Draft ID: %s\n", updatedDraft.Id)
	fmt.Printf("Thread ID: %s\n", threadID)
	fmt.Printf("To: %s\n", to)
	fmt.Printf("Subject: %s\n", sub)
	fmt.Printf("\nThe updated draft is now saved in your Gmail drafts folder.\n")

	return nil
}