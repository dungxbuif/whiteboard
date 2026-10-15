package usecase

import (
	"context"
	"fmt"
	"strings"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type searchUsecase struct {
	repo domain.SearchRepository
}

func NewSearchUsecase(repo domain.SearchRepository) domain.SearchUsecase {
	return &searchUsecase{repo: repo}
}

func (u *searchUsecase) Search(ctx context.Context, workspaceID uuid.UUID, query string, view string) ([]domain.SearchItem, error) {
	query = strings.TrimSpace(query)
	if len(query) > domain.MaxSearchQueryLen {
		return nil, fmt.Errorf("search query must be at most %d characters", domain.MaxSearchQueryLen)
	}
	if view == "" {
		view = domain.SearchViewAll
	}
	if view != domain.SearchViewAll && view != domain.SearchViewStarred && view != domain.SearchViewTrash {
		return nil, fmt.Errorf("invalid search view")
	}
	return u.repo.Search(ctx, workspaceID, query, view, domain.MaxSearchResults)
}
