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
go install github.com/jiikko/fdup@latest
```

## コマンド

### グローバルオプション

すべてのコマンドで使用可能なオプション:

| オプション | 説明 |
|-----------|------|
| `-q, --quiet` | 出力を抑制 |
| `--verbose` | 詳細な出力を表示 |

### `fdup init`

カレントディレクトリに`.fdup/`を作成し、初期化します。

```bash
fdup init [options]
```

| オプション | 説明 |
|-----------|------|
| `-f, --force` | 既存データを削除して再初期化 |

### `fdup scan`

ファイルをスキャンしてインデックスを更新します。

```bash
fdup scan [options]
```

| オプション | 説明 |
|-----------|------|
| `-p, --progress` | プログレスバーを表示 |
| `-d, --drop` | データベースを削除して再作成 |

### `fdup dup`

重複ファイルを検出・一覧表示します。

```bash
fdup dup [options]
```

| オプション | 説明 |
|-----------|------|
| `-i, --interactive` | TUIモードで対話的に操作 |
| `-n, --dry-run` | 実際には変更せず、実行内容を表示 |
| `-t, --trash` | 削除ではなくゴミ箱に移動 |
| `-w, --web` | Web UIモードで起動 |

### `fdup test`

`config.yaml`に定義されたテストケースでパターンを検証します。

```bash
fdup test
```

### `fdup search <CODE>` (非推奨)

> **Warning**: このコマンドは非推奨です。将来のバージョンで削除予定です。

コードでファイルを検索します。

```bash
fdup search <CODE> [options]
```

| オプション | 説明 |
|-----------|------|
| `-e, --exact` | 完全一致のみ |
| `-j, --json` | JSON形式で出力 |

## 技術スタック

- Go
- SQLite (modernc.org/sqlite)
- Bubble Tea / Lipgloss

## 開発

### テスト用ディレクトリのセットアップ

```bash
make setup-test-dir
```

`./tmp` 以下にテスト用のファイルと設定が作成されます。

```bash
cd ./tmp
../fdup scan
../fdup dup
../fdup dup --web
```

テスト終了後のクリーンアップ:

```bash
make clean-test-dir
```

### テスト実行

```bash
make test
```

### Lint

```bash
make lint
```

## ライセンス

MIT
