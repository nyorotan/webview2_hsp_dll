🌐 [English](./README.md) | **日本語**

# webview2_hsp.dll

HSP3から従来のIEコンポーネント（`axobj`）の代替として利用できる **WebView2 埋め込み用DLLおよびHSP3用インクルードモジュール** を作成しました。

このパッケージは、Go言語でMicrosoft EdgeのWebView2コンポーネントを使用するためのインターフェース go-webview2 をベースとしており、cgo を用いてDLL化してあります。

ローカルHTMLの表示やWebサイトの遷移はもちろん、JavaScriptの双方向連携やリサイズ対応など汎用的な機能を実装しています。

---

### ソースファイル一覧

1. **[main.go](./main.go)**
   - C言語互換インターフェース（エクスポート関数）を持つGo側DLL実装。
   - HSP3のウィンドウハンドル（`HWND`）に埋め込み、位置・サイズ制御やJS連携を行います。
2. **[hsp3webview2.as](./hsp3webview2.as)**
   - HSP3から呼び出すためのインクルードモジュール（Shift-JIS）。
   - `wv2_create`, `wv2_navigate`, `wv2_set_html`, `wv2_eval` などの命令を提供します。
3. **[sample.hsp](./sample.hsp)**
   - HTML表示、JavaScript連携、Web側通知に加え、独立Markdown DLLを使ったファイル選択・表示のサンプルです。
4. **[hsp3markdown.as](./hsp3markdown.as) と [markdown_dll](./markdown_dll)**
   - Markdown変換用の独立DLL `markdown_hsp.dll` とHSP3モジュールです。必要なアプリだけが追加で利用できます。

### 動作要件

- Windows 10 / 11（Microsoft Edge WebView2 Runtime がインストールされている環境）
- HSP 3.6 以降

Markdown表示を利用する場合は、[build_markdown.bat](./build_markdown.bat) で `markdown_hsp.dll` を別途ビルドし、[hsp3markdown.as](./hsp3markdown.as) をインクルードします。Marked.js と DOMPurify は [markdown_dll/assets/markdown](./markdown_dll/assets/markdown) からMarkdown DLLに埋め込まれ、実行時に追加のライブラリファイルやネット接続は不要です。[sample.hsp](./sample.hsp) はMarkdownモジュールを利用する例です。

---

### 基本的な使い方

1. `#include "hsp3webview2.as"` でモジュールを読み込みます。
2. `screen` 等でウィンドウを作成し、`hwnd` を取得します。
3. `wv2_create` で WebView2 を生成してウィンドウに埋め込み、識別用ID（`id`）を取得します。
4. 取得した `id` に対して `wv2_navigate` や `wv2_set_html` などを呼び出します。

#### 最小限のサンプルコード
```hsp
#include "hsp3webview2.as"

    screen 0, 800, 600
    title "WebView2 HSP3 Minimal Sample"

    // WebView2の作成 (変数id, 親HWND, X, Y, 幅, 高さ, デバッグ有効フラグ)
    wv2_create id, hwnd, 0, 0, 800, 600, 1

    // 指定URLへ移動
    wv2_navigate id, "https://www.google.com"
    stop
```

### 提供される主なHSP3命令

| 命令 | 書式 | 説明 |
|---|---|---|
| `wv2_create` | `wv2_create id, hwnd, x, y, w, h, debug` | 指定ウィンドウ・座標にWebView2を生成（debug=1でF12開発者ツール有効）<br>`id`は生成された識別用IDを受け取る変数、`hwnd`は親ウィンドウハンドル。 |
| `wv2_set_bounds` | `wv2_set_bounds id, x, y, w, h` | 表示位置・サイズを変更（ウィンドウリサイズ時に呼び出し） |
| `wv2_navigate` | `wv2_navigate id, "URL"` | URL（`http://` や `file:///...`）へ移動 |
| `wv2_set_html` | `wv2_set_html id, "HTML文字列"` | ローカルのHTML文字列を直接描画 |
| `wv2_eval` | `wv2_eval id, "JSコード"` | JavaScriptコードを実行 |
| `wv2_add_init_script` | `wv2_add_init_script id, "JSコード"` | ドキュメント読み込み時に毎回自動実行するスクリプトを登録 |
| `wv2_set_notify` | `wv2_set_notify id, hwnd, msg_id` | Web側の `window.chrome.webview.postMessage` 受信時にHSP3へ送るメッセージを設定 |
| `wv2_get_message` | `wv2_get_message id, buf, max_len` | 受信したメッセージ文字列を取得 |
| `wv2_reload` | `wv2_reload id` | ページの再読み込み |
| `wv2_destroy` | `wv2_destroy id` | WebView2インスタンスの破棄 |

#### Markdownモジュール（任意）

```hsp
#include "hsp3markdown.as"

    md_init
    if stat == 0 : dialog "Markdown DLL initialization failed.", 1
    wv2_show_markdown id, "docs\\guide.md"
    if stat == 0 : dialog "Markdown file could not be displayed.", 1
```

`hsp3markdown.as` は `hsp3webview2.as` も読み込みます。`wv2_show_markdown` はMarkdownファイルを読み込み、UTF-8 HTMLを生成して既存のWebView2へ渡します。相対画像などはMarkdownファイルの場所を基準に解決します。変換HTMLはDOMPurifyでサニタイズされ、入力ファイルは最大32 MiB、生成HTMLは最大64 MiBです。

> [!NOTE]
> **識別用ID（`id`）について:**
> `wv2_create` で取得できる `id` は、Windowsのウィンドウハンドル（`HWND`）ではなく、**本DLLが内部で個々のWebView2を管理するために発行する識別用ID番号（1, 2... の連番整数）** です。
> `MoveWindow` などのWindows APIに直接渡すことはできません。操作する際は必ず本モジュールの `wv2_*` 命令の引数として渡してください。

> [!TIP]
> **文字コード自動判定・変換機能:**
> HSP3標準の **Shift_JIS（CP932）** と **UTF-8** の両方に対応しています。DLL内部で自動判別し、適切に変換してWebView2へ渡すため、HSP3側で事前に文字コード変換を行う必要はありません。また、Web側（JavaScript）からメッセージを受信した際もShift_JISへ自動変換されて返されます。
> Markdown DLLはMarkdownファイルのUTF-8（BOM付き含む）とWindowsのシステムコードページに対応します。

---

### DLLのビルド手順

> cgo を使用しているためビルドには gcc環境 が必須になります。
> 以下は `MinGW-w64` の使用を前提としたビルド手順です。

#### 使用するHSP3ランタイムに合わせたビット版をビルドする

32bit版をビルドするには `MinGW-w64 32bit` 、64bit版をビルドするには `MinGW-w64 64bit` をインストールし、パスを通しておきます。

ターミナル（コマンドプロンプト）で以下のコマンドを実行することでDLLが生成されます。

> **注意 (32bit / 64bit):**
> 通常のHSP3（標準ランタイム）は **32bit版** です。そのため、標準のHSP3から呼び出す場合は `GOARCH=386`（32bit用GCCが必要）でビルドする必要があります。64bit版ランタイム（hsp3_64等）をご利用の場合は `GOARCH=amd64`（64bit用GCCが必要）でビルドする必要があります。

```cmd
REM --- 64bit版HSP3向け (MinGW-w64の64bit gcc環境がある場合) ---
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1
set CC=x86_64-w64-mingw32-gcc
set CXX=x86_64-w64-mingw32-g++
go build -buildmode=c-shared -o webview2_hsp.dll

REM --- 32bit版HSP3向け (MinGW-w64の32bit gcc環境がある場合) ---
set GOOS=windows
set GOARCH=386
set CGO_ENABLED=1
set CC=i686-w64-mingw32-gcc
set CXX=i686-w64-mingw32-g++
go build -buildmode=c-shared -o webview2_hsp.dll
```

生成された `webview2_hsp.dll` を [sample.hsp](./sample.hsp) と同じフォルダに配置し、HSP3スクリプトエディタから [sample.hsp](./sample.hsp) を実行して動作をご確認ください。

Markdown機能を使う場合は、別途 `build_markdown.bat` を実行して `markdown_hsp.dll` をビルドし、同じアプリケーションフォルダに配置してください。32bit/64bitはWebView2 DLLおよびHSPランタイムと一致させます。


## ライセンス

本プロジェクトは`MITライセンス`の元で公開されています。商用・非商用問わず自由にご利用ください。

- **著作権**: © 2026 nyorotan

## バージョン情報

- **バージョン**: v1.0.0
- **作者**: nyorotan
