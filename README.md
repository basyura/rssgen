# rssgen

HTML ページから指定した一覧要素を抽出し、子要素内のリンクを RSS 2.0
として出力する CLI です。

## 使い方

```sh
go run . -config config.example.yml
```

ファイルへ出力する場合:

```sh
go run . -config config.example.yml -output feed.xml
```

## 設定

設定ファイルは YAML です。

```yaml
settings:
  channel:
    link: https://basyura.org/rssgen/feed.xml

feeds:
  - title: basyura's feed
    url: https://bz-vermillion.com/news/
    xpath: ul.news_list
```

`xpath` には XPath を指定できます。`ul.news_list` のような
`tag.class` 形式も利用できます。この場合は該当要素の直接の子要素を
繰り返し項目とみなし、その子要素内の最初のリンクを RSS item にします。

`feeds` は複数指定できます。複数指定した場合は RSS XML を標準出力へ
順番に出力します。`-output` は設定が 1 件の場合のみ利用できます。

`settings.channel.link` は RSS チャンネルのリンクです。未指定の場合は
各 `feeds` の `url` が使われます。
