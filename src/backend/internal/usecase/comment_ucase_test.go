package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCommentUnicodeLimit(t *testing.T) {
	service := NewCommentService(nil, nil, nil, nil)
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	principal := domain.UserPrincipal(uuid.New())

	for name, body := range map[string]string{
		"empty":        "",
		"whitespace":   " \n\t",
		"4001 unicode": strings.Repeat("界", 4001),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.CreateThread(context.Background(), principal, ref, domain.CreateThreadInput{
				RequestID: uuid.New(), Anchor: domain.CommentAnchor{X: 1, Y: 2}, Body: body,
			})
			if !errors.Is(err, domain.ErrInvalidCommentInput) {
				t.Fatalf("body accepted: %v", err)
			}
		})
	}
}
