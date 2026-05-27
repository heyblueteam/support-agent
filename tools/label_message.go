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
	createIfMissing := fs.Bool("create-if-missing", false, "Create labels in --add-label that don't exist yet")
	
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

	// Prepare label lists (parse comma-separated user input)
	var addNames, removeNames []string

	if *addLabel != "" {
		for _, label := range strings.Split(*addLabel, ",") {
			if name := strings.TrimSpace(label); name != "" {
				addNames = append(addNames, name)
			}
		}
	}

	if *removeLabel != "" {
		for _, label := range strings.Split(*removeLabel, ",") {
			if name := strings.TrimSpace(label); name != "" {
				removeNames = append(removeNames, name)
			}
		}
	}

	// Resolve label names → Gmail label IDs. The modify API rejects names for
	// user labels; we look up the actual ID via labels.list. --create-if-missing
	// only applies to add-label (creating a label just to remove it is silly).
	addLabels, err := client.ResolveLabelNames(addNames, *createIfMissing)
	if err != nil {
		return fmt.Errorf("failed to resolve add-label: %v", err)
	}
	removeLabels, err := client.ResolveLabelNames(removeNames, false)
	if err != nil {
		return fmt.Errorf("failed to resolve remove-label: %v", err)
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
		if len(addNames) > 0 {
			fmt.Printf("Added labels: %s\n", strings.Join(addNames, ", "))
		}
		if len(removeNames) > 0 {
			fmt.Printf("Removed labels: %s\n", strings.Join(removeNames, ", "))
		}
		
	} else {
		// Modify single message
		msg, err := client.ModifyMessage(*messageID, addLabels, removeLabels)
		if err != nil {
			return fmt.Errorf("failed to modify message labels: %v", err)
		}
		
		fmt.Printf("Message labels updated successfully!\n")
		fmt.Printf("Message ID: %s\n", msg.Id)
		if len(addNames) > 0 {
			fmt.Printf("Added labels: %s\n", strings.Join(addNames, ", "))
		}
		if len(removeNames) > 0 {
			fmt.Printf("Removed labels: %s\n", strings.Join(removeNames, ", "))
		}
		fmt.Printf("Current labels: %s\n", strings.Join(common.GetLabelNames(msg.LabelIds), ", "))
	}

	return nil
}