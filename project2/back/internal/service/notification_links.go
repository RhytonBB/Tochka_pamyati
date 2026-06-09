package service

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func profileEditPostLink(postID uuid.UUID) string {
	return fmt.Sprintf("/submission-edit/post/%s", postID.String())
}

func profileEditMonumentLink(monumentID uuid.UUID) string {
	return fmt.Sprintf("/submission-edit/monument/%s", monumentID.String())
}

func signalLink(signalID uuid.UUID) string {
	return fmt.Sprintf("/signal/%s", signalID.String())
}

func monumentLink(monumentID uuid.UUID) string {
	return fmt.Sprintf("/monument/%s", monumentID.String())
}

func notificationsLink(filter string) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "/notifications"
	}
	return "/notifications?filter=" + filter
}
