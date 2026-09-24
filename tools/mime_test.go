package tools

import (
	"encoding/base64"
	"mime"
	"net/mail"
	"strings"
	"testing"
)

func TestMIMEMessageBuildEncodesNonASCIISubject(t *testing.T) {
	const subject = "Re: Automation Issue – TestLauncher Recurring Record Not Being Generated"

	raw, err := (&MIMEMessage{
		From:    "me",
		To:      "customer@example.com",
		Subject: subject,
		Body:    "Hello",
	}).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	decoded, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode raw message: %v", err)
	}
	message, err := mail.ReadMessage(strings.NewReader(string(decoded)))
	if err != nil {
		t.Fatalf("parse raw message: %v", err)
	}

	encodedSubject := message.Header.Get("Subject")
	if !strings.Contains(encodedSubject, "=?UTF-8?") {
		t.Fatalf("Subject header = %q, want an RFC 2047 encoded-word", encodedSubject)
	}
	got, err := new(mime.WordDecoder).DecodeHeader(encodedSubject)
	if err != nil {
		t.Fatalf("decode Subject header: %v", err)
	}
	if got != subject {
		t.Fatalf("decoded Subject = %q, want %q", got, subject)
	}
}

func TestMIMEMessageBuildLeavesASCIISubjectReadable(t *testing.T) {
	raw, err := (&MIMEMessage{
		From:    "me",
		To:      "customer@example.com",
		Subject: "Re: Plain ASCII subject",
		Body:    "Hello",
	}).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	decoded, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode raw message: %v", err)
	}
	if !strings.Contains(string(decoded), "Subject: Re: Plain ASCII subject\r\n") {
		t.Fatalf("raw message does not contain the expected ASCII Subject header")
	}
}
