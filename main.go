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

type ConfigFile struct {
	Settings Settings `yaml:"settings"`
	Feeds    []Config `yaml:"feeds"`
}

type Settings struct {
	Channel ChannelSettings `yaml:"channel"`
}

type ChannelSettings struct {
	Link string `yaml:"link"`
}

type Config struct {
	URL   string `yaml:"url"`
	Title string `yaml:"title"`
	Link  string `yaml:"-"`
	XPath string `yaml:"xpath"`
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
	GUID  GUID   `xml:"guid"`
}

type GUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type FeedItem struct {
	Title string
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

	configs, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	if *outputPath != "" && len(configs) > 1 {
		return errors.New("-output can only be used with a single config entry")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var buf bytes.Buffer
	for i, cfg := range configs {
		items, err := collectItems(ctx, http.DefaultClient, cfg)
		if err != nil {
			return err
		}
		if i > 0 {
			buf.WriteByte('\n')
		}
		if err := writeRSS(&buf, buildRSS(cfg, items)); err != nil {
			return err
		}
	}

	if *outputPath == "" {
		_, err = stdout.Write(buf.Bytes())
		return err
	}
	return os.WriteFile(*outputPath, buf.Bytes(), 0o644)
}

func loadConfig(path string) ([]Config, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var configFile ConfigFile
	if err := yaml.Unmarshal(body, &configFile); err != nil {
		return nil, err
	}
	configs := configFile.Feeds
	if len(configs) == 0 {
		return nil, errors.New("config feeds must contain at least one entry")
	}
	for i, cfg := range configs {
		if strings.TrimSpace(cfg.Title) == "" {
			return nil, fmt.Errorf("feeds[%d].title is required", i)
		}
		if strings.TrimSpace(cfg.URL) == "" {
			return nil, fmt.Errorf("feeds[%d].url is required", i)
		}
		if strings.TrimSpace(cfg.XPath) == "" {
			return nil, fmt.Errorf("feeds[%d].xpath is required", i)
		}
		configs[i].Link = configFile.Settings.Channel.Link
	}
	return configs, nil
}

func collectItems(ctx context.Context, client *http.Client, cfg Config) ([]FeedItem, error) {
	seen := map[string]struct{}{}
	var items []FeedItem
	body, err := fetch(ctx, client, cfg.URL)
	if err != nil {
		return nil, err
	}

	sourceItems, err := collectItemsFromHTML(cfg.URL, cfg.XPath, body)
	if err != nil {
		return nil, err
	}
	for _, item := range sourceItems {
		if _, ok := seen[item.Link]; ok {
			continue
		}
		seen[item.Link] = struct{}{}
		items = append(items, item)
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

	var items []FeedItem
	for _, parent := range parents {
		for child := parent.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != html.ElementNode {
				continue
			}
			linkNode := htmlquery.FindOne(child, ".//a[@href]")
			if linkNode == nil {
				continue
			}
			href := htmlquery.SelectAttr(linkNode, "href")
			link, err := resolveURL(baseURL, href)
			if err != nil {
				continue
			}
			title := strings.Join(strings.Fields(htmlquery.InnerText(linkNode)), " ")
			if title == "" {
				title = link
			}
			items = append(items, FeedItem{Title: title, Link: link})
		}
	}
	return items, nil
}

func normalizeXPath(selector string) string {
	selector = strings.TrimSpace(selector)
	if strings.HasPrefix(selector, "/") || strings.HasPrefix(selector, "(") {
		return selector
	}

	tag, className, ok := strings.Cut(selector, ".")
	if ok && tag != "" && className != "" && !strings.ContainsAny(className, " .#>/[") {
		return fmt.Sprintf("//%s[contains(concat(' ', normalize-space(@class), ' '), ' %s ')]", tag, className)
	}
	return selector
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

func buildRSS(cfg Config, items []FeedItem) RSS {
	rssItems := make([]Item, 0, len(items))
	for _, item := range items {
		rssItems = append(rssItems, Item{
			Title: item.Title,
			Link:  item.Link,
			GUID: GUID{
				IsPermaLink: "true",
				Value:       item.Link,
			},
		})
	}

	channelLink := strings.TrimSpace(cfg.Link)
	if channelLink == "" {
		channelLink = cfg.URL
	}

	return RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       cfg.Title,
			Link:        channelLink,
			Description: cfg.Title,
			Items:       rssItems,
		},
	}
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
