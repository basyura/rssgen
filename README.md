# rssgen

複数の HTML ページから指定した一覧要素を抽出し、子要素内のリンクを
一つの RSS 2.0 フィードとして出力する CLI です。

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
    title: Music News
    link: https://basyura.org/rssgen/feed.xml
    description: 複数サイトの更新情報

feeds:
  - title: B'z NEWS
    url: https://bz-vermillion.com/news/
    xpath: ul.news_list
```

`xpath` には XPath を指定できます。`ul.news_list` のような
`tag.class` 形式も利用できます。この場合は該当要素の直接の子要素を
繰り返し項目とみなし、その子要素内の最初のリンクを RSS item にします。

`feeds` は複数指定できます。各サイトから取得した項目は一つの RSS
チャンネルにまとめられ、同じ URL の項目は重複を除外します。

`settings.channel.title`、`settings.channel.link`、
`settings.channel.description` は、統合後の RSS チャンネル情報です。
いずれも必須です。
