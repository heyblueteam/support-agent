package tools

import (
	"flag"
	"fmt"
	"strings"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

// RunDraftReply creates a Gmail draft reply to a message — it does NOT send.
//
// This is the unattended-triage counterpart to RunReplyMessage: identical
// recipient resolution, subject/threading (In-Reply-To/References), and
// attachment handling, but the result lands in Gmail's Drafts in-thread for a
// human to review and send. Drafting is the only outbound write the autonomous
// triage job is allowed to perform — the Send button stays the human gate.
func RunDraftReply(args []string) error {
	fs := flag.NewFlagSet("draft-reply", flag.ExitOnError)

	messageID := fs.String("message-id", "", "Message ID to draft a reply to (required)")
	body := fs.String("body", "", "Reply body text (required)")
	threadID := fs.String("thread-id", "", "Thread ID (optional, will be fetched if not provided)")
	toOverride := fs.String("to", "", "Override recipient — defaults to the original sender. Use when the thread was started by a no-reply bot and you want to route the reply to the real customer.")
	cc := fs.String("cc", "", "Cc recipients (comma-separated)")
	bcc := fs.String("bcc", "", "Bcc recipients (comma-separated)")
	var attachments StringSliceFlag
	fs.Var(&attachments, "attach", "Path to file to attach (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *messageID == "" || *body == "" {
		fmt.Println("Error: message-id and body are required")
		fmt.Println("\nUsage: draft-reply --message-id MESSAGE_ID --body \"Reply text\" [--to EMAIL] [--cc EMAIL] [--bcc EMAIL] [--attach PATH ...] [--thread-id THREAD_ID]")
		return fmt.Errorf("message-id and body are required")
	}

	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	originalMsg, err := client.GetMessage(*messageID)
	if err != nil {
		return fmt.Errorf("failed to get original message: %v", err)
	}

	headers := common.ExtractHeaders(originalMsg)

	tid := *threadID
	if tid == "" {
		tid = originalMsg.ThreadId
	}

	to := defaultReplyRecipient(client, headers, tid)
	if *toOverride != "" {
		to = *toOverride
	} else if to != headers["from"] {
		fmt.Printf("Note: original message is from an internal address (%s); routing reply to %s (first external participant in thread). Use --to to override.\n",
			headers["from"], to)
	}

	subject := headers["subject"]
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	originalMessageID := headers["message-id"]
	references := headers["references"]
	if references != "" {
		references += " " + originalMessageID
	} else {
		references = originalMessageID
	}

	msg := &MIMEMessage{
		From:        "me",
		To:          to,
		Cc:          *cc,
		Bcc:         *bcc,
		Subject:     subject,
		Body:        *body,
		InReplyTo:   originalMessageID,
		References:  references,
		Attachments: attachments,
	}

	encoded, err := msg.Build()
	if err != nil {
		return fmt.Errorf("failed to build message: %v", err)
	}

	draft, err := client.CreateDraft(&gmail.Message{
		Raw:      encoded,
		ThreadId: tid,
	})
	if err != nil {
		return fmt.Errorf("failed to create draft: %v", err)
	}

	fmt.Printf("Draft created (NOT sent).\n")
	fmt.Printf("Draft ID: %s\n", draft.Id)
	if draft.Message != nil {
		fmt.Printf("Thread ID: %s\n", draft.Message.ThreadId)
	}
	fmt.Printf("To: %s\n", to)
	if *cc != "" {
		fmt.Printf("Cc: %s\n", *cc)
	}
	if *bcc != "" {
		fmt.Printf("Bcc: %s\n", *bcc)
	}
	fmt.Printf("Subject: %s\n", subject)
	if len(attachments) > 0 {
		fmt.Printf("Attachments: %d\n", len(attachments))
	}
	fmt.Printf("\nReview and send from Gmail Drafts.\n")

	return nil
}
