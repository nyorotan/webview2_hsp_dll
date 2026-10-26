🌐 **English** | [日本語](./README.ja.md)

# webview2_hsp.dll

A **WebView2 embedding DLL and HSP3 include module** designed for Hot Soup Processor (HSP3) as a modern replacement for the legacy Internet Explorer component (`axobj`).

This package is based on [go-webview2](https://github.com/jchv/go-webview2), a Go interface for Microsoft Edge's WebView2 component, compiled into a Windows DLL using cgo.

It supports rendering local HTML, navigating websites, bidirectional communication with JavaScript, window resizing, and other versatile features.

---

### File Overview

1. **[main.go](./main.go)**
   - Go DLL implementation exporting a C-compatible interface.
   - Embeds WebView2 into an HSP3 window handle (`HWND`), controls position/size, and handles JavaScript interaction.
2. **[hsp3webview2.as](./hsp3webview2.as)**
   - HSP3 include module (Shift-JIS encoded).
   - Provides commands such as `wv2_create`, `wv2_navigate`, `wv2_set_html`, and `wv2_eval`.
3. **[sample.hsp](./sample.hsp)**
   - Sample script demonstrating local HTML rendering, JavaScript execution, and event reception from the web side via `postMessage`.

### Requirements

- Windows 10 / 11 (with Microsoft Edge WebView2 Runtime installed)
- HSP 3.6 or later

---

### Basic Usage

1. Include the module using `#include "hsp3webview2.as"`.
2. Create a window (e.g., via `screen`) and obtain its `hwnd`.
3. Call `wv2_create` to instantiate WebView2 and embed it into the window, obtaining an instance identifier (`id`).
4. Call `wv2_navigate`, `wv2_set_html`, or other commands using the obtained `id`.

#### Minimal Sample Code
```hsp
#include "hsp3webview2.as"

    screen 0, 800, 600
    title "WebView2 HSP3 Minimal Sample"

    // Create WebView2 (var id, parent HWND, X, Y, Width, Height, debugEnabled)
    wv2_create id, hwnd, 0, 0, 800, 600, 1

    // Navigate to a URL
    wv2_navigate id, "https://www.google.com"
    stop
```

### Available HSP3 Commands

| Command | Syntax | Description |
|---|---|---|
| `wv2_create` | `wv2_create id, hwnd, x, y, w, h, debug` | Creates a WebView2 instance at the specified window and coordinates (`debug=1` enables F12 DevTools).<br>`id` is the variable receiving the instance ID, and `hwnd` is the parent window handle. |
| `wv2_set_bounds` | `wv2_set_bounds id, x, y, w, h` | Updates position and size (typically called on window resize). |
| `wv2_navigate` | `wv2_navigate id, "URL"` | Navigates to a URL (`http://`, `https://`, or `file:///...`). |
| `wv2_set_html` | `wv2_set_html id, "HTML string"` | Directly renders a local HTML string. |
| `wv2_eval` | `wv2_eval id, "JS code"` | Executes JavaScript code asynchronously. |
| `wv2_add_init_script` | `wv2_add_init_script id, "JS code"` | Registers a script to execute automatically on document load. |
| `wv2_set_notify` | `wv2_set_notify id, hwnd, msg_id` | Configures the window message sent to HSP3 when `window.chrome.webview.postMessage` is triggered in web content. |
| `wv2_get_message` | `wv2_get_message id, buf, max_len` | Retrieves the message string received from JavaScript. |
| `wv2_reload` | `wv2_reload id` | Reloads the current page. |
| `wv2_destroy` | `wv2_destroy id` | Destroys the WebView2 instance. |

> [!NOTE]
> **About the Instance Identifier (`id`):**
> The `id` returned by `wv2_create` is an internal sequential integer ID (1, 2, ...) issued by the DLL to manage WebView2 instances, **not a Windows handle (`HWND`)**.
> Do not pass it directly to Windows APIs such as `MoveWindow`. Always pass it to the `wv2_*` commands provided by this module.

> [!TIP]
> **Automatic Character Encoding Detection:**
> Both **Shift_JIS (CP932)** (HSP3 default) and **UTF-8** strings are supported. The DLL automatically detects the encoding and converts it appropriately for WebView2, eliminating the need for manual conversion in HSP3. Messages received from JavaScript are also automatically converted back to Shift_JIS.

---

### Building the DLL

> Since this project uses cgo, a GCC environment is required for building.
> Below are the build instructions using `MinGW-w64`.

#### Build for your target HSP3 runtime architecture

Install `MinGW-w64 32-bit` to build a 32-bit DLL, or `MinGW-w64 64-bit` to build a 64-bit DLL, and add them to your `PATH`.

Run the following commands in Command Prompt:

> **Important (32-bit vs. 64-bit):**
> Standard HSP3 (default runtime) is **32-bit**. Therefore, to call it from standard HSP3, build with `GOARCH=386` (requires 32-bit GCC). If you use a 64-bit runtime (such as `hsp3_64`), build with `GOARCH=amd64` (requires 64-bit GCC).

```cmd
REM --- For 64-bit HSP3 (using MinGW-w64 64-bit gcc) ---
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1
set CC=x86_64-w64-mingw32-gcc
set CXX=x86_64-w64-mingw32-g++
go build -buildmode=c-shared -o webview2_hsp.dll

REM --- For 32-bit HSP3 (using MinGW-w64 32-bit gcc) ---
set GOOS=windows
set GOARCH=386
set CGO_ENABLED=1
set CC=i686-w64-mingw32-gcc
set CXX=i686-w64-mingw32-g++
go build -buildmode=c-shared -o webview2_hsp.dll
```

Place the generated `webview2_hsp.dll` in the same directory as [sample.hsp](./sample.hsp), and run [sample.hsp](./sample.hsp) from the HSP3 Script Editor to test it.

## License

This project is released under the `MIT License`. Feel free to use it for both commercial and non-commercial purposes.

- **Copyright**: © 2026 nyorotan

## Version Information

- **Version**: v1.0.0
- **Author**: nyorotan
