# RSS 生成プログラム実装計画

## 目的

設定ファイルに指定された URL と XPath をもとに HTML を取得し、XPath で
見つけた要素の子要素を一覧として扱い、その中のリンクを RSS item として
出力するプログラムを作成する。

最初の検証対象は以下とする。

- URL: `https://bz-vermillion.com/news/`
- XPath: `ul.news_list`
- 一覧要素: `ul.news_list` の子要素である `li`
- RSS item: 各 `li` 内のリンク

## 修正案

- `main.go` を CLI として実装する。
  - `-config` で設定ファイルを指定する。
  - `-output` で RSS XML の出力先を指定する。未指定時は標準出力へ出す。
- 設定ファイルは JSON とする。
  - URL と XPath の組を複数指定できる形にする。
  - RSS のタイトル、説明、リンクも設定可能にする。
- HTML 解析と XPath 評価には Go の既存ライブラリを利用する。
  - HTML 取得は標準ライブラリの `net/http` を使う。
  - HTML XPath は `github.com/antchfx/htmlquery` を使う。
- RSS 2.0 XML を生成する。
  - 各 XPath の一致要素について、その直接の子要素を繰り返し対象にする。
  - 各子要素内の最初のリンクを RSS item とする。
  - 相対 URL は取得元 URL を基準に絶対 URL へ変換する。
  - item のタイトルはリンクテキストを優先し、空なら URL を使う。
- `config.example.json` を追加する。
  - `https://bz-vermillion.com/news/` と `//ul[contains(concat(' ', normalize-space(@class), ' '), ' news_list ')]` を設定する。
- `README.md` に実行方法を書く。

## 検証

- `go test ./...` を実行する。
- サンプル設定で `go run . -config config.example.json` を実行し、
  RSS XML が出力されることを確認する。

## 実施結果

- `main.go` に設定読み込み、HTML 取得、XPath 抽出、RSS 2.0 出力を実装した。
- `config.example.json` を追加し、`https://bz-vermillion.com/news/` と
  `ul.news_list` で検証できるようにした。
- `README.md` に実行方法と設定形式を追記した。
- `main_test.go` に HTML 抽出、RSS 組み立て、`ul.news_list` 変換のテストを追加した。
- `go test ./...` は成功した。
- `go run . -config config.example.json` で実サイトから RSS XML が出力されることを確認した。

## 追加修正案

- `config.example.json` の形式に合わせ、設定ファイルを配列として読み込む。
- 各設定項目は `title`, `url`, `xpath` を持つものとする。
- RSS チャンネルの `title` と `description` は設定の `title` を使う。
- RSS チャンネルの `link` は設定の `url` を使う。
- 複数設定がある場合は、標準出力へ RSS XML を順番に出力する。
- `-output` は設定が 1 件の場合のみ許可し、複数設定ではエラーにする。

## 追加実施結果

- 設定ファイルを `config.example.json` の配列形式で読み込むよう変更した。
- RSS チャンネルの `title` と `description` に設定の `title` を使うよう変更した。
- RSS チャンネルの `link` に設定の `url` を使うよう変更した。
- 複数設定時の標準出力と、`-output` の単一設定制限を追加した。
- README とテストを新しい JSON 形式に更新した。
- `go test ./...` は成功した。
- `go run . -config config.example.json` で実サイトから RSS XML が出力されることを確認した。

## channel link 修正案

- 設定項目に任意の `link` を追加する。
- RSS チャンネルの `link` は、設定の `link` があればそれを使う。
- `link` が未指定の場合は、従来どおり設定の `url` を使う。
- `config.example.json` の `link` は `https://basyura.org` とする。

## channel link 実施結果

- `Config` に任意の `link` を追加した。
- RSS チャンネルの `link` に設定の `link` を優先して使うよう変更した。
- `config.example.json` に `"link": "https://basyura.org"` を追加した。
- README とテストを更新した。
- `go test ./...` は成功した。

## YAML 設定修正案

- 設定ファイルを JSON から YAML に変更する。
- YAML は `settings.channel.link` と `feeds` を持つ形式として読み込む。
- `settings.channel.link` は全フィードの RSS チャンネル `link` として使う。
- `feeds` の各項目は `title`, `url`, `xpath` を持つものとする。
- `config.example.yml` の `link:https://...` は YAML として曖昧なため、
  `link: https://...` に修正する。
- README とテストを YAML 形式に更新する。

## YAML 設定実施結果

- `gopkg.in/yaml.v3` を追加し、設定読み込みを YAML に変更した。
- `settings.channel.link` と `feeds` を読み込む構造を追加した。
- `settings.channel.link` を RSS チャンネル `link` に反映するよう変更した。
- README とテストを YAML 形式に更新した。
- `config.example.yml` の `link` 表記を YAML として有効な形式に修正した。
- `go test ./...` は成功した。
- `go run . -config config.example.yml` で実サイトから RSS XML が出力されることを確認した。
