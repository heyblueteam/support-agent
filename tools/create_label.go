package tools

import (
	"flag"
	"fmt"

	"github.com/blue/support-agent/common"
)

// RunCreateLabel creates a new Gmail label.
func RunCreateLabel(args []string) error {
	fs := flag.NewFlagSet("create-label", flag.ExitOnError)

	name := fs.String("name", "", "Label name (required, e.g. \"Follow-up\")")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		fmt.Println("Error: --name is required")
		fmt.Println("\nUsage: create-label --name \"Follow-up\"")
		return fmt.Errorf("name is required")
	}

	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	label, err := client.CreateLabel(*name)
	if err != nil {
		return err
	}

	fmt.Printf("Label created successfully!\n")
	fmt.Printf("Name: %s\n", label.Name)
	fmt.Printf("ID:   %s\n", label.Id)
	fmt.Printf("Type: %s\n", label.Type)

	return nil
}
