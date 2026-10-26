package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertMarkdownFileBuildsSanitizedPageWithSourceBase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	markdown := "# 日本語\n\n![image](images/picture.png)\n\n<script>alert(1)</script>"
	if err := os.WriteFile(path, []byte(markdown), 0o600); err != nil {
		t.Fatal(err)
	}

	page, err := convertMarkdownFile(path)
	if err != nil {
		t.Fatal(err)
	}
	htmlPage := string(page)
	if !strings.Contains(htmlPage, `<base href="file:///`) {
		t.Fatalf("expected a file URL base, got no base tag")
	}
	if !strings.Contains(htmlPage, "日本語") {
		t.Fatal("expected UTF-8 Markdown content in the generated page")
	}
	if !strings.Contains(htmlPage, `globalThis.DOMPurify.sanitize(rendered)`) {
		t.Fatal("expected rendered Markdown to be sanitized")
	}
	if !strings.Contains(htmlPage, `images/picture.png`) {
		t.Fatal("expected relative image URL to remain in the source directory context")
	}
}

func TestDecodeMarkdownUTF8BOM(t *testing.T) {
	got, err := decodeMarkdown([]byte{0xef, 0xbb, 0xbf, 'o', 'k'})
	if err != nil {
		t.Fatal(err)
	}
	if got != "ok" {
		t.Fatalf("decodeMarkdown() = %q, want %q", got, "ok")
	}
}
