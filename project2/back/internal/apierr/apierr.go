package apierr

import "net/http"

type Error struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Data    map[string]any    `json:"data,omitempty"`
}

func (e Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func Status(code string) int {
	switch code {
	case "validation_failed":
		return http.StatusBadRequest
	case "content_requires_ack":
		return http.StatusUnprocessableEntity
	case "email_already_used", "username_already_used":
		return http.StatusConflict
	case "invalid_credentials":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "email_not_verified":
		return http.StatusForbidden
	case "account_blocked":
		return http.StatusForbidden
	case "rate_limited":
		return http.StatusTooManyRequests
	case "verification_expired", "verification_invalid":
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
	}
}
