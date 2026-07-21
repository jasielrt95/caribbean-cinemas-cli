package caribbeancinemas

import (
	"errors"
	"fmt"
)

// API error codes observed from the backend. The GraphQL server returns a
// non-standard error shape ({"error": {"message", "code"}}) rather than the
// usual GraphQL "errors" array.
const (
	// CodeValidation is a schema/validation error (unknown field, missing or
	// bad argument).
	CodeValidation = 210
	// CodeUnauthenticated means the query requires a logged-in customer
	// session. Such queries are intentionally not wrapped by this library.
	CodeUnauthenticated = 101
	// CodeForbidden means the caller lacks permission for the operation.
	CodeForbidden = 102
)

// APIError is returned when the GraphQL endpoint responds with an error.
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("caribbeancinemas: API error %d: %s", e.Code, e.Message)
	}
	return "caribbeancinemas: " + e.Message
}

// ErrNotFound is returned when a lookup (e.g. a movie or theater by ID) yields
// no result.
var ErrNotFound = errors.New("caribbeancinemas: not found")

// IsAuthRequired reports whether err is an APIError indicating the underlying
// query needs a customer login (and therefore is not available to this
// read-only client).
func IsAuthRequired(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == CodeUnauthenticated || apiErr.Code == CodeForbidden
	}
	return false
}
