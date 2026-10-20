package handler

import (
	"context"
	"devtool/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type collectionUsecaseStub struct {
	workspaceID uuid.UUID
	created     domain.CollectionCreateInput
}

func (s *collectionUsecaseStub) Create(_ domain.Context, _ domain.Principal, workspaceID uuid.UUID, input domain.CollectionCreateInput) (domain.Collection, error) {
	s.workspaceID, s.created = workspaceID, input
	return domain.Collection{ID: uuid.New(), Name: input.Name, Visibility: input.Visibility, Revision: 1}, nil
}
func (*collectionUsecaseStub) Update(domain.Context, domain.Principal, uuid.UUID, uuid.UUID, domain.CollectionUpdateInput) (domain.Collection, error) {
	return domain.Collection{}, nil
}
func (*collectionUsecaseStub) Delete(domain.Context, domain.Principal, uuid.UUID, uuid.UUID, domain.CollectionRevisionInput) error {
	return nil
}
func (*collectionUsecaseStub) Get(domain.Context, domain.Principal, uuid.UUID, uuid.UUID) (domain.Collection, error) {
	return domain.Collection{}, nil
}
func (*collectionUsecaseStub) List(domain.Context, domain.Principal, uuid.UUID, string, int) (domain.CollectionPage, error) {
	return domain.CollectionPage{Items: []domain.Collection{}}, nil
}
func (*collectionUsecaseStub) AddScene(domain.Context, domain.Principal, uuid.UUID, uuid.UUID, domain.CollectionSceneInput) (domain.Collection, error) {
	return domain.Collection{}, nil
}
func (*collectionUsecaseStub) RemoveScene(domain.Context, domain.Principal, uuid.UUID, uuid.UUID, domain.CollectionSceneInput) (domain.Collection, error) {
	return domain.Collection{}, nil
}
func (*collectionUsecaseStub) Reorder(domain.Context, domain.Principal, uuid.UUID, uuid.UUID, domain.CollectionReorderInput) (domain.Collection, error) {
	return domain.Collection{}, nil
}

type collectionHandlerAccessStub struct {
	workspaceID uuid.UUID
	err         error
}

func (s collectionHandlerAccessStub) ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error) {
	return s.workspaceID, s.err
}

func TestCollectionHandlerPreservesInaccessibleWorkspace(t *testing.T) {
	userID := uuid.New()
	h := NewCollectionHandler(&collectionUsecaseStub{}, collectionHandlerAccessStub{err: domain.ErrInaccessible})
	router := chi.NewRouter()
	h.Register(router, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, userID)))
		})
	})
	request := httptest.NewRequest(http.MethodGet, "/workspaces/unknown/collections/not-a-uuid", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestCollectionHandlerCreatesInResolvedWorkspace(t *testing.T) {
	workspaceID, userID, requestID := uuid.New(), uuid.New(), uuid.New()
	uc := &collectionUsecaseStub{}
	h := NewCollectionHandler(uc, collectionHandlerAccessStub{workspaceID: workspaceID})
	router := chi.NewRouter()
	h.Register(router, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, userID)))
		})
	})
	request := httptest.NewRequest(http.MethodPost, "/workspaces/workspace-alias/collections", strings.NewReader(`{"requestId":"`+requestID.String()+`","name":"Launch","visibility":"private"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || uc.workspaceID != workspaceID || uc.created.RequestID != requestID || uc.created.Name != "Launch" || uc.created.Visibility != domain.CollectionPrivate {
		t.Fatalf("status=%d workspace=%s input=%+v body=%s", response.Code, uc.workspaceID, uc.created, response.Body.String())
	}
}

func TestCollectionHandlerRejectsUnknownFields(t *testing.T) {
	workspaceID, userID := uuid.New(), uuid.New()
	h := NewCollectionHandler(&collectionUsecaseStub{}, collectionHandlerAccessStub{workspaceID: workspaceID})
	router := chi.NewRouter()
	h.Register(router, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, userID)))
		})
	})
	request := httptest.NewRequest(http.MethodPost, "/workspaces/x/collections", strings.NewReader(`{"requestId":"`+uuid.NewString()+`","name":"Launch","visibility":"workspace","admin":true}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestCollectionHandlerRejectsMalformedLimit(t *testing.T) {
	workspaceID, userID := uuid.New(), uuid.New()
	h := NewCollectionHandler(&collectionUsecaseStub{}, collectionHandlerAccessStub{workspaceID: workspaceID})
	router := chi.NewRouter()
	h.Register(router, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, userID)))
		})
	})
	request := httptest.NewRequest(http.MethodGet, "/workspaces/x/collections?limit=abc", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_INPUT"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
