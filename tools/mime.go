package tools

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// StringSliceFlag implements flag.Value for repeatable string flags
type StringSliceFlag []string

func (s *StringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *StringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// MIMEMessage builds an RFC 2822 message, optionally with attachments.
// If attachments is empty, produces a simple text/plain message.
// Otherwise produces a multipart/mixed message.
type MIMEMessage struct {
	From        string
	To          string
	Cc          string
	Bcc         string
	Subject     string
	Body        string
	InReplyTo   string
	References  string
	Attachments []string // file paths
}

const maxTotalAttachmentBytes = 25 * 1024 * 1024 // 25MB Gmail limit

// Build returns the base64url-encoded raw message ready for gmail.Message.Raw
func (m *MIMEMessage) Build() (string, error) {
	var totalSize int64
	for _, p := range m.Attachments {
		fi, err := os.Stat(p)
		if err != nil {
			return "", fmt.Errorf("attachment %s: %v", p, err)
		}
		totalSize += fi.Size()
	}
	if totalSize > maxTotalAttachmentBytes {
		return "", fmt.Errorf("total attachment size %d bytes exceeds 25MB Gmail limit", totalSize)
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "From: %s\r\n", m.From)
	fmt.Fprintf(&buf, "To: %s\r\n", m.To)
	if m.Cc != "" {
		fmt.Fprintf(&buf, "Cc: %s\r\n", m.Cc)
	}
	if m.Bcc != "" {
		fmt.Fprintf(&buf, "Bcc: %s\r\n", m.Bcc)
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", m.Subject)
	if m.InReplyTo != "" {
		fmt.Fprintf(&buf, "In-Reply-To: %s\r\n", m.InReplyTo)
	}
	if m.References != "" {
		fmt.Fprintf(&buf, "References: %s\r\n", m.References)
	}
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")

	if len(m.Attachments) == 0 {
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(m.Body)
	} else {
		boundary := "----=_SupportAgent_Boundary_7a3f9c2e"
		fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary)

		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: 7bit\r\n\r\n")
		buf.WriteString(m.Body)
		buf.WriteString("\r\n")

		for _, path := range m.Attachments {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read attachment %s: %v", path, err)
			}
			filename := filepath.Base(path)
			ctype := mime.TypeByExtension(filepath.Ext(filename))
			if ctype == "" {
				ctype = "application/octet-stream"
			}
			fmt.Fprintf(&buf, "--%s\r\n", boundary)
			fmt.Fprintf(&buf, "Content-Type: %s; name=\"%s\"\r\n", ctype, filename)
			fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n")
			fmt.Fprintf(&buf, "Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", filename)

			encoded := base64.StdEncoding.EncodeToString(data)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				buf.WriteString(encoded[i:end])
				buf.WriteString("\r\n")
			}
		}
		fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	}

	return base64.URLEncoding.EncodeToString([]byte(buf.String())), nil
}
