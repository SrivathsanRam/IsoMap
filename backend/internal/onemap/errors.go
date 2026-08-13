package onemap

import "errors"

var (
	ErrNotConfigured   = errors.New("onemap authentication is not configured")
	ErrAuthentication  = errors.New("onemap authentication failed")
	ErrRateLimited     = errors.New("onemap rate limit exceeded")
	ErrUnavailable     = errors.New("onemap is unavailable")
	ErrInvalidResponse = errors.New("onemap returned an invalid response")
)
