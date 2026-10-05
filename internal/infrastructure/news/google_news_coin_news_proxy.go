package news

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// maximumFeedBytes bounds one news feed; a search feed is far smaller.
const maximumFeedBytes = 4 << 20

type googleNewsFeedWire struct {
	Items []struct {
		Title   string `xml:"title"`
		PubDate string `xml:"pubDate"`
		Source  string `xml:"source"`
	} `xml:"channel>item"`
}

// GoogleNewsCoinNewsProxy searches Google News' free RSS for a coin's recent headlines.
type GoogleNewsCoinNewsProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewGoogleNewsCoinNewsProxy(httpClient *http.Client, baseUrl string) *GoogleNewsCoinNewsProxy {
	return &GoogleNewsCoinNewsProxy{httpClient: httpClient, baseUrl: baseUrl}
}

// FindRecentHeadlines searches `"SYMBOL" crypto` within whole days back to since, then keeps items published since then.
func (googleNewsCoinNewsProxy *GoogleNewsCoinNewsProxy) FindRecentHeadlines(
	executionContext context.Context, coinSymbol string, since time.Time, limit int,
) ([]vo.HeadlineVo, error) {
	// Truncated to the minute so "72 hours ago" asked a moment later still searches three days, not four.
	lookbackDays := max(1, int(math.Ceil(time.Since(since).Truncate(time.Minute).Hours()/24)))
	query := url.Values{
		"q":    {fmt.Sprintf(`"%s" crypto when:%dd`, coinSymbol, lookbackDays)},
		"hl":   {"en-US"},
		"gl":   {"US"},
		"ceid": {"US:en"},
	}
	request, requestError := http.NewRequestWithContext(executionContext, http.MethodGet,
		googleNewsCoinNewsProxy.baseUrl+"/rss/search?"+query.Encode(), nil)
	if requestError != nil {
		return nil, fmt.Errorf("build news request: %w", requestError)
	}
	request.Header.Set("User-Agent", "go-coin-hunter/1.0")
	response, responseError := googleNewsCoinNewsProxy.httpClient.Do(request)
	if responseError != nil {
		return nil, fmt.Errorf("request news: %w", responseError)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request news: unexpected status %d", response.StatusCode)
	}

	feed := googleNewsFeedWire{}
	if decodeError := xml.NewDecoder(io.LimitReader(response.Body, maximumFeedBytes)).Decode(&feed); decodeError != nil {
		return nil, fmt.Errorf("decode news feed: %w", decodeError)
	}

	headlines := []vo.HeadlineVo{}
	for _, item := range feed.Items {
		publishedAt, parseError := time.Parse(time.RFC1123, item.PubDate)
		if parseError != nil || publishedAt.Before(since) {
			continue
		}
		sourceName := strings.TrimSpace(item.Source)
		if sourceName == "" {
			sourceName = "Google News"
		}
		headlines = append(headlines, vo.HeadlineVo{SourceName: sourceName, Title: strings.TrimSpace(item.Title), PublishedAt: publishedAt.UTC()})
	}
	slices.SortStableFunc(headlines, func(left, right vo.HeadlineVo) int {
		return right.PublishedAt.Compare(left.PublishedAt)
	})

	return headlines[:min(limit, len(headlines))], nil
}
