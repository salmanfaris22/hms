package service

import (
	"context"

	"github.com/salman/hms-backend/internal/modules/logs/model"
	"github.com/salman/hms-backend/internal/modules/logs/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f model.ListFilter) (model.ListResult, error) {
	total, err := s.repo.Count(ctx, f)
	if err != nil {
		return model.ListResult{}, err
	}
	logs, err := s.repo.List(ctx, f)
	if err != nil {
		return model.ListResult{}, err
	}
	users, _ := s.repo.DistinctActors(ctx, f.TenantID)
	return model.ListResult{
		Logs:     logs,
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
		Users:    users,
	}, nil
}

func (s *Service) Export(ctx context.Context, f model.ExportFilter) ([]model.ExportRow, error) {
	return s.repo.ExportQuery(ctx, f)
}
