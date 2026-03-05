package tools

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

// AttachmentInfo holds metadata and ID needed to download an attachment
type AttachmentInfo struct {
	Filename     string
	AttachmentID string
	MimeType     string
	Size         int64
}

// RunDownloadAttachment downloads attachments from a Gmail message
func RunDownloadAttachment(args []string) error {
	fs := flag.NewFlagSet("download-attachment", flag.ExitOnError)

	messageID := fs.String("message-id", "", "Message ID to download attachments from (required)")
	filename := fs.String("filename", "", "Specific attachment filename to download (downloads all if omitted)")
	outputDir := fs.String("output-dir", ".", "Directory to save attachments (default: current directory)")
	listOnly := fs.Bool("list", false, "List attachments without downloading")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *messageID == "" {
		fmt.Println("Error: message-id is required")
		fmt.Println("\nUsage: download-attachment --message-id MESSAGE_ID [--filename NAME] [--output-dir DIR] [--list]")
		return fmt.Errorf("message-id is required")
	}

	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	msg, err := client.GetMessage(*messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %v", err)
	}

	attachments := extractAttachmentInfos(msg.Payload)
	if len(attachments) == 0 {
		fmt.Println("No attachments found in this message.")
		return nil
	}

	// List mode
	if *listOnly {
		fmt.Printf("Attachments (%d):\n", len(attachments))
		for i, a := range attachments {
			fmt.Printf("  %d. %s (%s, %d bytes)\n", i+1, a.Filename, a.MimeType, a.Size)
		}
		return nil
	}

	// Filter to specific filename if requested
	toDownload := attachments
	if *filename != "" {
		toDownload = nil
		for _, a := range attachments {
			if strings.EqualFold(a.Filename, *filename) {
				toDownload = append(toDownload, a)
				break
			}
		}
		if len(toDownload) == 0 {
			return fmt.Errorf("attachment %q not found in message", *filename)
		}
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Download each attachment
	for _, a := range toDownload {
		if err := downloadAttachment(client, *messageID, a, *outputDir); err != nil {
			return fmt.Errorf("failed to download %s: %v", a.Filename, err)
		}
	}

	return nil
}

func downloadAttachment(client *common.GmailClient, messageID string, a AttachmentInfo, outputDir string) error {
	body, err := client.GetAttachment(messageID, a.AttachmentID)
	if err != nil {
		return err
	}

	data, err := base64.URLEncoding.DecodeString(body.Data)
	if err != nil {
		return fmt.Errorf("failed to decode attachment data: %v", err)
	}

	outPath := filepath.Join(outputDir, a.Filename)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	fmt.Printf("Downloaded: %s -> %s (%d bytes)\n", a.Filename, outPath, len(data))
	return nil
}

// extractAttachmentInfos recursively finds all attachments in message parts
func extractAttachmentInfos(part *gmail.MessagePart) []AttachmentInfo {
	var attachments []AttachmentInfo

	if part == nil {
		return attachments
	}

	if part.Filename != "" && part.Body != nil && part.Body.AttachmentId != "" {
		attachments = append(attachments, AttachmentInfo{
			Filename:     part.Filename,
			AttachmentID: part.Body.AttachmentId,
			MimeType:     part.MimeType,
			Size:         part.Body.Size,
		})
	}

	for _, p := range part.Parts {
		attachments = append(attachments, extractAttachmentInfos(p)...)
	}

	return attachments
}
