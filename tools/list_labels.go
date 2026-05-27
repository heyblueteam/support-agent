package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

// RunListLabels lists all labels in the mailbox.
func RunListLabels(args []string) error {
	fs := flag.NewFlagSet("list-labels", flag.ExitOnError)

	userOnly := fs.Bool("user-only", false, "Only show user-created labels (hide system labels)")
	output := fs.String("output", "simple", "Output format: simple, detailed, or json")

	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	labels, err := client.ListLabels()
	if err != nil {
		return err
	}

	if *userOnly {
		filtered := labels[:0]
		for _, l := range labels {
			if l.Type == "user" {
				filtered = append(filtered, l)
			}
		}
		labels = filtered
	}

	// Stable sort: user labels first (alphabetical), then system labels.
	sort.SliceStable(labels, func(i, j int) bool {
		if labels[i].Type != labels[j].Type {
			return labels[i].Type == "user"
		}
		return labels[i].Name < labels[j].Name
	})

	switch *output {
	case "json":
		jsonData, err := json.MarshalIndent(labels, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal labels: %v", err)
		}
		fmt.Println(string(jsonData))

	case "detailed":
		fmt.Printf("Found %d labels:\n\n", len(labels))
		for _, l := range labels {
			printDetailedLabel(l)
		}

	default: // simple
		fmt.Printf("Found %d labels:\n\n", len(labels))
		for _, l := range labels {
			fmt.Printf("  [%s] %s (%s)\n", l.Type, l.Name, l.Id)
		}
	}

	return nil
}

func printDetailedLabel(l *gmail.Label) {
	fmt.Printf("- %s\n", l.Name)
	fmt.Printf("    ID:       %s\n", l.Id)
	fmt.Printf("    Type:     %s\n", l.Type)
	if l.MessagesTotal > 0 || l.MessagesUnread > 0 {
		fmt.Printf("    Messages: %d total, %d unread\n", l.MessagesTotal, l.MessagesUnread)
	}
	if l.ThreadsTotal > 0 || l.ThreadsUnread > 0 {
		fmt.Printf("    Threads:  %d total, %d unread\n", l.ThreadsTotal, l.ThreadsUnread)
	}
	fmt.Println()
}
