package main

/*
#include <windows.h>
*/
import "C"
import (
	"sync"
	"syscall"
	"unicode/utf8"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
)

// C文字列ポインタを受け取り、UTF-8かShift_JIS(CP932)かを判定してGoの文字列(UTF-8)に変換します
func cStrToUTF8String(cstr *C.char) string {
	if cstr == nil {
		return ""
	}
	rawBytes := C.GoBytes(unsafe.Pointer(cstr), C.int(C.strlen(cstr)))
	if len(rawBytes) == 0 {
		return ""
	}

	// すでに正常なUTF-8バイト列であればそのまま文字列化
	if utf8.Valid(rawBytes) {
		return string(rawBytes)
	}

	// UTF-8として不正な場合はShift_JIS (CP932 / CP_ACP) とみなしてUTF-16経由でUTF-8に変換
	// CP_ACP = 0 (システムのANSIコードページ、日本語Windowsでは932)
	nWide, err := windows.MultiByteToWideChar(0, 0, &rawBytes[0], int32(len(rawBytes)), nil, 0)
	if err != nil || nWide <= 0 {
		return string(rawBytes)
	}

	wideBuf := make([]uint16, nWide)
	_, err = windows.MultiByteToWideChar(0, 0, &rawBytes[0], int32(len(rawBytes)), &wideBuf[0], nWide)
	if err != nil {
		return string(rawBytes)
	}

	return windows.UTF16ToString(wideBuf)
}

var (
	kernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procWideCharToMultiByte    = kernel32.NewProc("WideCharToMultiByte")
	user32                     = windows.NewLazySystemDLL("user32.dll")
	procPostMessageW           = user32.NewProc("PostMessageW")
)

// UTF-8文字列をShift_JIS(CP932)バイト列に変換します
func utf8ToSJIS(s string) []byte {
	if len(s) == 0 {
		return []byte{0}
	}
	utf16Chars := windows.StringToUTF16(s)
	// WideCharToMultiByte (CP_ACP = 0)
	r0, _, _ := procWideCharToMultiByte.Call(
		0, 0,
		uintptr(unsafe.Pointer(&utf16Chars[0])),
		uintptr(len(utf16Chars)),
		0, 0, 0, 0,
	)
	nBytes := int(r0)
	if nBytes <= 0 {
		return append([]byte(s), 0)
	}
	buf := make([]byte, nBytes)
	_, _, _ = procWideCharToMultiByte.Call(
		0, 0,
		uintptr(unsafe.Pointer(&utf16Chars[0])),
		uintptr(len(utf16Chars)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(nBytes),
		0, 0,
	)
	return buf
}

// Win32 RECT 構造体
type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type WebviewInstance struct {
	chromium    *edge.Chromium
	hwnd        uintptr
	lastMessage string
	notifyHwnd  uintptr
	notifyMsg   uint32
	mu          sync.Mutex
}

var (
	instances = map[uintptr]*WebviewInstance{}
	instMu    sync.Mutex
	nextID    uintptr = 1
)

func getInstance(id uintptr) *WebviewInstance {
	instMu.Lock()
	defer instMu.Unlock()
	return instances[id]
}

// controllerのVTable(PutBounds)を呼び出して描画領域を設定するヘルパー関数
func setControllerBounds(controller *edge.ICoreWebView2Controller, r rect) error {
	if controller == nil {
		return windows.ERROR_INVALID_HANDLE
	}
	// controller構造体は先頭フィールドにvtblポインタを持つ
	type controllerHeader struct {
		vtbl *[7]uintptr
	}

	header := (*controllerHeader)(unsafe.Pointer(controller))
	if header.vtbl == nil {
		return windows.ERROR_INVALID_HANDLE
	}
	// 6: PutBounds
	putBoundsFn := header.vtbl[6]

	var ret uintptr
	// 32bit(386)環境ではCOMのRECT引数(16byte)はスタックに各フィールド値が展開されて渡されます
	if unsafe.Sizeof(uintptr(0)) == 4 {
		ret, _, _ = syscall.SyscallN(
			putBoundsFn,
			uintptr(unsafe.Pointer(controller)),
			uintptr(r.Left),
			uintptr(r.Top),
			uintptr(r.Right),
			uintptr(r.Bottom),
		)
	} else {
		// 64bit(amd64)環境ではポインタ渡し
		ret, _, _ = syscall.SyscallN(
			putBoundsFn,
			uintptr(unsafe.Pointer(controller)),
			uintptr(unsafe.Pointer(&r)),
		)
	}

	if ret != 0 {
		return windows.Errno(ret)
	}
	return nil
}

// 指定された親ウィンドウ(parentHwnd)にWebView2を埋め込みます
//
//export WV2_Create
func WV2_Create(parentHwnd C.HWND, x C.int, y C.int, width C.int, height C.int, debug C.int) C.uintptr_t {
	chromium := edge.NewChromium()
	hwnd := uintptr(unsafe.Pointer(parentHwnd))

	inst := &WebviewInstance{
		chromium: chromium,
		hwnd:     hwnd,
	}

	chromium.MessageCallback = func(msg string) {
		inst.mu.Lock()
		inst.lastMessage = msg
		notifyHwnd := inst.notifyHwnd
		notifyMsg := inst.notifyMsg
		inst.mu.Unlock()

		if notifyHwnd != 0 && notifyMsg != 0 {
			_, _, _ = procPostMessageW.Call(notifyHwnd, uintptr(notifyMsg), 0, 0)
		}
	}

	if !chromium.Embed(hwnd) {
		return 0
	}

	// 開発者ツールの有効/無効
	settings, err := chromium.GetSettings()
	if err == nil && settings != nil {
		_ = settings.PutAreDevToolsEnabled(debug != 0)
		_ = settings.PutAreDefaultContextMenusEnabled(debug != 0)
	}

	controller := chromium.GetController()
	if controller != nil {
		r := rect{
			Left:   int32(x),
			Top:    int32(y),
			Right:  int32(x + width),
			Bottom: int32(y + height),
		}
		_ = setControllerBounds(controller, r)
		_ = controller.PutIsVisible(true)
	}

	instMu.Lock()
	id := nextID
	nextID++
	instances[id] = inst
	instMu.Unlock()

	return C.uintptr_t(id)
}

// 表示位置・サイズを変更します
//
//export WV2_SetBounds
func WV2_SetBounds(id C.uintptr_t, x C.int, y C.int, width C.int, height C.int) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || inst.chromium == nil {
		return 0
	}

	controller := inst.chromium.GetController()
	if controller != nil {
		r := rect{
			Left:   int32(x),
			Top:    int32(y),
			Right:  int32(x + width),
			Bottom: int32(y + height),
		}
		if err := setControllerBounds(controller, r); err == nil {
			return 1
		}
	}
	return 0
}

// 表示/非表示を切り替えます
//
//export WV2_SetVisible
func WV2_SetVisible(id C.uintptr_t, visible C.int) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || inst.chromium == nil {
		return 0
	}

	controller := inst.chromium.GetController()
	if controller != nil {
		if err := controller.PutIsVisible(visible != 0); err == nil {
			return 1
		}
	}
	return 0
}

// URLに遷移します
//
//export WV2_Navigate
func WV2_Navigate(id C.uintptr_t, url *C.char) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || inst.chromium == nil || url == nil {
		return 0
	}
	inst.chromium.Navigate(cStrToUTF8String(url))
	return 1
}

// HTML文字列を直接読み込みます
//
//export WV2_NavigateToString
func WV2_NavigateToString(id C.uintptr_t, htmlContent *C.char) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || inst.chromium == nil || htmlContent == nil {
		return 0
	}
	inst.chromium.NavigateToString(cStrToUTF8String(htmlContent))
	return 1
}

// JavaScriptを実行します
//
//export WV2_ExecuteScript
func WV2_ExecuteScript(id C.uintptr_t, script *C.char) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || inst.chromium == nil || script == nil {
		return 0
	}
	inst.chromium.Eval(cStrToUTF8String(script))
	return 1
}

// ドキュメント読み込み時に実行するJavaScriptを登録します
//
//export WV2_AddInitScript
func WV2_AddInitScript(id C.uintptr_t, script *C.char) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || inst.chromium == nil || script == nil {
		return 0
	}
	inst.chromium.Init(cStrToUTF8String(script))
	return 1
}

// ページを再読み込みします
//
//export WV2_Reload
func WV2_Reload(id C.uintptr_t) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || inst.chromium == nil {
		return 0
	}
	// JavaScript経由でリロードを実行
	inst.chromium.Eval("window.location.reload();")
	return 1
}

// ページの読み込みを中止します
//
//export WV2_Stop
func WV2_Stop(id C.uintptr_t) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || inst.chromium == nil {
		return 0
	}
	// JavaScript経由で読み込みを中止
	inst.chromium.Eval("window.stop();")
	return 1
}

// WebView2へフォーカスを当てます
//
//export WV2_Focus
func WV2_Focus(id C.uintptr_t) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || inst.chromium == nil {
		return 0
	}
	inst.chromium.Focus()
	return 1
}

// Web側からのメッセージ受信時に通知するウィンドウとメッセージIDを設定します
//
//export WV2_SetNotifyWindow
func WV2_SetNotifyWindow(id C.uintptr_t, targetHwnd C.HWND, msg C.uint) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil {
		return 0
	}
	inst.mu.Lock()
	inst.notifyHwnd = uintptr(unsafe.Pointer(targetHwnd))
	inst.notifyMsg = uint32(msg)
	inst.mu.Unlock()
	return 1
}

// Web側から送信された直近のメッセージを取得します
//
//export WV2_GetLastMessage
func WV2_GetLastMessage(id C.uintptr_t, outBuf *C.char, maxLen C.int) C.int {
	inst := getInstance(uintptr(id))
	if inst == nil || outBuf == nil || maxLen <= 0 {
		return 0
	}

	inst.mu.Lock()
	msg := inst.lastMessage
	inst.mu.Unlock()

	// Web側(UTF-8)からのメッセージをHSP3向けにShift_JIS(CP932)へ変換
	sjisBytes := utf8ToSJIS(msg)
	if len(sjisBytes) >= int(maxLen) {
		sjisBytes = sjisBytes[:maxLen-1]
	}

	dst := unsafe.Slice((*byte)(unsafe.Pointer(outBuf)), maxLen)
	copy(dst, sjisBytes)
	dst[len(sjisBytes)] = 0

	return C.int(len(sjisBytes))
}

// WebView2インスタンスを破棄します
//
//export WV2_Destroy
func WV2_Destroy(id C.uintptr_t) C.int {
	instMu.Lock()
	inst, ok := instances[uintptr(id)]
	if ok {
		delete(instances, uintptr(id))
	}
	instMu.Unlock()

	if !ok || inst == nil {
		return 0
	}

	if inst.chromium != nil {
		controller := inst.chromium.GetController()
		if controller != nil {
			// ICoreWebView2Controller::Close() の呼び出し
			type controllerHeader struct {
				vtbl *[25]uintptr
			}
			header := (*controllerHeader)(unsafe.Pointer(controller))
			if header.vtbl != nil {
				// 24: Close (IUnknown 3個 + 前段21個)
				closeFn := header.vtbl[24]
				_, _, _ = syscall.SyscallN(closeFn, uintptr(unsafe.Pointer(controller)))
			}
		}
	}
	return 1
}

func main() {
}
