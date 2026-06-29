package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCollectItemsFromHTML(t *testing.T) {
	html := []byte(`
<!doctype html>
<html>
  <body>
    <ul class="news_list">
      <li><a href="/news/1"><p class="date">2026.6.26</p><p class="comment">First News</p></a></li>
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
	if items[0].Date != "2026.6.26" {
		t.Fatalf("items[0].Date = %q", items[0].Date)
	}
	if items[0].Link != "https://example.com/news/1" {
		t.Fatalf("items[0].Link = %q", items[0].Link)
	}
	if items[1].Title != "Second News" {
		t.Fatalf("items[1].Title = %q", items[1].Title)
	}
}

func TestCollectItemsFromHTMLWithDirectChildLinks(t *testing.T) {
	html := []byte(`
<section class="news-index">
  <a href="/news/1"><div><h5>2026.6.9</h5><div class="NW-R"><p>First News</p></div></div></a>
  <a href="/news/2"><div><p>Second News</p></div></a>
</section>`)

	items, err := collectItemsFromHTML("https://example.com/", ".news-index", html)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Title != "First News" {
		t.Fatalf("items[0].Title = %q", items[0].Title)
	}
	if items[0].Date != "2026.6.9" {
		t.Fatalf("items[0].Date = %q", items[0].Date)
	}
	if items[0].Link != "https://example.com/news/1" {
		t.Fatalf("items[0].Link = %q", items[0].Link)
	}
}

func TestCollectItemsFromHTMLDoesNotLimitItemsPerFeed(t *testing.T) {
	html := []byte(`
<ul class="news">
  <li><a href="/news/1">One</a></li>
  <li><a href="/news/1">One Duplicate</a></li>
  <li><a href="/news/2">Two</a></li>
  <li><a href="/news/3">Three</a></li>
  <li><a href="/news/4">Four</a></li>
  <li><a href="/news/5">Five</a></li>
  <li><a href="/news/6">Six</a></li>
</ul>`)

	items, err := collectItemsFromHTML("https://example.com/", "ul.news", html)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 6 {
		t.Fatalf("len(items) = %d, want 6", len(items))
	}
	if items[5].Link != "https://example.com/news/6" {
		t.Fatalf("items[5].Link = %q", items[5].Link)
	}
}

func TestBuildRSS(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	buildTime := time.Date(2026, 6, 29, 10, 20, 30, 0, jst)
	rss := buildRSS(ChannelSettings{
		Title:       "Test Feed",
		Link:        "https://feed.example.com/",
		Description: "Test Description",
	}, []FeedItem{
		{Title: "Entry", Date: "2026.6.26", Link: "https://example.com/entry"},
		{Title: "No Date", Link: "https://example.com/no-date"},
		{Title: "Latest Entry", Date: "2026.6.28", Link: "https://example.com/latest"},
	}, buildTime, 0)

	if rss.Version != "2.0" {
		t.Fatalf("rss.Version = %q", rss.Version)
	}
	if len(rss.Channel.Items) != 3 {
		t.Fatalf("len(rss.Channel.Items) = %d, want 3", len(rss.Channel.Items))
	}
	if rss.Channel.Link != "https://feed.example.com/" {
		t.Fatalf("rss.Channel.Link = %q", rss.Channel.Link)
	}
	if rss.Channel.Title != "Test Feed" {
		t.Fatalf("rss.Channel.Title = %q", rss.Channel.Title)
	}
	if rss.Channel.Description != "Test Description" {
		t.Fatalf("rss.Channel.Description = %q", rss.Channel.Description)
	}
	if rss.Channel.LastBuildDate != "Mon, 29 Jun 2026 10:20:30 +0900" {
		t.Fatalf("LastBuildDate = %q", rss.Channel.LastBuildDate)
	}
	if rss.Channel.Items[0].PubDate != "Sun, 28 Jun 2026 00:00:00 +0900" {
		t.Fatalf("PubDate = %q", rss.Channel.Items[0].PubDate)
	}
	if rss.Channel.Items[0].GUID.Value != "https://example.com/latest" {
		t.Fatalf("GUID.Value = %q", rss.Channel.Items[0].GUID.Value)
	}
	if rss.Channel.Items[1].PubDate != "Fri, 26 Jun 2026 00:00:00 +0900" {
		t.Fatalf("second PubDate = %q", rss.Channel.Items[1].PubDate)
	}
	if rss.Channel.Items[2].PubDate != "" {
		t.Fatalf("PubDate without date = %q", rss.Channel.Items[2].PubDate)
	}
}

func TestBuildRSSLimitsItemsAfterSorting(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	buildTime := time.Date(2026, 6, 29, 10, 20, 30, 0, jst)
	rss := buildRSS(ChannelSettings{
		Title:       "Test Feed",
		Link:        "https://feed.example.com/",
		Description: "Test Description",
	}, []FeedItem{
		{Title: "Old", Date: "2026.6.20", Link: "https://example.com/old"},
		{Title: "Latest", Date: "2026.6.28", Link: "https://example.com/latest"},
		{Title: "Middle", Date: "2026.6.26", Link: "https://example.com/middle"},
		{Title: "No Date", Link: "https://example.com/no-date"},
	}, buildTime, 2)

	if len(rss.Channel.Items) != 2 {
		t.Fatalf("len(rss.Channel.Items) = %d, want 2", len(rss.Channel.Items))
	}
	if rss.Channel.Items[0].Link != "https://example.com/latest" {
		t.Fatalf("first item link = %q", rss.Channel.Items[0].Link)
	}
	if rss.Channel.Items[1].Link != "https://example.com/middle" {
		t.Fatalf("second item link = %q", rss.Channel.Items[1].Link)
	}
}

func TestFormatItemTitle(t *testing.T) {
	item := FeedItem{
		Title: "オフィシャル グッズ公開！",
		Date:  "2026.6.9",
	}
	got := formatItemTitle("L'arc-en-Ciel NEWS", item)
	want := "L'arc-en-Ciel NEWS - 2026.6.9 : オフィシャル グッズ公開！"
	if got != want {
		t.Fatalf("formatItemTitle() = %q, want %q", got, want)
	}

	item.Date = ""
	got = formatItemTitle("L'arc-en-Ciel NEWS", item)
	want = "L'arc-en-Ciel NEWS - オフィシャル グッズ公開！"
	if got != want {
		t.Fatalf("formatItemTitle() without date = %q, want %q", got, want)
	}
}

func TestLoadConfig(t *testing.T) {
	path := t.TempDir() + "/config.yml"
	err := os.WriteFile(path, []byte(`settings:
  channel:
    title: Combined Feed
    link: https://feed.example.com/
    description: Combined description
  limit: 20

feeds:
  - title: Example Feed
    url: https://example.com/news/
    xpath: ul.news_list
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	configFile, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if configFile.Settings.Channel.Title != "Combined Feed" {
		t.Fatalf("channel.Title = %q", configFile.Settings.Channel.Title)
	}
	if configFile.Settings.Channel.Link != "https://feed.example.com/" {
		t.Fatalf("channel.Link = %q", configFile.Settings.Channel.Link)
	}
	if configFile.Settings.Channel.Description != "Combined description" {
		t.Fatalf("channel.Description = %q", configFile.Settings.Channel.Description)
	}
	if configFile.Settings.Limit != 20 {
		t.Fatalf("settings.Limit = %d, want 20", configFile.Settings.Limit)
	}
	if len(configFile.Feeds) != 1 {
		t.Fatalf("len(feeds) = %d, want 1", len(configFile.Feeds))
	}
	if configFile.Feeds[0].Title != "Example Feed" {
		t.Fatalf("feeds[0].Title = %q", configFile.Feeds[0].Title)
	}
	if configFile.Feeds[0].URL != "https://example.com/news/" {
		t.Fatalf("feeds[0].URL = %q", configFile.Feeds[0].URL)
	}
	if configFile.Feeds[0].XPath != "ul.news_list" {
		t.Fatalf("feeds[0].XPath = %q", configFile.Feeds[0].XPath)
	}
}

func TestCollectAllItemsMergesFeedsAndRemovesDuplicates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first":
			_, _ = w.Write([]byte(`<ul class="news"><li><a href="/shared">Shared</a></li><li><a href="/one">One</a></li></ul>`))
		case "/second":
			_, _ = w.Write([]byte(`<ul class="news"><li><a href="/shared">Shared Again</a></li><li><a href="/two">Two</a></li></ul>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	items, err := collectAllItems(context.Background(), server.Client(), []Config{
		{Title: "First", URL: server.URL + "/first", XPath: "ul.news"},
		{Title: "Second", URL: server.URL + "/second", XPath: "ul.news"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if items[0].Link != server.URL+"/shared" {
		t.Fatalf("items[0].Link = %q", items[0].Link)
	}
	if items[2].Link != server.URL+"/two" {
		t.Fatalf("items[2].Link = %q", items[2].Link)
	}
}

func TestLoadConfigRequiresChannelSettings(t *testing.T) {
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

	_, err = loadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "settings.channel.title") {
		t.Fatalf("loadConfig() error = %v", err)
	}
}

func TestRunWritesMergedFeedToOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first":
			_, _ = w.Write([]byte(`<ul class="news"><li><a href="/one">One</a></li></ul>`))
		case "/second":
			_, _ = w.Write([]byte(`<ul class="news"><li><a href="/two">Two</a></li></ul>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	configPath := tempDir + "/config.yml"
	outputPath := tempDir + "/feed.xml"
	config := `settings:
  channel:
    title: Combined Feed
    link: https://feed.example.com/
    description: Combined description

feeds:
  - title: First
    url: ` + server.URL + `/first
    xpath: ul.news
  - title: Second
    url: ` + server.URL + `/second
    xpath: ul.news
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout strings.Builder
	if err := run([]string{"-config", configPath, "-output", outputPath}, &stdout); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	output := string(body)
	if strings.Count(output, "<rss ") != 1 {
		t.Fatalf("rss count = %d, want 1", strings.Count(output, "<rss "))
	}
	if strings.Count(output, "<item>") != 2 {
		t.Fatalf("item count = %d, want 2", strings.Count(output, "<item>"))
	}
}

func TestNormalizeXPath(t *testing.T) {
	xpath := normalizeXPath("ul.news_list")
	if !strings.Contains(xpath, "news_list") {
		t.Fatalf("xpath = %q", xpath)
	}
	classXPath := normalizeXPath(".news-index")
	if classXPath != "//*[contains(concat(' ', normalize-space(@class), ' '), ' news-index ')]" {
		t.Fatalf("class xpath = %q", classXPath)
	}
	if got := normalizeXPath("//ul"); got != "//ul" {
		t.Fatalf("normalizeXPath xpath = %q", got)
	}
}
