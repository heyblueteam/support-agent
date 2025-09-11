package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/blue/support-agent/common"
)

// RunSearchMessages searches messages using Gmail query syntax
func RunSearchMessages(args []string) error {
	fs := flag.NewFlagSet("search-messages", flag.ExitOnError)
	
	// Define flags
	query := fs.String("query", "", "Gmail search query (required)")
	limit := fs.Int64("limit", 20, "Maximum number of results")
	output := fs.String("output", "simple", "Output format: simple, detailed, or json")
	
	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate
	if *query == "" {
		fmt.Println("Error: query is required")
		fmt.Println("\nUsage: search-messages --query \"search query\" [--limit N] [--output FORMAT]")
		fmt.Println("\nExample queries:")
		fmt.Println("  from:user@example.com")
		fmt.Println("  subject:\"important update\"")
		fmt.Println("  after:2024/1/1 before:2024/2/1")
		fmt.Println("  has:attachment")
		fmt.Println("  is:unread label:important")
		fmt.Println("  \"specific phrase\" in:anywhere")
		return fmt.Errorf("query is required")
	}

	// Create client
	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	// Search messages
	messages, err := client.ListMessages(*query, *limit)
	if err != nil {
		return fmt.Errorf("failed to search messages: %v", err)
	}

	if len(messages) == 0 {
		fmt.Println("No messages found matching query.")
		return nil
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
	fmt.Printf("Found %d messages matching query: %s\n\n", len(messageInfos), *query)
	
	switch *output {
	case "json":
		jsonData, err := json.MarshalIndent(messageInfos, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		
	case "detailed":
		for i, msg := range messageInfos {
			fmt.Printf("=== Result %d of %d ===\n", i+1, len(messageInfos))
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
		for i, msg := range messageInfos {
			fmt.Printf("%d. %s | %s | %s | %s\n", 
				i+1,
				msg.ID, 
				msg.From,
				msg.Subject,
				msg.Date)
		}
	}

	return nil
}