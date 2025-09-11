package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/blue/support-agent/common"
)

// RunReadThreads gets full conversation threads
func RunReadThreads(args []string) error {
	fs := flag.NewFlagSet("read-threads", flag.ExitOnError)
	
	// Define flags
	threadID := fs.String("thread-id", "", "Thread ID to retrieve (required)")
	output := fs.String("output", "simple", "Output format: simple, detailed, or json")
	
	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate
	if *threadID == "" {
		fmt.Println("Error: thread-id is required")
		fmt.Println("\nUsage: read-threads --thread-id THREAD_ID [--output FORMAT]")
		return fmt.Errorf("thread-id is required")
	}

	// Create client
	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	// Get thread
	thread, err := client.GetThread(*threadID)
	if err != nil {
		return fmt.Errorf("failed to get thread: %v", err)
	}

	// Build thread info
	threadInfo := common.ThreadInfo{
		ID:           thread.Id,
		MessageCount: len(thread.Messages),
		Messages:     []common.MessageInfo{},
	}

	// Extract participants
	participantMap := make(map[string]bool)
	var lastMessageTime time.Time

	// Process messages
	for _, msg := range thread.Messages {
		headers := common.ExtractHeaders(msg)
		
		// Track participants
		if from := headers["from"]; from != "" {
			participantMap[from] = true
		}
		
		// Extract body for detailed/json output
		body := ""
		if *output == "detailed" || *output == "json" {
			body = common.ExtractMessageBody(msg)
		}

		// Parse date for sorting
		var msgTime time.Time
		if dateStr := headers["date"]; dateStr != "" {
			if parsed, err := time.Parse(time.RFC1123Z, dateStr); err == nil {
				msgTime = parsed
				if msgTime.After(lastMessageTime) {
					lastMessageTime = msgTime
				}
			}
		}

		msgInfo := common.MessageInfo{
			ID:        msg.Id,
			ThreadID:  msg.ThreadId,
			From:      headers["from"],
			To:        headers["to"],
			Subject:   headers["subject"],
			Date:      headers["date"],
			Snippet:   msg.Snippet,
			Body:      body,
			Labels:    common.GetLabelNames(msg.LabelIds),
			Timestamp: msgTime,
		}
		
		threadInfo.Messages = append(threadInfo.Messages, msgInfo)
		
		// Set thread subject from first message
		if threadInfo.Subject == "" && headers["subject"] != "" {
			threadInfo.Subject = headers["subject"]
		}
	}

	// Set participants
	for participant := range participantMap {
		threadInfo.Participants = append(threadInfo.Participants, participant)
	}
	
	threadInfo.LastMessage = lastMessageTime

	// Output results
	switch *output {
	case "json":
		jsonData, err := json.MarshalIndent(threadInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		
	case "detailed":
		fmt.Printf("=== Thread: %s ===\n", threadInfo.ID)
		fmt.Printf("Subject: %s\n", threadInfo.Subject)
		fmt.Printf("Participants: %s\n", strings.Join(threadInfo.Participants, ", "))
		fmt.Printf("Message Count: %d\n", threadInfo.MessageCount)
		if !threadInfo.LastMessage.IsZero() {
			fmt.Printf("Last Message: %s\n", threadInfo.LastMessage.Format(time.RFC3339))
		}
		fmt.Println("\n--- Messages ---")
		
		for i, msg := range threadInfo.Messages {
			fmt.Printf("\n[Message %d of %d]\n", i+1, len(threadInfo.Messages))
			fmt.Printf("ID: %s\n", msg.ID)
			fmt.Printf("From: %s\n", msg.From)
			fmt.Printf("To: %s\n", msg.To)
			fmt.Printf("Date: %s\n", msg.Date)
			fmt.Printf("Labels: %s\n", strings.Join(msg.Labels, ", "))
			if msg.Body != "" {
				fmt.Printf("\n%s\n", msg.Body)
			} else {
				fmt.Printf("\nSnippet: %s\n", msg.Snippet)
			}
			fmt.Println(strings.Repeat("-", 50))
		}
		
	default: // simple
		fmt.Printf("Thread ID: %s\n", threadInfo.ID)
		fmt.Printf("Subject: %s\n", threadInfo.Subject)
		fmt.Printf("Messages: %d\n", threadInfo.MessageCount)
		fmt.Printf("Participants: %s\n", strings.Join(threadInfo.Participants, ", "))
		fmt.Println("\nMessages:")
		for i, msg := range threadInfo.Messages {
			fmt.Printf("  %d. %s -> %s (%s)\n", 
				i+1,
				msg.From,
				msg.To,
				msg.Date)
		}
	}

	return nil
}