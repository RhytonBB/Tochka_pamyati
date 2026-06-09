package service

import (
	"context"

	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
)

type AnalyticsService struct {
	monuments *repo.Monuments
	posts     *repo.Posts
	users     *repo.Users
	signals   *repo.Signals
	reports   *repo.Reports
}

func NewAnalyticsService(monuments *repo.Monuments, posts *repo.Posts, users *repo.Users, signals *repo.Signals, reports *repo.Reports) *AnalyticsService {
	return &AnalyticsService{
		monuments: monuments,
		posts:     posts,
		users:     users,
		signals:   signals,
		reports:   reports,
	}
}

type GlobalStats struct {
	Monuments map[string]any   `json:"monuments"`
	Posts     map[string]int64 `json:"posts"`
	Users     map[string]int64 `json:"users"`
	Signals   map[string]any   `json:"signals"`
	Reports   map[string]int64 `json:"reports"`
}

func (s *AnalyticsService) GetGlobalStats(ctx context.Context) (GlobalStats, error) {
	monStats, err := s.monuments.GetStats(ctx)
	if err != nil {
		return GlobalStats{}, err
	}
	postStats, err := s.posts.GetStats(ctx)
	if err != nil {
		return GlobalStats{}, err
	}
	userStats, err := s.users.GetStats(ctx)
	if err != nil {
		return GlobalStats{}, err
	}
	sigStats, err := s.signals.GetStats(ctx)
	if err != nil {
		return GlobalStats{}, err
	}
	reportStats := map[string]int64{}
	if s.reports != nil {
		reportStats, err = s.reports.CountCaseStats(ctx)
		if err != nil {
			return GlobalStats{}, err
		}
	}

	return GlobalStats{
		Monuments: monStats,
		Posts:     postStats,
		Users:     userStats,
		Signals:   sigStats,
		Reports:   reportStats,
	}, nil
}

type Dynamics struct {
	Monuments []map[string]any `json:"monuments"`
	Posts     []map[string]any `json:"posts"`
	Users     []map[string]any `json:"users"`
	Signals   []map[string]any `json:"signals"`
}

func (s *AnalyticsService) GetDynamics(ctx context.Context, days int) (Dynamics, error) {
	if days <= 0 {
		days = 30
	}
	monDyn, err := s.monuments.GetDynamics(ctx, days)
	if err != nil {
		return Dynamics{}, err
	}
	postDyn, err := s.posts.GetDynamics(ctx, days)
	if err != nil {
		return Dynamics{}, err
	}
	userDyn, err := s.users.GetDynamics(ctx, days)
	if err != nil {
		return Dynamics{}, err
	}
	sigDyn, err := s.signals.GetDynamics(ctx, days)
	if err != nil {
		return Dynamics{}, err
	}

	return Dynamics{
		Monuments: monDyn,
		Posts:     postDyn,
		Users:     userDyn,
		Signals:   sigDyn,
	}, nil
}

func (s *AnalyticsService) GetTopAuthors(ctx context.Context, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.posts.GetTopAuthors(ctx, limit)
}
