package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// maximumResponseBytes bounds one source response so a misbehaving source cannot exhaust memory.
const maximumResponseBytes = 8 << 20

// getJson is the one way every source is read; the type parameter is each source's own wire shape.
func getJson[T any](executionContext context.Context, httpClient *http.Client, requestUrl string) (T, error) {
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
		return decoded, fmt.Errorf("request %s: unexpected status %d", request.URL.Host, response.StatusCode)
	}
	if decodeError := json.NewDecoder(io.LimitReader(response.Body, maximumResponseBytes)).Decode(&decoded); decodeError != nil {
		return decoded, fmt.Errorf("decode %s response: %w", request.URL.Host, decodeError)
	}

	return decoded, nil
}
