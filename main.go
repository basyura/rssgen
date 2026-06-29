package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

const maxItemsPerFeed = 5

type ConfigFile struct {
	Settings Settings `yaml:"settings"`
	Feeds    []Config `yaml:"feeds"`
}

type Settings struct {
	Channel ChannelSettings `yaml:"channel"`
}

type ChannelSettings struct {
	Title       string `yaml:"title"`
	Link        string `yaml:"link"`
	Description string `yaml:"description"`
}

type Config struct {
	URL   string `yaml:"url"`
	Title string `yaml:"title"`
	XPath string `yaml:"xpath"`
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	Description   string `xml:"description"`
	LastBuildDate string `xml:"lastBuildDate,omitempty"`
	Items         []Item `xml:"item"`
}

type Item struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate,omitempty"`
	GUID    GUID   `xml:"guid"`
}

type GUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type FeedItem struct {
	Title string
	Date  string
	Link  string
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("rssgen", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	configPath := flags.String("config", "", "path to config YAML")
	outputPath := flags.String("output", "", "path to output RSS XML")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *configPath == "" {
		return errors.New("missing required -config")
	}

	configFile, err := loadConfig(*configPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	items, err := collectAllItems(ctx, http.DefaultClient, configFile.Feeds)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := writeRSS(&buf, buildRSS(configFile.Settings.Channel, items, time.Now())); err != nil {
		return err
	}

	if *outputPath == "" {
		_, err = stdout.Write(buf.Bytes())
		return err
	}
	return os.WriteFile(*outputPath, buf.Bytes(), 0o644)
}

func loadConfig(path string) (ConfigFile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return ConfigFile{}, err
	}

	var configFile ConfigFile
	if err := yaml.Unmarshal(body, &configFile); err != nil {
		return ConfigFile{}, err
	}
	channel := configFile.Settings.Channel
	if strings.TrimSpace(channel.Title) == "" {
		return ConfigFile{}, errors.New("settings.channel.title is required")
	}
	if strings.TrimSpace(channel.Link) == "" {
		return ConfigFile{}, errors.New("settings.channel.link is required")
	}
	if strings.TrimSpace(channel.Description) == "" {
		return ConfigFile{}, errors.New("settings.channel.description is required")
	}
	if len(configFile.Feeds) == 0 {
		return ConfigFile{}, errors.New("config feeds must contain at least one entry")
	}
	for i, cfg := range configFile.Feeds {
		if strings.TrimSpace(cfg.Title) == "" {
			return ConfigFile{}, fmt.Errorf("feeds[%d].title is required", i)
		}
		if strings.TrimSpace(cfg.URL) == "" {
			return ConfigFile{}, fmt.Errorf("feeds[%d].url is required", i)
		}
		if strings.TrimSpace(cfg.XPath) == "" {
			return ConfigFile{}, fmt.Errorf("feeds[%d].xpath is required", i)
		}
	}
	return configFile, nil
}

func collectAllItems(ctx context.Context, client *http.Client, configs []Config) ([]FeedItem, error) {
	seen := map[string]struct{}{}
	var items []FeedItem
	for _, cfg := range configs {
		sourceItems, err := collectItems(ctx, client, cfg)
		if err != nil {
			return nil, fmt.Errorf("collect %s: %w", cfg.Title, err)
		}
		for _, item := range sourceItems {
			if _, ok := seen[item.Link]; ok {
				continue
			}
			seen[item.Link] = struct{}{}
			items = append(items, item)
		}
	}
	return items, nil
}

func collectItems(ctx context.Context, client *http.Client, cfg Config) ([]FeedItem, error) {
	body, err := fetch(ctx, client, cfg.URL)
	if err != nil {
		return nil, err
	}

	items, err := collectItemsFromHTML(cfg.URL, cfg.XPath, body)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Title = formatItemTitle(cfg.Title, items[i])
	}
	return items, nil
}

func fetch(ctx context.Context, client *http.Client, sourceURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "rssgen/0.1")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: unexpected status %s", sourceURL, res.Status)
	}
	return io.ReadAll(res.Body)
}

func collectItemsFromHTML(sourceURL, selector string, body []byte) ([]FeedItem, error) {
	doc, err := htmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	xpath := normalizeXPath(selector)
	parents := htmlquery.Find(doc, xpath)
	if len(parents) == 0 {
		return nil, fmt.Errorf("no elements matched xpath %q", selector)
	}

	baseURL, err := url.Parse(sourceURL)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	items := make([]FeedItem, 0, maxItemsPerFeed)
parentsLoop:
	for _, parent := range parents {
		for child := parent.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != html.ElementNode {
				continue
			}
			linkNode := child
			if child.Data != "a" || htmlquery.SelectAttr(child, "href") == "" {
				linkNode = htmlquery.FindOne(child, ".//a[@href]")
			}
			if linkNode == nil {
				continue
			}
			href := htmlquery.SelectAttr(linkNode, "href")
			link, err := resolveURL(baseURL, href)
			if err != nil {
				continue
			}
			if _, ok := seen[link]; ok {
				continue
			}
			date := extractText(linkNode,
				".//*[contains(concat(' ', normalize-space(@class), ' '), ' date ')]",
				".//h5",
			)
			title := extractText(linkNode,
				".//*[contains(concat(' ', normalize-space(@class), ' '), ' comment ')]",
				".//*[contains(concat(' ', normalize-space(@class), ' '), ' NW-R ')]//p",
			)
			if title == "" {
				title = strings.Join(strings.Fields(htmlquery.InnerText(linkNode)), " ")
			}
			if title == "" {
				title = link
			}
			seen[link] = struct{}{}
			items = append(items, FeedItem{Title: title, Date: date, Link: link})
			if len(items) == maxItemsPerFeed {
				break parentsLoop
			}
		}
	}
	return items, nil
}

func extractText(node *html.Node, xpaths ...string) string {
	for _, xpath := range xpaths {
		textNode := htmlquery.FindOne(node, xpath)
		if textNode == nil {
			continue
		}
		text := strings.Join(strings.Fields(htmlquery.InnerText(textNode)), " ")
		if text != "" {
			return text
		}
	}
	return ""
}

func formatItemTitle(sourceTitle string, item FeedItem) string {
	sourceTitle = strings.TrimSpace(sourceTitle)
	if item.Date == "" {
		return fmt.Sprintf("%s - %s", sourceTitle, item.Title)
	}
	return fmt.Sprintf("%s - %s : %s", sourceTitle, item.Date, item.Title)
}

func normalizeXPath(selector string) string {
	selector = strings.TrimSpace(selector)
	if strings.HasPrefix(selector, "/") || strings.HasPrefix(selector, "(") {
		return selector
	}

	tag, className, ok := strings.Cut(selector, ".")
	if !ok || className == "" || strings.ContainsAny(className, " .#>/[") {
		return selector
	}
	if tag == "" {
		return fmt.Sprintf("//*[contains(concat(' ', normalize-space(@class), ' '), ' %s ')]", className)
	}
	return fmt.Sprintf("//%s[contains(concat(' ', normalize-space(@class), ' '), ' %s ')]", tag, className)
}

func resolveURL(base *url.URL, href string) (string, error) {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return "", errors.New("empty href")
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(parsed).String(), nil
}

func buildRSS(channel ChannelSettings, items []FeedItem, buildTime time.Time) RSS {
	rssItems := make([]Item, 0, len(items))
	for _, item := range items {
		rssItem := Item{
			Title: item.Title,
			Link:  item.Link,
			GUID: GUID{
				IsPermaLink: "true",
				Value:       item.Link,
			},
		}
		if pubDate, ok := parseFeedItemDate(item.Date, buildTime.Location()); ok {
			rssItem.PubDate = pubDate.Format(time.RFC1123Z)
		}
		rssItems = append(rssItems, rssItem)
	}

	return RSS{
		Version: "2.0",
		Channel: Channel{
			Title:         channel.Title,
			Link:          channel.Link,
			Description:   channel.Description,
			LastBuildDate: buildTime.Format(time.RFC1123Z),
			Items:         rssItems,
		},
	}
}

func parseFeedItemDate(value string, loc *time.Location) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	if loc == nil {
		loc = time.Local
	}
	parsed, err := time.ParseInLocation("2006.1.2", value, loc)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func writeRSS(w io.Writer, rss RSS) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(rss); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}
