package main

import (
	"os"
	"strings"
	"testing"
)

func TestCollectItemsFromHTML(t *testing.T) {
	html := []byte(`
<!doctype html>
<html>
  <body>
    <ul class="news_list">
      <li><a href="/news/1">First News</a></li>
      <li><span><a href="https://example.com/news/2">Second News</a></span></li>
      <li><span>No link</span></li>
    </ul>
  </body>
</html>`)

	items, err := collectItemsFromHTML("https://example.com/news/", "ul.news_list", html)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Title != "First News" {
		t.Fatalf("items[0].Title = %q", items[0].Title)
	}
	if items[0].Link != "https://example.com/news/1" {
		t.Fatalf("items[0].Link = %q", items[0].Link)
	}
	if items[1].Title != "Second News" {
		t.Fatalf("items[1].Title = %q", items[1].Title)
	}
}

func TestBuildRSS(t *testing.T) {
	rss := buildRSS(Config{
		Title: "Test Feed",
		URL:   "https://example.com/",
		Link:  "https://feed.example.com/",
	}, []FeedItem{{Title: "Entry", Link: "https://example.com/entry"}})

	if rss.Version != "2.0" {
		t.Fatalf("rss.Version = %q", rss.Version)
	}
	if len(rss.Channel.Items) != 1 {
		t.Fatalf("len(rss.Channel.Items) = %d, want 1", len(rss.Channel.Items))
	}
	if rss.Channel.Link != "https://feed.example.com/" {
		t.Fatalf("rss.Channel.Link = %q", rss.Channel.Link)
	}
	if rss.Channel.Items[0].GUID.Value != "https://example.com/entry" {
		t.Fatalf("GUID.Value = %q", rss.Channel.Items[0].GUID.Value)
	}
}

func TestBuildRSSUsesURLAsDefaultLink(t *testing.T) {
	rss := buildRSS(Config{
		Title: "Test Feed",
		URL:   "https://example.com/",
	}, nil)

	if rss.Channel.Link != "https://example.com/" {
		t.Fatalf("rss.Channel.Link = %q", rss.Channel.Link)
	}
}

func TestLoadConfig(t *testing.T) {
	path := t.TempDir() + "/config.yml"
	err := os.WriteFile(path, []byte(`settings:
  channel:
    link: https://feed.example.com/

feeds:
  - title: Example Feed
    url: https://example.com/news/
    xpath: ul.news_list
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	configs, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 {
		t.Fatalf("len(configs) = %d, want 1", len(configs))
	}
	if configs[0].Title != "Example Feed" {
		t.Fatalf("configs[0].Title = %q", configs[0].Title)
	}
	if configs[0].URL != "https://example.com/news/" {
		t.Fatalf("configs[0].URL = %q", configs[0].URL)
	}
	if configs[0].Link != "https://feed.example.com/" {
		t.Fatalf("configs[0].Link = %q", configs[0].Link)
	}
	if configs[0].XPath != "ul.news_list" {
		t.Fatalf("configs[0].XPath = %q", configs[0].XPath)
	}
}

func TestNormalizeXPath(t *testing.T) {
	xpath := normalizeXPath("ul.news_list")
	if !strings.Contains(xpath, "news_list") {
		t.Fatalf("xpath = %q", xpath)
	}
	if got := normalizeXPath("//ul"); got != "//ul" {
		t.Fatalf("normalizeXPath xpath = %q", got)
	}
}
