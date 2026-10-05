package news_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleNewsKeepsRecentHeadlinesNewestFirst(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	rfc := func(at time.Time) string { return at.Format(time.RFC1123) }
	query := ""
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query = request.URL.Query().Get("q")
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><rss><channel>
			<item><title>Older PENGU news</title><pubDate>` + rfc(now.Add(-5*time.Hour)) + `</pubDate><source url="x">CoinDesk</source></item>
			<item><title> Newest PENGU news </title><pubDate>` + rfc(now.Add(-time.Hour)) + `</pubDate><source url="y"></source></item>
			<item><title>Stale PENGU news</title><pubDate>` + rfc(now.Add(-100*time.Hour)) + `</pubDate></item>
			<item><title>Undated</title><pubDate>yesterday</pubDate></item>
			<item><title>Middle PENGU news</title><pubDate>` + rfc(now.Add(-3*time.Hour)) + `</pubDate><source>The Block</source></item>
		</channel></rss>`))
	}))
	t.Cleanup(server.Close)

	headlines, findError := news.NewGoogleNewsCoinNewsProxy(http.DefaultClient, server.URL).
		FindRecentHeadlines(context.Background(), "PENGU", now.Add(-72*time.Hour), 2)

	require.NoError(t, findError)
	assert.Equal(t, `"PENGU" crypto when:3d`, query)
	assert.Equal(t, []vo.HeadlineVo{
		{SourceName: "Google News", Title: "Newest PENGU news", PublishedAt: now.Add(-time.Hour)},
		{SourceName: "The Block", Title: "Middle PENGU news", PublishedAt: now.Add(-3 * time.Hour)},
	}, headlines)
}

func TestGoogleNewsFailsOnBadAnswers(t *testing.T) {
	testCases := []struct {
		name    string
		baseUrl func(t *testing.T) string
	}{
		{name: "an error status", baseUrl: func(t *testing.T) string {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusServiceUnavailable) }))
			t.Cleanup(server.Close)
			return server.URL
		}},
		{name: "a body that is not a feed", baseUrl: func(t *testing.T) string {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write([]byte("<rss><channel>")) }))
			t.Cleanup(server.Close)
			return server.URL
		}},
		{name: "a refused connection", baseUrl: func(*testing.T) string { return "http://127.0.0.1:1" }},
		{name: "an unparseable address", baseUrl: func(*testing.T) string { return "http://bad host" }},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, findError := news.NewGoogleNewsCoinNewsProxy(http.DefaultClient, testCase.baseUrl(t)).
				FindRecentHeadlines(context.Background(), "PENGU", time.Now().Add(-time.Hour), 10)

			assert.Error(t, findError)
		})
	}
}
