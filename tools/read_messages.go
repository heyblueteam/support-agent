package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/blue/support-agent/common"
)

// RunReadMessages lists messages from Gmail
func RunReadMessages(args []string) error {
	fs := flag.NewFlagSet("read-messages", flag.ExitOnError)
	
	// Define flags
	unread := fs.Bool("unread", false, "Show only unread messages")
	from := fs.String("from", "", "Filter by sender email")
	subject := fs.String("subject", "", "Filter by subject (partial match)")
	label := fs.String("label", "", "Filter by label (e.g., INBOX, IMPORTANT)")
	limit := fs.Int64("limit", 10, "Maximum number of messages to return")
	output := fs.String("output", "simple", "Output format: simple, detailed, or json")
	
	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Build query
	var queryParts []string
	if *unread {
		queryParts = append(queryParts, "is:unread")
	}
	if *from != "" {
		queryParts = append(queryParts, fmt.Sprintf("from:%s", *from))
	}
	if *subject != "" {
		queryParts = append(queryParts, fmt.Sprintf("subject:%s", *subject))
	}
	if *label != "" {
		queryParts = append(queryParts, fmt.Sprintf("label:%s", *label))
	}
	
	// Default to inbox if no query specified
	if len(queryParts) == 0 {
		queryParts = append(queryParts, "in:inbox")
	}
	
	query := strings.Join(queryParts, " ")

	// Create client
	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	// List messages
	messages, err := client.ListMessages(query, *limit)
	if err != nil {
		return fmt.Errorf("failed to list messages: %v", err)
	}

	// Get full message details
	var messageInfos []common.MessageInfo
	for _, msg := range messages {
		fullMsg, err := client.GetMessage(msg.Id)
		if err != nil {
			fmt.Printf("Warning: failed to get message %s: %v\n", msg.Id, err)
			continue
		}

		headers := common.ExtractHeaders(fullMsg)
		body := ""
		if *output == "detailed" || *output == "json" {
			body = common.ExtractMessageBody(fullMsg)
		}

		info := common.MessageInfo{
			ID:       fullMsg.Id,
			ThreadID: fullMsg.ThreadId,
			From:     headers["from"],
			To:       headers["to"],
			Subject:  headers["subject"],
			Date:     headers["date"],
			Snippet:  fullMsg.Snippet,
			Body:     body,
			Labels:   common.GetLabelNames(fullMsg.LabelIds),
		}
		
		messageInfos = append(messageInfos, info)
	}

	// Output results
	switch *output {
	case "json":
		jsonData, err := json.MarshalIndent(messageInfos, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		
	case "detailed":
		for i, msg := range messageInfos {
			fmt.Printf("=== Message %d ===\n", i+1)
			fmt.Printf("ID: %s\n", msg.ID)
			fmt.Printf("Thread ID: %s\n", msg.ThreadID)
			fmt.Printf("From: %s\n", msg.From)
			fmt.Printf("To: %s\n", msg.To)
			fmt.Printf("Subject: %s\n", msg.Subject)
			fmt.Printf("Date: %s\n", msg.Date)
			fmt.Printf("Labels: %s\n", strings.Join(msg.Labels, ", "))
			if msg.Body != "" {
				fmt.Printf("\nBody:\n%s\n", msg.Body)
			} else {
				fmt.Printf("\nSnippet: %s\n", msg.Snippet)
			}
			fmt.Println()
		}
		
	default: // simple
		for _, msg := range messageInfos {
			fmt.Printf("%s | %s | %s | %s\n", 
				msg.ID, 
				msg.From,
				msg.Subject,
				msg.Date)
		}
	}

	return nil
}