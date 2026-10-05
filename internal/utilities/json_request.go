package utilities

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// maximumResponseBytes bounds one response so a misbehaving source cannot exhaust memory; CoinGecko's full coin list
// with contract addresses is about 4 MB and growing.
const maximumResponseBytes = 32 << 20

// GetJson is how every external source is read: one GET, a 200 required, the body decoded into the source's own wire
// shape. It is the only shared piece of the proxies, kept here because it knows nothing about any domain.
func GetJson[T any](executionContext context.Context, httpClient *http.Client, requestUrl string) (T, error) {
	decoded := *new(T)
	request, requestError := http.NewRequestWithContext(executionContext, http.MethodGet, requestUrl, nil)
	if requestError != nil {
		return decoded, fmt.Errorf("build request: %w", requestError)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "go-coin-hunter/1.0")

	response, responseError := httpClient.Do(request)
	if responseError != nil {
		return decoded, fmt.Errorf("request %s: %w", request.URL.Host, responseError)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return decoded, fmt.Errorf("request %s: %w", request.URL.Host, HttpStatusError{StatusCode: response.StatusCode})
	}
	if decodeError := json.NewDecoder(io.LimitReader(response.Body, maximumResponseBytes)).Decode(&decoded); decodeError != nil {
		return decoded, fmt.Errorf("decode %s response: %w", request.URL.Host, decodeError)
	}

	return decoded, nil
}
