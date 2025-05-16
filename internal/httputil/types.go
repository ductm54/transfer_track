// Package httputil provides HTTP utility functions and types.
package httputil

// CommonError represents a common error response format.
type CommonError struct {
	Error string `json:"error"`
}
