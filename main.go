package main

import (
	"fmt"
	"os"

	"github.com/blue/support-agent/tools"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	var err error

	switch command {
	// Read operations
	case "read-messages":
		err = tools.RunReadMessages(args)
	case "read-threads":
		err = tools.RunReadThreads(args)
	case "read-message-detail":
		err = tools.RunReadMessageDetail(args)
	case "search-messages":
		err = tools.RunSearchMessages(args)
		
	// Write operations
	case "reply-message":
		err = tools.RunReplyMessage(args)
	case "archive-message":
		err = tools.RunArchiveMessage(args)
	case "label-message":
		err = tools.RunLabelMessage(args)
		
	// Help
	case "help", "-h", "--help":
		printUsage()
		os.Exit(0)
		
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Gmail Support Agent - Command-line tools for Gmail integration")
	fmt.Println()
	fmt.Println("Usage: support-agent <command> [options]")
	fmt.Println()
	fmt.Println("Read Commands:")
	fmt.Println("  read-messages          List messages from inbox with filters")
	fmt.Println("    --unread            Show only unread messages")
	fmt.Println("    --from EMAIL        Filter by sender")
	fmt.Println("    --subject TEXT      Filter by subject")
	fmt.Println("    --label LABEL       Filter by label")
	fmt.Println("    --limit N           Max results (default: 10)")
	fmt.Println("    --output FORMAT     Output format: simple, detailed, json")
	fmt.Println()
	fmt.Println("  read-threads           Get full conversation thread")
	fmt.Println("    --thread-id ID      Thread ID (required)")
	fmt.Println("    --output FORMAT     Output format: simple, detailed, json")
	fmt.Println()
	fmt.Println("  read-message-detail    Get complete message with body")
	fmt.Println("    --message-id ID     Message ID (required)")
	fmt.Println("    --output FORMAT     Output format: simple, detailed, json")
	fmt.Println()
	fmt.Println("  search-messages        Search using Gmail query syntax")
	fmt.Println("    --query QUERY       Search query (required)")
	fmt.Println("    --limit N           Max results (default: 20)")
	fmt.Println("    --output FORMAT     Output format: simple, detailed, json")
	fmt.Println()
	fmt.Println("Write Commands:")
	fmt.Println("  reply-message          Send a reply to a message")
	fmt.Println("    --message-id ID     Original message ID (required)")
	fmt.Println("    --body TEXT         Reply text (required)")
	fmt.Println("    --thread-id ID      Thread ID (optional)")
	fmt.Println()
	fmt.Println("  archive-message        Archive messages or threads")
	fmt.Println("    --message-id ID     Message to archive")
	fmt.Println("    --thread-id ID      Thread to archive")
	fmt.Println()
	fmt.Println("  label-message          Add/remove labels")
	fmt.Println("    --message-id ID     Message to label")
	fmt.Println("    --thread-id ID      Thread to label")
	fmt.Println("    --add-label LABEL   Label(s) to add (comma-separated)")
	fmt.Println("    --remove-label LABEL Label(s) to remove")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  support-agent read-messages --unread --limit 5")
	fmt.Println("  support-agent search-messages --query \"from:customer@example.com\"")
	fmt.Println("  support-agent read-threads --thread-id THREAD_ID --output json")
	fmt.Println("  support-agent reply-message --message-id MSG_ID --body \"Thank you for contacting us\"")
	fmt.Println("  support-agent archive-message --thread-id THREAD_ID")
	fmt.Println()
	fmt.Println("Authentication:")
	fmt.Println("  On first run, you'll be prompted to authenticate with Gmail.")
	fmt.Println("  The token is saved in ~/.support-agent/token.json")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Create a .env file with:")
	fmt.Println("    GMAIL_CREDENTIALS_PATH=./gmail.json")
	fmt.Println("    TOKEN_DIR=~/.support-agent")
	fmt.Println("    USER_EMAIL=your-email@example.com")
}