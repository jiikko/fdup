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

## 設定ファイル (config.yaml)

`fdup init`を実行すると`.fdup/config.yaml`が作成されます。

### 構造

```yaml
patterns:
  - name: パターン名
    regex: 正規表現パターン
ignore:
  - 無視パターン
test:
  - input: テスト入力
    expected: 期待される出力
```

### patterns

ファイル名からコードを抽出するための正規表現パターンを定義します。

```yaml
patterns:
  - name: standard
    regex: '([A-Z]{2,5}-\d{3,5})'
  - name: no_hyphen
    regex: '([A-Z]{2,5})(\d{3,5})'
```

| フィールド | 説明 |
|-----------|------|
| `name` | パターンの識別名（ログ出力用） |
| `regex` | 正規表現パターン（キャプチャグループ必須） |

**正規表現の仕様:**

- **大文字小文字を区別しない** - 自動的に`(?i)`フラグが付与される
- **キャプチャグループ必須** - `()`で囲んだ部分がコードとして抽出される
- **複数キャプチャグループ** - 複数のグループがある場合は結合される
- **パターンは順番に評価** - 最初にマッチしたパターンが使用される

**コードの正規化:**

抽出されたコードは以下のルールで正規化されます:

1. 大文字に変換
2. ハイフン（`-`）を削除
3. アンダースコア（`_`）を削除

例: `prj-001` → `PRJ001`, `hoge_9851` → `HOGE9851`

### ignore

スキャン対象から除外するパスのパターンを定義します。

```yaml
ignore:
  - node_modules/
  - .git/
  - "*.tmp"
  - "*.log"
  - .DS_Store
  - .fdup/
```

**パターンの種類:**

| パターン | 説明 | 例 |
|---------|------|----|
| `dir/` | 末尾が`/`の場合、ディレクトリ名にマッチ | `node_modules/` |
| `*.ext` | ワイルドカード（`*`）でglob形式マッチ | `*.tmp`, `*.log` |
| `name` | 完全一致（ファイル名・パス要素にマッチ） | `.DS_Store` |

**マッチング動作:**

- 相対パス全体に対してマッチ
- ファイル名単体に対してマッチ
- パスの各要素に対してマッチ
- ディレクトリがマッチした場合、その配下は再帰的にスキップ

### test

`fdup test`コマンドで使用するテストケースを定義します。

```yaml
test:
  - input: PRJ-001_final.zip
    expected: PRJ001
  - input: doc123.pdf
    expected: DOC123
  - input: random_file.txt
    expected: null  # マッチしないことを期待
```

| フィールド | 説明 |
|-----------|------|
| `input` | テスト対象のファイル名 |
| `expected` | 期待される正規化後のコード。`null`の場合はマッチしないことを期待 |

### 設定例

```yaml
# カメラの連番ファイル向け設定
patterns:
  - name: sony
    regex: '(DSC\d{5})'
  - name: canon
    regex: '(IMG_\d{4})'
  - name: gopro
    regex: '(GOPR\d{4})'
  - name: dji
    regex: '(DJI_\d{4})'

ignore:
  - .git/
  - .fdup/
  - "*.tmp"
  - Thumbs.db
  - .DS_Store

test:
  - input: DSC00001.ARW
    expected: DSC00001
  - input: IMG_1234.CR2
    expected: IMG1234
  - input: GOPR0001.MP4
    expected: GOPR0001
  - input: screenshot.png
    expected: null
```

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
