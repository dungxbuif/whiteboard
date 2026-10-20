package usecase

import (
	"errors"
	"time"

	"devtool/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type SessionService struct {
	repo        domain.SessionRepository
	connections domain.SSORepository
	jwtSecret   []byte
	now         func() time.Time
}

func NewSessionService(repo domain.SessionRepository, connections domain.SSORepository, jwtSecret []byte, clocks ...func() time.Time) (*SessionService, error) {
	if repo == nil || connections == nil || len(jwtSecret) < 32 {
		return nil, errors.New("session authority requires repositories and JWT_SECRET of at least 32 bytes")
	}
	now := time.Now
	if len(clocks) > 0 && clocks[0] != nil {
		now = clocks[0]
	}
	return &SessionService{repo: repo, connections: connections, jwtSecret: append([]byte(nil), jwtSecret...), now: now}, nil
}

func (s *SessionService) create(ctx domain.Context, user uuid.UUID, method string, connection, workspace *uuid.UUID, lifetime time.Duration) (domain.AuthSession, error) {
	now := s.now().UTC()
	session := domain.AuthSession{ID: uuid.New(), UserID: user, Method: method, ConnectionID: connection, WorkspaceID: workspace, AuthenticatedAt: now, ExpiresAt: now.Add(lifetime)}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return domain.AuthSession{}, err
	}
	return session, nil
}
func (s *SessionService) IssuePasswordSession(ctx domain.Context, user uuid.UUID) (uuid.UUID, error) {
	session, e := s.create(ctx, user, domain.AuthMethodPassword, nil, nil, domain.DefaultSessionDuration)
	return session.ID, e
}
func (s *SessionService) IssueOIDCSession(ctx domain.Context, user, connection uuid.UUID) (domain.AuthSession, error) {
	configured, err := s.connections.GetConnectionByID(ctx, connection)
	if err != nil {
		return domain.AuthSession{}, err
	}
	return s.create(ctx, user, domain.AuthMethodOIDC, &connection, &configured.WorkspaceID, domain.DefaultSessionDuration)
}
func (s *SessionService) IssueBreakGlassSession(ctx domain.Context, user, workspace uuid.UUID) (domain.AuthSession, error) {
	ok, e := s.repo.HasTestedBreakGlass(ctx, workspace, user, s.now().Add(-30*24*time.Hour))
	if e != nil {
		return domain.AuthSession{}, e
	}
	if !ok {
		return domain.AuthSession{}, domain.ErrBreakGlassRequired
	}
	return s.create(ctx, user, domain.AuthMethodBreakGlass, nil, &workspace, domain.BreakGlassSessionLifetime)
}

func (s *SessionService) IssueLoginToken(user *domain.User, session domain.AuthSession) (string, error) {
	claims := jwt.MapClaims{"sub": user.ID.String(), "email": user.Email, "sid": session.ID.String(), "auth_method": session.Method, "auth_time": session.AuthenticatedAt.Unix(), "exp": session.ExpiresAt.Unix()}
	if session.Method == domain.AuthMethodBreakGlass {
		if session.WorkspaceID == nil || *session.WorkspaceID == uuid.Nil {
			return "", domain.ErrWorkspaceAssurance
		}
		claims["recovery_workspace_id"] = session.WorkspaceID.String()
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *SessionService) validateSession(ctx domain.Context, p domain.Principal) (domain.AuthSession, error) {
	if p.SessionID == uuid.Nil || p.UserID == nil {
		return domain.AuthSession{}, domain.ErrSessionRevoked
	}
	session, err := s.repo.GetSession(ctx, p.SessionID)
	if err != nil || session.UserID != *p.UserID || session.RevokedAt != nil || !s.now().Before(session.ExpiresAt) || (p.AuthMethod != "" && session.Method != p.AuthMethod) {
		return domain.AuthSession{}, domain.ErrSessionRevoked
	}
	if session.Method == domain.AuthMethodBreakGlass {
		if session.WorkspaceID == nil || p.RecoveryWorkspaceID == uuid.Nil || *session.WorkspaceID != p.RecoveryWorkspaceID {
			return domain.AuthSession{}, domain.ErrWorkspaceAssurance
		}
	} else if p.RecoveryWorkspaceID != uuid.Nil {
		return domain.AuthSession{}, domain.ErrWorkspaceAssurance
	}
	return session, nil
}

func (s *SessionService) RequireWorkspaceAssurance(ctx domain.Context, p domain.Principal, workspace uuid.UUID) error {
	if p.Kind != domain.PrincipalUser || p.UserID == nil || *p.UserID == uuid.Nil {
		return domain.ErrAccessDenied
	}
	connection, e := s.connections.GetConnection(ctx, workspace)
	if errors.Is(e, domain.ErrSSONotConfigured) {
		if p.SessionID == uuid.Nil {
			return nil
		}
		session, se := s.validateSession(ctx, p)
		if se != nil {
			return se
		}
		if session.Method == domain.AuthMethodBreakGlass {
			return domain.ErrWorkspaceAssurance
		}
		return nil
	}
	if e != nil {
		return e
	}
	if p.SessionID == uuid.Nil {
		if connection.Mode == domain.SSOModeEnforced {
			return domain.ErrWorkspaceAssurance
		}
		return nil
	}
	session, e := s.validateSession(ctx, p)
	if e != nil {
		return e
	}
	if session.Method == domain.AuthMethodBreakGlass && connection.Mode != domain.SSOModeEnforced {
		return domain.ErrWorkspaceAssurance
	}
	if connection.Mode == domain.SSOModeEnforced {
		if session.Method == domain.AuthMethodBreakGlass {
			if session.WorkspaceID == nil || *session.WorkspaceID != workspace {
				return domain.ErrWorkspaceAssurance
			}
			ok, e := s.repo.HasTestedBreakGlass(ctx, workspace, *p.UserID, s.now().Add(-30*24*time.Hour))
			if e != nil {
				return e
			}
			if !ok {
				return domain.ErrWorkspaceAssurance
			}
			return nil
		}
		if session.Method != domain.AuthMethodOIDC || session.ConnectionID == nil || *session.ConnectionID != connection.ID || session.WorkspaceID == nil || *session.WorkspaceID != workspace {
			return domain.ErrWorkspaceAssurance
		}
	}
	return nil
}
func (s *SessionService) RequirePublicLinkAccess(ctx domain.Context, workspace uuid.UUID) error {
	connection, err := s.connections.GetConnection(ctx, workspace)
	if errors.Is(err, domain.ErrSSONotConfigured) {
		return nil
	}
	if err != nil {
		return err
	}
	if !connection.PublicLinksEnabled {
		return domain.ErrAccessDenied
	}
	return nil
}
func (s *SessionService) RevokeWorkspaceSessions(ctx domain.Context, workspace, user uuid.UUID, connection *uuid.UUID) error {
	return s.repo.RevokeWorkspaceSessions(ctx, workspace, user, connection, s.now().UTC())
}
