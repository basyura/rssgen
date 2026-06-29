# pubDate 降順ソート計画

## 目的

生成する RSS 2.0 XML の `item` を `pubDate` の降順で出力する。

## 修正案

- `buildRSS` で RSS item を組み立てる際、`FeedItem.Date` を解析して得た日時を保持する。
- `pubDate` を持つ item は新しい日付が先になるように並べ替える。
- `pubDate` を持たない item は日付あり item の後ろに置く。
- 同じ日時、または日付なし item 同士は元の順序を維持する。
- `settings.limit` を読み込み、降順ソート後の item を指定件数までに制限する。
- `settings.limit` が未指定または 0 以下の場合は、全 item を出力する。
- 各フィード取得時の固定 5 件制限を外し、最終的な `settings.limit` で件数を制御する。

## 実装方針

- `buildRSS` 内でソート用の日時を持つ一時構造体を使う。
- `sort.SliceStable` で安定ソートし、同順序の item が不用意に入れ替わらないようにする。
- 日付解析処理は既存の `parseFeedItemDate` を使い、RSS の `pubDate` 表示形式は変更しない。
- `Settings` に `Limit int` を追加し、`run` から `buildRSS` へ渡す。
- `buildRSS` は受け取った limit をソート後に適用する。
- `collectItemsFromHTML` は固定件数で打ち切らず、取得できた item を返す。

## テスト

- `buildRSS` のテストに、入力順と異なる日付の item を追加する。
- 生成された `rss.Channel.Items` が `pubDate` の降順になることを確認する。
- 日付なし item が日付あり item の後に残ることを確認する。
- `settings.limit` が YAML から読み込まれることを確認する。
- limit 指定時に降順ソート後の上位 item だけが出力されることを確認する。
- HTML からの収集が 5 件で打ち切られないことを確認する。
- `go test ./...` を実行する。

## 実施結果

- `buildRSS` で `pubDate` の降順になるように安定ソートを追加した。
- `pubDate` がない item は日付あり item の後に出力するようにした。
- 同じ日付、または日付なし item 同士は元の順序を維持する。
- `buildRSS` のテストで降順と日付なし item の位置を確認するようにした。
- `settings.limit` を読み込み、降順ソート後の item 数を制限するようにした。
- limit 指定時に降順ソート後の上位 item だけが出力されることをテストした。
- 各フィード取得時の固定 5 件制限を削除し、最終的な `settings.limit` で件数を制御するようにした。
- HTML からの収集が 5 件で打ち切られないことをテストした。
- `go test ./...` が成功した。
