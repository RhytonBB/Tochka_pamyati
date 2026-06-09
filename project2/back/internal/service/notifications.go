package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
)

type NotificationsService struct {
	repo *repo.Notifications
}

func NewNotificationsService(r *repo.Notifications) *NotificationsService {
	return &NotificationsService{repo: r}
}

func (s *NotificationsService) ListLatest(ctx context.Context, userID uuid.UUID, limit int) ([]repo.Notification, error) {
	return s.repo.ListLatest(ctx, userID, limit)
}

func (s *NotificationsService) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

func (s *NotificationsService) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *NotificationsService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}
