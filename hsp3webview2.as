; ==============================================================================
; HSP3用 WebView2 埋め込みモジュール (hsp3webview2.as)
; ==============================================================================
; ※文字コード: Shift-JIS
; ※呼び出し規約: cdecl
; ==============================================================================

#ifndef __HSP3_WEBVIEW2__
#define __HSP3_WEBVIEW2__

#uselib "webview2_hsp.dll"
#func global _WV2_Create "WV2_Create" int, int, int, int, int, int
#func global _WV2_SetBounds "WV2_SetBounds" int, int, int, int, int
#func global _WV2_SetVisible "WV2_SetVisible" int, int
#func global _WV2_Navigate "WV2_Navigate" int, sptr
#func global _WV2_NavigateToString "WV2_NavigateToString" int, sptr
#func global _WV2_ExecuteScript "WV2_ExecuteScript" int, sptr
#func global _WV2_AddInitScript "WV2_AddInitScript" int, sptr
#func global _WV2_Reload "WV2_Reload" int
#func global _WV2_Stop "WV2_Stop" int
#func global _WV2_Focus "WV2_Focus" int
#func global _WV2_SetNotifyWindow "WV2_SetNotifyWindow" int, int, int
#func global _WV2_GetLastMessage "WV2_GetLastMessage" int, var, int
#func global _WV2_Destroy "WV2_Destroy" int

#module hsp3webview2

// WebView2 コントロールの初期化・埋め込み
// wv2_create var_id, target_hwnd, x, y, width, height, debug_flag
#deffunc wv2_create var id_out, int target_hwnd, int x, int y, int w, int h, int debug
    _WV2_Create target_hwnd, x, y, w, h, debug
    id_out = stat
    return id_out

// 表示位置・サイズの変更
#deffunc wv2_set_bounds int id, int x, int y, int w, int h
    _WV2_SetBounds id, x, y, w, h
    return stat

// 表示・非表示の切り替え (visible: 1=表示, 0=非表示)
#deffunc wv2_set_visible int id, int visible
    _WV2_SetVisible id, visible
    return stat

// 指定URLへの遷移 (http://, https://, file:///)
#deffunc wv2_navigate int id, str url
    _WV2_Navigate id, url
    return stat

// HTML文字列の直接表示
#deffunc wv2_set_html int id, str html_content
    _WV2_NavigateToString id, html_content
    return stat

// JavaScriptの実行
#deffunc wv2_eval int id, str script
    _WV2_ExecuteScript id, script
    return stat

// ページロード時に自動実行する初期化スクリプトの登録
#deffunc wv2_add_init_script int id, str script
    _WV2_AddInitScript id, script
    return stat

// 再読み込み
#deffunc wv2_reload int id
    _WV2_Reload id
    return stat

// 読み込み中止
#deffunc wv2_stop int id
    _WV2_Stop id
    return stat

// フォーカス設定
#deffunc wv2_focus int id
    _WV2_Focus id
    return stat

// JavaScript (window.chrome.webview.postMessage) からのメッセージ通知設定
#deffunc wv2_set_notify int id, int notify_hwnd, int message_id
    _WV2_SetNotifyWindow id, notify_hwnd, message_id
    return stat

// 直近の受信メッセージ文字列を取得
#deffunc wv2_get_message int id, var out_buf, int max_len
    _WV2_GetLastMessage id, out_buf, max_len
    return stat

// インスタンスの破棄
#deffunc wv2_destroy int id
    _WV2_Destroy id
    return stat

#global

#endif
