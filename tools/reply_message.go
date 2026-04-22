package tools

import (
	"flag"
	"fmt"
	"strings"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

// RunReplyMessage sends a reply to a message
func RunReplyMessage(args []string) error {
	fs := flag.NewFlagSet("reply-message", flag.ExitOnError)

	messageID := fs.String("message-id", "", "Message ID to reply to (required)")
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
		fmt.Println("\nUsage: reply-message --message-id MESSAGE_ID --body \"Reply text\" [--to EMAIL] [--cc EMAIL] [--bcc EMAIL] [--attach PATH ...] [--thread-id THREAD_ID]")
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

	sentMsg, err := client.SendMessage(&gmail.Message{
		Raw:      encoded,
		ThreadId: tid,
	})
	if err != nil {
		return fmt.Errorf("failed to send reply: %v", err)
	}

	fmt.Printf("Reply sent successfully!\n")
	fmt.Printf("Message ID: %s\n", sentMsg.Id)
	fmt.Printf("Thread ID: %s\n", sentMsg.ThreadId)
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

	return nil
}

// defaultReplyRecipient resolves the default To: for a reply.
//
// Preference order:
//  1. Reply-To header on the original message (standard RFC convention).
//  2. From header on the original message, if external.
//  3. First external From in the thread — so a reply to an internal
//     handoff message (e.g. a teammate forwarded a support ticket back into
//     the thread) still routes back to the customer.
//
// This avoids the footgun where Gmail's "reply-all logic" would otherwise
// bounce the reply back to the last internal sender and the customer
// never receives it.
func defaultReplyRecipient(client *common.GmailClient, headers map[string]string, threadID string) string {
	if rt := headers["reply-to"]; rt != "" && !common.IsInternalAddress(rt) {
		return rt
	}
	if from := headers["from"]; from != "" && !common.IsInternalAddress(from) {
		return from
	}

	thread, err := client.GetThread(threadID)
	if err != nil || thread == nil {
		return headers["from"]
	}
	for _, msg := range thread.Messages {
		h := common.ExtractHeaders(msg)
		if from := h["from"]; from != "" && !common.IsInternalAddress(from) {
			return from
		}
	}
	return headers["from"]
}
