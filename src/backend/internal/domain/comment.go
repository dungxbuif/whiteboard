package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	CommentStatusOpen     = "open"
	CommentStatusResolved = "resolved"
)

var (
	ErrInvalidCommentInput = errors.New("invalid comment input")
	ErrCommentRateLimited  = errors.New("comment write rate limited")
)

type CommentAnchor struct {
	ElementID *string `json:"elementId,omitempty"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
}

type Comment struct {
	ID          uuid.UUID     `json:"id"`
	WorkspaceID uuid.UUID     `json:"-"`
	SceneID     uuid.UUID     `json:"-"`
	ThreadID    uuid.UUID     `json:"threadId"`
	AuthorKind  PrincipalKind `json:"authorKind"`
	AuthorID    string        `json:"-"`
	AuthorName  string        `json:"authorName"`
	Body        string        `json:"body"`
	Revision    int64         `json:"revision"`
	Mine        bool          `json:"mine"`
	EditedAt    *time.Time    `json:"editedAt,omitempty"`
	DeletedAt   *time.Time    `json:"deletedAt,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

type CommentThread struct {
	ID                uuid.UUID     `json:"id"`
	WorkspaceID       uuid.UUID     `json:"-"`
	SceneID           uuid.UUID     `json:"sceneId"`
	CreatorKind       PrincipalKind `json:"creatorKind"`
	CreatorID         string        `json:"-"`
	CreatorName       string        `json:"creatorName"`
	Anchor            CommentAnchor `json:"anchor"`
	Status            string        `json:"status"`
	Revision          int64         `json:"revision"`
	Subscribed        bool          `json:"subscribed"`
	Messages          []Comment     `json:"messages"`
	NextMessageCursor string        `json:"nextMessageCursor,omitempty"`
	CreatedAt         time.Time     `json:"createdAt"`
	UpdatedAt         time.Time     `json:"updatedAt"`
}

type CommentPage struct {
	Items      []CommentThread `json:"items"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

type CreateThreadInput struct {
	RequestID uuid.UUID
	Anchor    CommentAnchor
	Body      string
}

type ReplyInput struct {
	RequestID uuid.UUID
	Body      string
}

type CommentEditInput struct {
	RequestID        uuid.UUID
	ExpectedRevision int64
	Body             string
}

type RevisionInput struct {
	RequestID        uuid.UUID
	ExpectedRevision int64
}

type ThreadEditInput struct {
	RequestID        uuid.UUID
	ExpectedRevision int64
	Resolved         *bool
	Subscribed       *bool
}

type CommentRepository interface {
	CreateThread(Context, CommentThread, Comment, *uuid.UUID) error
	AddReply(Context, Comment, *uuid.UUID) error
	Thread(Context, SceneRef, uuid.UUID, Principal) (CommentThread, error)
	ThreadPage(Context, SceneRef, uuid.UUID, Principal, *uuid.UUID, int) (CommentThread, error)
	Message(Context, SceneRef, uuid.UUID, uuid.UUID) (Comment, error)
	UpdateMessage(Context, SceneRef, uuid.UUID, uuid.UUID, int64, string, bool) (Comment, error)
	UpdateThread(Context, SceneRef, uuid.UUID, int64, *bool) (CommentThread, error)
	SetSubscription(Context, SceneRef, uuid.UUID, uuid.UUID, bool) error
	ListThreads(Context, SceneRef, Principal, *uuid.UUID, int) ([]CommentThread, error)
	RecentWrites(Context, Principal, SceneRef, time.Time) (int, error)
	DisplayName(Context, Principal) (string, error)
}

type CommentUsecase interface {
	CreateThread(Context, Principal, SceneRef, CreateThreadInput) (CommentThread, error)
	Reply(Context, Principal, SceneRef, uuid.UUID, ReplyInput) (Comment, error)
	UpdateMessage(Context, Principal, SceneRef, uuid.UUID, uuid.UUID, CommentEditInput) (Comment, error)
	DeleteMessage(Context, Principal, SceneRef, uuid.UUID, uuid.UUID, RevisionInput) error
	UpdateThread(Context, Principal, SceneRef, uuid.UUID, ThreadEditInput) (CommentThread, error)
	List(Context, Principal, SceneRef, string, int) (CommentPage, error)
	GetThread(Context, Principal, SceneRef, uuid.UUID, string, int) (CommentThread, error)
}
