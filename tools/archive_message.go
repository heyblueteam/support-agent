package tools

import (
	"flag"
	"fmt"

	"github.com/blue/support-agent/common"
)

// RunArchiveMessage archives messages or threads
func RunArchiveMessage(args []string) error {
	fs := flag.NewFlagSet("archive-message", flag.ExitOnError)
	
	// Define flags
	messageID := fs.String("message-id", "", "Message ID to archive")
	threadID := fs.String("thread-id", "", "Thread ID to archive (archives all messages in thread)")
	
	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate - need either message or thread ID
	if *messageID == "" && *threadID == "" {
		fmt.Println("Error: either message-id or thread-id is required")
		fmt.Println("\nUsage:")
		fmt.Println("  archive-message --message-id MESSAGE_ID")
		fmt.Println("  archive-message --thread-id THREAD_ID")
		return fmt.Errorf("message-id or thread-id required")
	}

	// Create client
	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	// Archive by removing INBOX label
	removeLabels := []string{"INBOX"}
	
	if *threadID != "" {
		// Archive entire thread
		thread, err := client.ModifyThread(*threadID, nil, removeLabels)
		if err != nil {
			return fmt.Errorf("failed to archive thread: %v", err)
		}
		
		fmt.Printf("Thread archived successfully!\n")
		fmt.Printf("Thread ID: %s\n", thread.Id)
		fmt.Printf("Messages archived: %d\n", len(thread.Messages))
		
	} else {
		// Archive single message
		msg, err := client.ModifyMessage(*messageID, nil, removeLabels)
		if err != nil {
			return fmt.Errorf("failed to archive message: %v", err)
		}
		
		fmt.Printf("Message archived successfully!\n")
		fmt.Printf("Message ID: %s\n", msg.Id)
		fmt.Printf("Thread ID: %s\n", msg.ThreadId)
	}

	return nil
}