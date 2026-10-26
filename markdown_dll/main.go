package main

/*
#include <windows.h>
*/
import "C"

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed assets/markdown/marked.min.js assets/markdown/purify.min.js
var markdownAssets embed.FS

const maxMarkdownBytes = 32 * 1024 * 1024
const maxHTMLBytes = 64 * 1024 * 1024

var (
	libraryOnce sync.Once
	markedJS    string
	purifyJS    string
	libraryErr  error
)

func loadLibraries() error {
	libraryOnce.Do(func() {
		marked, err := markdownAssets.ReadFile("assets/markdown/marked.min.js")
		if err != nil {
			libraryErr = fmt.Errorf("read embedded Marked.js: %w", err)
			return
		}
		purify, err := markdownAssets.ReadFile("assets/markdown/purify.min.js")
		if err != nil {
			libraryErr = fmt.Errorf("read embedded DOMPurify: %w", err)
			return
		}
		markedJS = string(marked)
		purifyJS = string(purify)
	})
	return libraryErr
}

func cStringToUTF8(value *C.char) (string, error) {
	if value == nil {
		return "", errors.New("path is nil")
	}
	raw := C.GoBytes(unsafe.Pointer(value), C.int(C.strlen(value)))
	if utf8.Valid(raw) {
		return string(raw), nil
	}
	if len(raw) == 0 {
		return "", nil
	}

	wideLen, err := windows.MultiByteToWideChar(0, 0, &raw[0], int32(len(raw)), nil, 0)
	if err != nil {
		return "", fmt.Errorf("decode path using the system code page: %w", err)
	}
	if wideLen <= 0 {
		return "", errors.New("decode path using the system code page: no characters decoded")
	}
	wide := make([]uint16, wideLen)
	if _, err := windows.MultiByteToWideChar(0, 0, &raw[0], int32(len(raw)), &wide[0], wideLen); err != nil {
		return "", fmt.Errorf("decode path using the system code page: %w", err)
	}
	return windows.UTF16ToString(wide), nil
}

func decodeMarkdown(data []byte) (string, error) {
	if len(data) >= 3 && data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf {
		data = data[3:]
	}
	if utf8.Valid(data) {
		return string(data), nil
	}
	if len(data) == 0 {
		return "", nil
	}

	wideLen, err := windows.MultiByteToWideChar(0, 0, &data[0], int32(len(data)), nil, 0)
	if err != nil {
		return "", fmt.Errorf("decode Markdown using the system code page: %w", err)
	}
	if wideLen <= 0 {
		return "", errors.New("decode Markdown using the system code page: no characters decoded")
	}
	wide := make([]uint16, wideLen)
	if _, err := windows.MultiByteToWideChar(0, 0, &data[0], int32(len(data)), &wide[0], wideLen); err != nil {
		return "", fmt.Errorf("decode Markdown using the system code page: %w", err)
	}
	return windows.UTF16ToString(wide), nil
}

func fileDirectoryURL(directory string) string {
	path := filepath.ToSlash(directory)
	if len(path) >= 2 && path[1] == ':' {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func convertMarkdownFile(path string) ([]byte, error) {
	if err := loadLibraries(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Markdown file: %w", err)
	}
	if len(data) > maxMarkdownBytes {
		return nil, fmt.Errorf("Markdown file exceeds %d bytes", maxMarkdownBytes)
	}
	markdown, err := decodeMarkdown(data)
	if err != nil {
		return nil, err
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve Markdown file path: %w", err)
	}
	markdownJSON, err := json.Marshal(markdown)
	if err != nil {
		return nil, fmt.Errorf("encode Markdown content: %w", err)
	}

	baseURL := html.EscapeString(fileDirectoryURL(filepath.Dir(absolutePath)))
	title := html.EscapeString(filepath.Base(path))
	page := fmt.Sprintf(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<base href="%s">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
body { font-family: "Segoe UI", "Meiryo", sans-serif; line-height: 1.65; margin: 2rem auto; padding: 0 1.5rem; max-width: 960px; color: #24292f; }
img { max-width: 100%%; height: auto; }
pre { overflow: auto; padding: 1rem; background: #f6f8fa; border-radius: 6px; }
code { font-family: Consolas, "Courier New", monospace; }
blockquote { margin-left: 0; padding: 0 1rem; color: #57606a; border-left: 4px solid #d0d7de; }
table { border-collapse: collapse; } th, td { border: 1px solid #d0d7de; padding: .4rem .7rem; }
</style>
</head>
<body><main id="content"></main>
<script>%s</script>
<script>%s</script>
<script>
const markdown = %s;
const rendered = globalThis.marked.parse(markdown);
document.getElementById("content").innerHTML = globalThis.DOMPurify.sanitize(rendered);
</script>
</body>
</html>`, baseURL, title, markedJS, purifyJS, markdownJSON)
	result := []byte(page)
	if len(result) > maxHTMLBytes {
		return nil, fmt.Errorf("converted HTML exceeds %d bytes", maxHTMLBytes)
	}
	return result, nil
}

//export MD_Init
func MD_Init() C.int {
	if err := loadLibraries(); err != nil {
		return 0
	}
	return 1
}

// MD_ConvertFile converts a Markdown file to UTF-8 HTML.
// It returns the HTML byte length on success, 0 on error, or the negative
// required buffer size (including the terminating NUL) when the buffer is small.
//
//export MD_ConvertFile
func MD_ConvertFile(path *C.char, outBuf *C.char, capacity C.int) C.int {
	if path == nil || capacity < 0 || int64(capacity) > maxHTMLBytes+1 {
		return 0
	}
	if capacity > 0 && outBuf == nil {
		return 0
	}

	filePath, err := cStringToUTF8(path)
	if err != nil {
		return 0
	}
	result, err := convertMarkdownFile(filePath)
	if err != nil {
		return 0
	}
	required := len(result) + 1
	if capacity < C.int(required) {
		return -C.int(required)
	}

	dst := unsafe.Slice((*byte)(unsafe.Pointer(outBuf)), int(capacity))
	copy(dst, result)
	dst[len(result)] = 0
	return C.int(len(result))
}

func main() {}
