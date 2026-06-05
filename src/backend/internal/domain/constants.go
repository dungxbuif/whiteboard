package domain

import (
	"time"
)

const (
	// Authentication
	HeaderAuthorization = "Authorization"
	HeaderWorkspaceID   = "X-Workspace-ID"
	BearerPrefix       = "Bearer "
	AuthCookieName     = "oauth_state"
	
	// Timeouts & Durations
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultSessionDuration   = 7 * 24 * time.Hour
	OAuthStateMaxAge        = 300 // 5 minutes in seconds
	
	// Context Keys
	ContextKeyUserID      = "user_id"
	ContextKeyWorkspaceID = "workspace_id"
	
	// External APIs
	GoogleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

	// Folder Constants
	MaxFolderDepth      = 30
	FolderPathSeparator = "/"
)
