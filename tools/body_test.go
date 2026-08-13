package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBodyFromFilePreservesLineBreaks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reply.txt")
	want := "Hi,\n\nThis is the second paragraph.\n"
	if err := os.WriteFile(path, []byte(want), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveBody("", path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("resolveBody() = %q, want %q", got, want)
	}
}

func TestResolveBodyRejectsLiteralEscapedLineBreak(t *testing.T) {
	if _, err := resolveBody(`Hi,\n\nThis must not be sent`, ""); err == nil {
		t.Fatal("resolveBody() accepted literal escaped line breaks")
	}
}

func TestResolveBodyRejectsBothSources(t *testing.T) {
	if _, err := resolveBody("text", "/tmp/body.txt"); err == nil {
		t.Fatal("resolveBody() accepted body and body-file together")
	}
}
