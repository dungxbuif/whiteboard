package usecase

import (
	"devtool/internal/domain"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxCommentCodePoints = 4000
	maxCommentPageSize   = 50
	maxCommentWrites     = 30
)

type commentEventService interface {
	AppendEvent(domain.Context, domain.EventInput) (domain.WorkspaceEvent, error)
}

type CommentService struct {
	repo     domain.CommentRepository
	features domain.FeatureService
	access   domain.AccessService
	events   commentEventService
}

func NewCommentService(repo domain.CommentRepository, features domain.FeatureService, access domain.AccessService, events commentEventService) *CommentService {
	return &CommentService{repo: repo, features: features, access: access, events: events}
}

func validCommentBody(body string) bool {
	count := utf8.RuneCountInString(body)
	return utf8.ValidString(body) && count >= 1 && count <= maxCommentCodePoints && strings.TrimSpace(body) != ""
}

func validCommentAnchor(anchor domain.CommentAnchor) bool {
	if math.IsNaN(anchor.X) || math.IsNaN(anchor.Y) || math.IsInf(anchor.X, 0) || math.IsInf(anchor.Y, 0) {
		return false
	}
	return anchor.ElementID == nil || len(*anchor.ElementID) >= 1 && len(*anchor.ElementID) <= 255 && strings.TrimSpace(*anchor.ElementID) == *anchor.ElementID
}

func validCommentPrincipal(principal domain.Principal) bool {
	return principal.ID != "" && (principal.Kind == domain.PrincipalUser && principal.UserID != nil || principal.Kind == domain.PrincipalPublic && principal.UserID == nil)
}

func commentSubscriber(principal domain.Principal) *uuid.UUID {
	if principal.Kind == domain.PrincipalUser && principal.UserID != nil {
		id := *principal.UserID
		return &id
	}
	return nil
}

func commentActor(principal domain.Principal) domain.EventActorKind {
	if principal.Kind == domain.PrincipalPublic {
		return domain.EventActorPublic
	}
	return domain.EventActorUser
}

func mutationPayload(value any) ([]byte, error) { return json.Marshal(value) }

func decodeStored[T any](stored domain.StoredFeatureResponse) (T, error) {
	var value T
	if err := json.Unmarshal(stored.Body, &value); err != nil {
		return value, err
	}
	return value, nil
}

func (s *CommentService) writeAllowed(ctx domain.Context, principal domain.Principal, ref domain.SceneRef) error {
	count, err := s.repo.RecentWrites(ctx, principal, ref, time.Now().Add(-time.Minute))
	if err != nil {
		return err
	}
	if count >= maxCommentWrites {
		return domain.ErrCommentRateLimited
	}
	return nil
}

func (s *CommentService) appendEvent(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, kind, subject, sourceKey string, metadata map[string]any) error {
	sceneID := ref.SceneID
	_, err := s.events.AppendEvent(ctx, domain.EventInput{WorkspaceID: ref.WorkspaceID, SceneID: &sceneID, ActorKind: commentActor(principal), ActorID: principal.ID, Kind: kind, SubjectID: subject, SafeMetadata: metadata, Source: "comment", SourceKey: sourceKey})
	return err
}

func (s *CommentService) CreateThread(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, input domain.CreateThreadInput) (domain.CommentThread, error) {
	if input.RequestID == uuid.Nil || !validCommentPrincipal(principal) || ref.WorkspaceID == uuid.Nil || ref.SceneID == uuid.Nil || !validCommentBody(input.Body) || !validCommentAnchor(input.Anchor) || s.repo == nil || s.features == nil || s.events == nil {
		return domain.CommentThread{}, domain.ErrInvalidCommentInput
	}
	payload, err := mutationPayload(input)
	if err != nil {
		return domain.CommentThread{}, err
	}
	stored, err := s.features.Mutate(ctx, principal, ref, domain.FeatureCommentWrite, input.RequestID, payload, func(txCtx domain.Context) (domain.StoredFeatureResponse, error) {
		if err := s.writeAllowed(txCtx, principal, ref); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		name, err := s.repo.DisplayName(txCtx, principal)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		threadID, messageID := uuid.New(), uuid.New()
		thread := domain.CommentThread{ID: threadID, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, CreatorKind: principal.Kind, CreatorID: principal.ID, CreatorName: name, Anchor: input.Anchor, Status: domain.CommentStatusOpen, Revision: 1, Subscribed: principal.Kind == domain.PrincipalUser, Messages: []domain.Comment{}}
		message := domain.Comment{ID: messageID, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, ThreadID: threadID, AuthorKind: principal.Kind, AuthorID: principal.ID, AuthorName: name, Body: input.Body, Revision: 1, Mine: true}
		if err = s.repo.CreateThread(txCtx, thread, message, commentSubscriber(principal)); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		if err = s.appendEvent(txCtx, principal, ref, "comment.thread_created", threadID.String(), "thread:"+threadID.String()+":created", map[string]any{"threadId": threadID.String(), "commentId": messageID.String(), "revision": 1, "requestId": input.RequestID.String()}); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		created, err := s.repo.Thread(txCtx, ref, threadID, principal)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		body, err := json.Marshal(created)
		return domain.StoredFeatureResponse{Status: 201, Body: body}, err
	})
	if err != nil {
		return domain.CommentThread{}, err
	}
	return decodeStored[domain.CommentThread](stored)
}

func (s *CommentService) Reply(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, threadID uuid.UUID, input domain.ReplyInput) (domain.Comment, error) {
	if input.RequestID == uuid.Nil || threadID == uuid.Nil || !validCommentPrincipal(principal) || !validCommentBody(input.Body) {
		return domain.Comment{}, domain.ErrInvalidCommentInput
	}
	payload, err := mutationPayload(struct {
		ThreadID uuid.UUID
		Input    domain.ReplyInput
	}{threadID, input})
	if err != nil {
		return domain.Comment{}, err
	}
	stored, err := s.features.Mutate(ctx, principal, ref, domain.FeatureCommentWrite, input.RequestID, payload, func(txCtx domain.Context) (domain.StoredFeatureResponse, error) {
		if err := s.writeAllowed(txCtx, principal, ref); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		if _, err := s.repo.Thread(txCtx, ref, threadID, principal); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		name, err := s.repo.DisplayName(txCtx, principal)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		message := domain.Comment{ID: uuid.New(), WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, ThreadID: threadID, AuthorKind: principal.Kind, AuthorID: principal.ID, AuthorName: name, Body: input.Body, Revision: 1, Mine: true}
		if err = s.repo.AddReply(txCtx, message, commentSubscriber(principal)); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		if err = s.appendEvent(txCtx, principal, ref, "comment.replied", message.ID.String(), "message:"+message.ID.String()+":created", map[string]any{"threadId": threadID.String(), "commentId": message.ID.String(), "revision": 1, "requestId": input.RequestID.String()}); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		created, err := s.repo.Message(txCtx, ref, threadID, message.ID)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		created.Mine = true
		body, err := json.Marshal(created)
		return domain.StoredFeatureResponse{Status: 201, Body: body}, err
	})
	if err != nil {
		return domain.Comment{}, err
	}
	return decodeStored[domain.Comment](stored)
}

func (s *CommentService) canChangeMessage(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, message domain.Comment, deleting bool) error {
	if message.AuthorKind == principal.Kind && message.AuthorID == principal.ID {
		return nil
	}
	if !deleting {
		return domain.ErrAccessDenied
	}
	return s.features.Require(ctx, principal, ref, domain.FeatureCommentModerate)
}

func (s *CommentService) UpdateMessage(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, threadID, messageID uuid.UUID, input domain.CommentEditInput) (domain.Comment, error) {
	if input.RequestID == uuid.Nil || threadID == uuid.Nil || messageID == uuid.Nil || input.ExpectedRevision < 1 || !validCommentBody(input.Body) {
		return domain.Comment{}, domain.ErrInvalidCommentInput
	}
	payload, err := mutationPayload(struct {
		ThreadID, MessageID uuid.UUID
		Input               domain.CommentEditInput
	}{threadID, messageID, input})
	if err != nil {
		return domain.Comment{}, err
	}
	stored, err := s.features.Mutate(ctx, principal, ref, domain.FeatureCommentWrite, input.RequestID, payload, func(txCtx domain.Context) (domain.StoredFeatureResponse, error) {
		if err := s.writeAllowed(txCtx, principal, ref); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		message, err := s.repo.Message(txCtx, ref, threadID, messageID)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		if err = s.canChangeMessage(txCtx, principal, ref, message, false); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		updated, err := s.repo.UpdateMessage(txCtx, ref, threadID, messageID, input.ExpectedRevision, input.Body, false)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		updated.Mine = true
		if err = s.appendEvent(txCtx, principal, ref, "comment.edited", messageID.String(), "message:"+messageID.String()+":revision:"+strconv.FormatInt(updated.Revision, 10), map[string]any{"threadId": threadID.String(), "commentId": messageID.String(), "revision": updated.Revision, "requestId": input.RequestID.String()}); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		body, err := json.Marshal(updated)
		return domain.StoredFeatureResponse{Status: 200, Body: body}, err
	})
	if err != nil {
		return domain.Comment{}, err
	}
	return decodeStored[domain.Comment](stored)
}

func (s *CommentService) DeleteMessage(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, threadID, messageID uuid.UUID, input domain.RevisionInput) error {
	if input.RequestID == uuid.Nil || threadID == uuid.Nil || messageID == uuid.Nil || input.ExpectedRevision < 1 {
		return domain.ErrInvalidCommentInput
	}
	payload, err := mutationPayload(struct {
		ThreadID, MessageID uuid.UUID
		Input               domain.RevisionInput
	}{threadID, messageID, input})
	if err != nil {
		return err
	}
	_, err = s.features.Mutate(ctx, principal, ref, domain.FeatureCommentWrite, input.RequestID, payload, func(txCtx domain.Context) (domain.StoredFeatureResponse, error) {
		if err := s.writeAllowed(txCtx, principal, ref); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		message, err := s.repo.Message(txCtx, ref, threadID, messageID)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		if err = s.canChangeMessage(txCtx, principal, ref, message, true); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		updated, err := s.repo.UpdateMessage(txCtx, ref, threadID, messageID, input.ExpectedRevision, "", true)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		if err = s.appendEvent(txCtx, principal, ref, "comment.deleted", messageID.String(), "message:"+messageID.String()+":revision:"+strconv.FormatInt(updated.Revision, 10), map[string]any{"threadId": threadID.String(), "commentId": messageID.String(), "revision": updated.Revision, "requestId": input.RequestID.String()}); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		return domain.StoredFeatureResponse{Status: 204, Body: []byte(`{"deleted":true}`)}, nil
	})
	return err
}

func (s *CommentService) UpdateThread(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, threadID uuid.UUID, input domain.ThreadEditInput) (domain.CommentThread, error) {
	if input.RequestID == uuid.Nil || threadID == uuid.Nil || input.ExpectedRevision < 1 || input.Resolved == nil && input.Subscribed == nil || input.Subscribed != nil && (principal.Kind != domain.PrincipalUser || principal.UserID == nil) {
		return domain.CommentThread{}, domain.ErrInvalidCommentInput
	}
	action := domain.FeatureCommentRead
	if input.Resolved != nil {
		action = domain.FeatureCommentWrite
	}
	payload, err := mutationPayload(struct {
		ThreadID uuid.UUID
		Input    domain.ThreadEditInput
	}{threadID, input})
	if err != nil {
		return domain.CommentThread{}, err
	}
	stored, err := s.features.Mutate(ctx, principal, ref, action, input.RequestID, payload, func(txCtx domain.Context) (domain.StoredFeatureResponse, error) {
		if err := s.writeAllowed(txCtx, principal, ref); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		thread, err := s.repo.Thread(txCtx, ref, threadID, principal)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		if thread.Revision != input.ExpectedRevision {
			return domain.StoredFeatureResponse{}, domain.ErrRevisionConflict
		}
		if input.Resolved != nil {
			decision, err := s.access.Resolve(txCtx, principal, ref)
			if err != nil {
				return domain.StoredFeatureResponse{}, err
			}
			if principal.Kind != domain.PrincipalUser || decision.Access != domain.AccessEdit {
				return domain.StoredFeatureResponse{}, domain.ErrAccessDenied
			}
			thread, err = s.repo.UpdateThread(txCtx, ref, threadID, input.ExpectedRevision, input.Resolved)
			if err != nil {
				return domain.StoredFeatureResponse{}, err
			}
			kind := "comment.reopened"
			if *input.Resolved {
				kind = "comment.resolved"
			}
			if err = s.appendEvent(txCtx, principal, ref, kind, threadID.String(), "thread:"+threadID.String()+":revision:"+strconv.FormatInt(thread.Revision, 10), map[string]any{"threadId": threadID.String(), "revision": thread.Revision, "requestId": input.RequestID.String(), "status": thread.Status}); err != nil {
				return domain.StoredFeatureResponse{}, err
			}
		}
		if input.Subscribed != nil {
			if err = s.repo.SetSubscription(txCtx, ref, threadID, *principal.UserID, *input.Subscribed); err != nil {
				return domain.StoredFeatureResponse{}, err
			}
		}
		thread, err = s.repo.Thread(txCtx, ref, threadID, principal)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		body, err := json.Marshal(thread)
		return domain.StoredFeatureResponse{Status: 200, Body: body}, err
	})
	if err != nil {
		return domain.CommentThread{}, err
	}
	return decodeStored[domain.CommentThread](stored)
}

func (s *CommentService) List(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, cursor string, limit int) (domain.CommentPage, error) {
	if limit < 1 || limit > maxCommentPageSize {
		return domain.CommentPage{}, domain.ErrInvalidCommentInput
	}
	var cursorID *uuid.UUID
	if cursor != "" {
		id, err := uuid.Parse(cursor)
		if err != nil {
			return domain.CommentPage{}, domain.ErrInvalidCommentInput
		}
		cursorID = &id
	}
	if err := s.features.Require(ctx, principal, ref, domain.FeatureCommentRead); err != nil {
		return domain.CommentPage{}, err
	}
	items, err := s.repo.ListThreads(ctx, ref, principal, cursorID, limit+1)
	if err != nil {
		return domain.CommentPage{}, err
	}
	page := domain.CommentPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		page.NextCursor = page.Items[len(page.Items)-1].ID.String()
	}
	return page, nil
}

func (s *CommentService) GetThread(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, threadID uuid.UUID, cursor string, limit int) (domain.CommentThread, error) {
	if threadID == uuid.Nil || limit < 1 || limit > maxCommentPageSize {
		return domain.CommentThread{}, domain.ErrInvalidCommentInput
	}
	var cursorID *uuid.UUID
	if cursor != "" {
		id, err := uuid.Parse(cursor)
		if err != nil {
			return domain.CommentThread{}, domain.ErrInvalidCommentInput
		}
		cursorID = &id
	}
	if err := s.features.Require(ctx, principal, ref, domain.FeatureCommentRead); err != nil {
		return domain.CommentThread{}, err
	}
	return s.repo.ThreadPage(ctx, ref, threadID, principal, cursorID, limit)
}

var _ domain.CommentUsecase = (*CommentService)(nil)
