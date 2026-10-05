package utilities

import "fmt"

// HttpStatusError is a source answering with something other than 200, so callers can tell "not there" from "down".
type HttpStatusError struct {
	StatusCode int
}

func (httpStatusError HttpStatusError) Error() string {
	return fmt.Sprintf("unexpected status %d", httpStatusError.StatusCode)
}
