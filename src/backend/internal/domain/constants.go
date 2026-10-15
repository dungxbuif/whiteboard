package domain

import (
	"time"
)

const (
	// Authentication
	HeaderAuthorization = "Authorization"
	HeaderWorkspaceID   = "X-Workspace-ID"
	BearerPrefix        = "Bearer "
	OAuthStateCookie    = "devtool_oauth_state"

	// Timeouts & Durations
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultSessionDuration   = 7 * 24 * time.Hour

	// Context Keys
	ContextKeyUserID      = "user_id"
	ContextKeyWorkspaceID = "workspace_id"

	// Folder Constants
	MaxFolderDepth      = 30
	FolderPathSeparator = "/"
)
