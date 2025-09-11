package tools

import (
	"flag"
	"fmt"
	"strings"

	"github.com/blue/support-agent/common"
)

// RunLabelMessage adds or removes labels from messages
func RunLabelMessage(args []string) error {
	fs := flag.NewFlagSet("label-message", flag.ExitOnError)
	
	// Define flags
	messageID := fs.String("message-id", "", "Message ID to label")
	threadID := fs.String("thread-id", "", "Thread ID to label (applies to all messages)")
	addLabel := fs.String("add-label", "", "Label to add (e.g., IMPORTANT, STARRED, or custom)")
	removeLabel := fs.String("remove-label", "", "Label to remove")
	
	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate
	if *messageID == "" && *threadID == "" {
		fmt.Println("Error: either message-id or thread-id is required")
		fmt.Println("\nUsage:")
		fmt.Println("  label-message --message-id MESSAGE_ID --add-label LABEL")
		fmt.Println("  label-message --thread-id THREAD_ID --remove-label LABEL")
		return fmt.Errorf("message-id or thread-id required")
	}

	if *addLabel == "" && *removeLabel == "" {
		fmt.Println("Error: at least one of --add-label or --remove-label is required")
		return fmt.Errorf("no label operation specified")
	}

	// Create client
	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	// Prepare label lists
	var addLabels, removeLabels []string
	
	if *addLabel != "" {
		// Split comma-separated labels
		labels := strings.Split(*addLabel, ",")
		for _, label := range labels {
			addLabels = append(addLabels, strings.TrimSpace(label))
		}
	}
	
	if *removeLabel != "" {
		// Split comma-separated labels
		labels := strings.Split(*removeLabel, ",")
		for _, label := range labels {
			removeLabels = append(removeLabels, strings.TrimSpace(label))
		}
	}

	// Apply labels
	if *threadID != "" {
		// Modify thread
		thread, err := client.ModifyThread(*threadID, addLabels, removeLabels)
		if err != nil {
			return fmt.Errorf("failed to modify thread labels: %v", err)
		}
		
		fmt.Printf("Thread labels updated successfully!\n")
		fmt.Printf("Thread ID: %s\n", thread.Id)
		if len(addLabels) > 0 {
			fmt.Printf("Added labels: %s\n", strings.Join(addLabels, ", "))
		}
		if len(removeLabels) > 0 {
			fmt.Printf("Removed labels: %s\n", strings.Join(removeLabels, ", "))
		}
		
	} else {
		// Modify single message
		msg, err := client.ModifyMessage(*messageID, addLabels, removeLabels)
		if err != nil {
			return fmt.Errorf("failed to modify message labels: %v", err)
		}
		
		fmt.Printf("Message labels updated successfully!\n")
		fmt.Printf("Message ID: %s\n", msg.Id)
		if len(addLabels) > 0 {
			fmt.Printf("Added labels: %s\n", strings.Join(addLabels, ", "))
		}
		if len(removeLabels) > 0 {
			fmt.Printf("Removed labels: %s\n", strings.Join(removeLabels, ", "))
		}
		fmt.Printf("Current labels: %s\n", strings.Join(common.GetLabelNames(msg.LabelIds), ", "))
	}

	return nil
}