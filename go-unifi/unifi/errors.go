package unifi

import "fmt"

type LoginRequiredError struct {
	APIKey bool // true when the rejection is for an API-key request
}

func (err *LoginRequiredError) Error() string {
	if err.APIKey {
		return "API key rejected (HTTP 401): check that the key is valid and has not been revoked"
	}
	return "login required"
}

type NotFoundError struct {
	Type  string
	Attr  string
	Value string
}

func (err *NotFoundError) Error() string {
	if err.Attr != "" && err.Value != "" {
		return fmt.Sprintf("not found: type=%s, attr=%s, value=%s", err.Type, err.Attr, err.Value)
	} else {
		return fmt.Sprintf("not found: type=%s", err.Type)
	}
}

type APIError struct {
	RC      string
	Message string
}

func (err *APIError) Error() string {
	return err.Message
}
