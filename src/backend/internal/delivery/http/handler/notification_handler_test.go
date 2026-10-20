package handler

import (
	"context"
	"devtool/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type notificationServiceStub struct {
	userID    uuid.UUID
	cursor    string
	limit     int
	markedID  uuid.UUID
	requestID uuid.UUID
}

func (*notificationServiceStub) DeliverEvent(domain.Context, int64) error { return nil }
func (s *notificationServiceStub) List(_ domain.Context, userID uuid.UUID, cursor string, limit int) (domain.NotificationPage, error) {
	s.userID, s.cursor, s.limit = userID, cursor, limit
	return domain.NotificationPage{Items: []domain.Notification{{ID: uuid.New(), DeepLink: "/w/workspace/scenes/scene?thread=thread"}}}, nil
}
func (s *notificationServiceStub) MarkRead(_ domain.Context, userID, id, requestID uuid.UUID) error {
	s.userID, s.markedID, s.requestID = userID, id, requestID
	return nil
}

func TestNotificationHandlerListAndMarkRead(t *testing.T) {
	userID, notificationID, requestID := uuid.New(), uuid.New(), uuid.New()
	service := &notificationServiceStub{}
	handler := NewNotificationHandler(service).Routes(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, userID)))
		})
	})

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/notifications?cursor=cursor&limit=17", nil))
	if list.Code != http.StatusOK || service.userID != userID || service.cursor != "cursor" || service.limit != 17 || !strings.Contains(list.Body.String(), "deepLink") {
		t.Fatalf("status=%d service=%+v body=%s", list.Code, service, list.Body.String())
	}
	read := httptest.NewRecorder()
	handler.ServeHTTP(read, httptest.NewRequest(http.MethodPost, "/notifications/"+notificationID.String()+"/read", strings.NewReader(`{"requestId":"`+requestID.String()+`"}`)))
	if read.Code != http.StatusNoContent || service.markedID != notificationID || service.requestID != requestID {
		t.Fatalf("status=%d service=%+v body=%s", read.Code, service, read.Body.String())
	}
}
