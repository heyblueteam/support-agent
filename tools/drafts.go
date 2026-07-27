package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/blue/support-agent/common"
)

// DraftInfo is a draft flattened for review before it is sent.
type DraftInfo struct {
	DraftID  string `json:"draft_id"`
	ThreadID string `json:"thread_id"`
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Body     string `json:"body,omitempty"`
}

// RunListDrafts lists drafts with the draft ids needed to send or delete them.
func RunListDrafts(args []string) error {
	fs := flag.NewFlagSet("list-drafts", flag.ExitOnError)
	limit := fs.Int64("limit", 0, "Max drafts (0 = all)")
	withBodies := fs.Bool("bodies", false, "Include full draft bodies")
	output := fs.String("output", "simple", "Output format: simple or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	drafts, err := client.ListDrafts(*limit)
	if err != nil {
		return err
	}
	if len(drafts) == 0 {
		fmt.Println("No drafts.")
		return nil
	}

	infos := make([]DraftInfo, 0, len(drafts))
	for _, d := range drafts {
		info := DraftInfo{DraftID: d.Id}

		msg := d.Message
		// List returns only ids and a stub message, so the body has to be
		// fetched per draft. Skip that round-trip unless bodies were asked for.
		if *withBodies || msg == nil || msg.Payload == nil {
			full, err := client.GetDraft(d.Id)
			if err != nil {
				fmt.Printf("Warning: %v\n", err)
				continue
			}
			msg = full.Message
		}
		if msg != nil {
			info.ThreadID = msg.ThreadId
			if msg.Payload != nil {
				headers := common.ExtractHeaders(msg)
				info.To = headers["to"]
				info.Subject = headers["subject"]
				if *withBodies {
					info.Body = common.ExtractMessageBody(msg)
				}
			}
		}
		infos = append(infos, info)
	}

	if *output == "json" {
		payload, err := json.MarshalIndent(infos, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}
		fmt.Println(string(payload))
		return nil
	}

	for _, d := range infos {
		fmt.Printf("%-20s %-40.40s %s\n", d.DraftID, d.To, d.Subject)
	}
	fmt.Printf("\n%d drafts\n", len(infos))
	return nil
}

// RunSendDraft sends existing drafts by id.
//
// Kept separate from reply-message on purpose: reply-message composes new text,
// whereas this sends words a human has already reviewed, unchanged, and clears
// the draft in the same operation.
func RunSendDraft(args []string) error {
	fs := flag.NewFlagSet("send-draft", flag.ExitOnError)
	ids := fs.String("draft-id", "", "Draft id to send (comma-separated for several)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *ids == "" {
		return fmt.Errorf("draft-id is required")
	}

	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	var failures int
	for _, id := range strings.Split(*ids, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		msg, err := client.SendDraft(id)
		if err != nil {
			failures++
			fmt.Printf("FAILED %s: %v\n", id, err)
			continue
		}
		fmt.Printf("sent %s -> message %s (thread %s)\n", id, msg.Id, msg.ThreadId)
	}

	if failures > 0 {
		return fmt.Errorf("%d draft(s) failed to send", failures)
	}
	return nil
}

// RunDeleteDraft discards drafts without sending them.
func RunDeleteDraft(args []string) error {
	fs := flag.NewFlagSet("delete-draft", flag.ExitOnError)
	ids := fs.String("draft-id", "", "Draft id to delete (comma-separated for several)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *ids == "" {
		return fmt.Errorf("draft-id is required")
	}

	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	var failures int
	for _, id := range strings.Split(*ids, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := client.DeleteDraft(id); err != nil {
			failures++
			fmt.Printf("FAILED %s: %v\n", id, err)
			continue
		}
		fmt.Printf("deleted %s\n", id)
	}

	if failures > 0 {
		return fmt.Errorf("%d draft(s) failed to delete", failures)
	}
	return nil
}
