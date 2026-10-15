package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	SearchViewAll     = "all"
	SearchViewStarred = "starred"
	SearchViewTrash   = "trash"
	MaxSearchQueryLen = 200
	MaxSearchResults  = 100
)

type SearchItemType string

const (
	SearchItemTypeFolder SearchItemType = "folder"
	SearchItemTypeScene  SearchItemType = "scene"
)

type SearchItem struct {
	ID          uuid.UUID      `json:"id"`
	WorkspaceID uuid.UUID      `json:"workspaceId"`
	Type        SearchItemType `json:"type"`
	Name        string         `json:"name"`
	FolderID    *uuid.UUID     `json:"folderId,omitempty"`
	IsStarred   bool           `json:"isStarred"`
	DeletedAt   *time.Time     `json:"deletedAt,omitempty"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type SearchRepository interface {
	Search(ctx Context, workspaceID uuid.UUID, query string, view string, limit int) ([]SearchItem, error)
}

type SearchUsecase interface {
	Search(ctx context.Context, workspaceID uuid.UUID, query string, view string) ([]SearchItem, error)
}
