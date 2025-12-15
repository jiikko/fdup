# fdup

ファイル名からパターン（ID・コード等）を抽出し、SQLiteでインデックス化して重複を検出するCLIツール。

## 概要

ファイル名に含まれるID・コード（例: `DSC00001`, `C0001`, `IMG_1234`）を正規表現で抽出し、同一コードを持つファイルが複数の場所に存在する場合に重複として検出します。

## 主な機能

- 正規表現パターンによるコード抽出
- SQLiteによるインデックス管理
- 重複ファイルの検出・一覧表示
- TUIによる対話的な重複整理

## インストール

```bash
go install github.com/koji/fdup@latest
```

## コマンド

| コマンド | 説明 |
|---------|------|
| `fdup init` | 初期化（`.fdup/`ディレクトリ作成） |
| `fdup scan` | ファイルをスキャンしてインデックス化 |
| `fdup dup` | 重複ファイルを一覧表示 |
| `fdup search <CODE>` | コードでファイルを検索 |
| `fdup test` | パターンのテスト実行 |

## 技術スタック

- Go
- SQLite (modernc.org/sqlite)
- Bubble Tea / Lipgloss

## ライセンス

MIT
