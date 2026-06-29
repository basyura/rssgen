# RSS 日付タグ追加計画

## 目的

生成する RSS 2.0 XML に `lastBuildDate` タグと `pubDate` タグを追加する。

## 修正案

- RSS チャンネルに `lastBuildDate` フィールドを追加する。
- RSS item に `pubDate` フィールドを追加する。
- 日付は RSS 2.0 で一般的な RFC 1123Z 形式で出力する。
- `lastBuildDate` は RSS 生成時点の現在時刻を使う。
- `pubDate` は HTML から抽出済みの `FeedItem.Date` を解析できる場合に出力する。
- 解析できない日付、または日付が空の場合は `pubDate` を省略する。
- 既存のタイトル整形で使っている日付文字列は変更しない。

## 実装方針

- `Channel` 構造体に `LastBuildDate string` を追加する。
- `Item` 構造体に `PubDate string` を追加する。
- `buildRSS` で現在時刻を受け取れるようにし、テストで固定時刻を使える形にする。
- `FeedItem.Date` の `2026.6.26` のような形式を `time.Time` に変換する補助関数を追加する。
- 変換できた item のみ `pubDate` を設定する。

## テスト

- `buildRSS` のテストで `lastBuildDate` が設定されることを確認する。
- 日付あり item で `pubDate` が RFC 1123Z 形式になることを確認する。
- 日付なし item では `pubDate` が空のままになることを確認する。
- `go test ./...` を実行する。

## 実施結果

- RSS チャンネルへ `lastBuildDate` を追加した。
- RSS item へ `pubDate` を追加した。
- `lastBuildDate` は RSS 生成時点の時刻を RFC 1123Z 形式で出力する。
- `pubDate` は `FeedItem.Date` を `2006.1.2` 形式として解析できた場合のみ、
  同じく RFC 1123Z 形式で出力する。
- 日付が空、または解析できない item では `pubDate` を省略する。
- 固定時刻を使った単体テストを追加し、`go test ./...` が成功した。
