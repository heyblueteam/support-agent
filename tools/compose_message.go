package tools

import (
	"flag"
	"fmt"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

// RunComposeMessage starts a brand-new email thread (not a reply).
func RunComposeMessage(args []string) error {
	fs := flag.NewFlagSet("compose-message", flag.ExitOnError)

	to := fs.String("to", "", "Recipient email (required)")
	subject := fs.String("subject", "", "Subject line (required)")
	body := fs.String("body", "", "Message body (required)")
	cc := fs.String("cc", "", "Cc recipients (comma-separated)")
	bcc := fs.String("bcc", "", "Bcc recipients (comma-separated)")
	var attachments StringSliceFlag
	fs.Var(&attachments, "attach", "Path to file to attach (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *to == "" || *subject == "" || *body == "" {
		fmt.Println("Error: to, subject and body are required")
		fmt.Println("\nUsage: compose-message --to EMAIL --subject \"Subject\" --body \"Body\" [--cc EMAIL] [--bcc EMAIL] [--attach PATH ...]")
		return fmt.Errorf("to, subject and body are required")
	}

	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	msg := &MIMEMessage{
		From:        "me",
		To:          *to,
		Cc:          *cc,
		Bcc:         *bcc,
		Subject:     *subject,
		Body:        *body,
		Attachments: attachments,
	}

	encoded, err := msg.Build()
	if err != nil {
		return fmt.Errorf("failed to build message: %v", err)
	}

	sentMsg, err := client.SendMessage(&gmail.Message{Raw: encoded})
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	fmt.Printf("Message sent successfully!\n")
	fmt.Printf("Message ID: %s\n", sentMsg.Id)
	fmt.Printf("Thread ID: %s\n", sentMsg.ThreadId)
	fmt.Printf("To: %s\n", *to)
	fmt.Printf("Subject: %s\n", *subject)
	if len(attachments) > 0 {
		fmt.Printf("Attachments: %d\n", len(attachments))
	}

	return nil
}
