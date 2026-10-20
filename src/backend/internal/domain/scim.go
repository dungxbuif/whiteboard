package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	SCIMUserSchema              = "urn:ietf:params:scim:schemas:core:2.0:User"
	SCIMGroupSchema             = "urn:ietf:params:scim:schemas:core:2.0:Group"
	SCIMListSchema              = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	SCIMErrorSchema             = "urn:ietf:params:scim:api:messages:2.0:Error"
	SCIMPatchSchema             = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	SCIMServiceProviderSchema   = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
	SCIMMaxPageSize             = 100
	SCIMTokenBytes              = 32
	SCIMMembershipSourceDirect  = "direct"
	SCIMMembershipSourceGroup   = "group"
	SCIMMembershipSourceAdopted = "adopted"
)

var (
	ErrSCIMInvalid      = errors.New("invalid scim request")
	ErrSCIMUnauthorized = errors.New("invalid scim credential")
	ErrSCIMNotFound     = errors.New("scim resource not found")
	ErrSCIMConflict     = errors.New("scim resource conflict")
	ErrSCIMPrecondition = errors.New("scim precondition failed")
	ErrSCIMUnsupported  = errors.New("scim operation unsupported")
)

type SCIMMeta struct {
	ResourceType string    `json:"resourceType"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Version      string    `json:"version"`
}

type SCIMName struct {
	Formatted  string `json:"formatted,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
}

type SCIMEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type SCIMUser struct {
	Schemas              []string    `json:"schemas"`
	ID                   string      `json:"id"`
	ExternalID           string      `json:"externalId,omitempty"`
	UserName             string      `json:"userName"`
	DisplayName          string      `json:"displayName,omitempty"`
	Name                 SCIMName    `json:"name,omitempty"`
	Emails               []SCIMEmail `json:"emails,omitempty"`
	Active               bool        `json:"active"`
	Role                 string      `json:"role,omitempty"`
	RetainedManualAccess bool        `json:"retainedManualAccess,omitempty"`
	Meta                 SCIMMeta    `json:"meta"`
}

type SCIMGroupMember struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
}

type SCIMGroup struct {
	Schemas     []string          `json:"schemas"`
	ID          string            `json:"id"`
	ExternalID  string            `json:"externalId,omitempty"`
	DisplayName string            `json:"displayName"`
	Role        string            `json:"role"`
	Members     []SCIMGroupMember `json:"members,omitempty"`
	Meta        SCIMMeta          `json:"meta"`
}

type SCIMPage[T any] struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    []T      `json:"Resources"`
}

type SCIMError struct {
	Schemas  []string `json:"schemas"`
	Status   string   `json:"status"`
	SCIMType string   `json:"scimType,omitempty"`
	Detail   string   `json:"detail"`
}

type SCIMPatchOperation struct {
	Operation string `json:"op"`
	Path      string `json:"path,omitempty"`
	Value     any    `json:"value,omitempty"`
}
type SCIMPatchRequest struct {
	Schemas    []string             `json:"schemas"`
	Operations []SCIMPatchOperation `json:"Operations"`
}

type SCIMCredential struct {
	WorkspaceID uuid.UUID
	TokenHash   []byte
	Prefix      string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	RevokedAt   *time.Time
}
type SCIMTokenResult struct {
	Token     string    `json:"token"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"createdAt"`
}
type SCIMTokenStatus struct {
	Configured bool       `json:"configured"`
	Prefix     string     `json:"prefix,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
}

type SCIMFilter struct{ Attribute, Value string }

type SCIMRepository interface {
	ResolveCredential(Context, []byte) (uuid.UUID, error)
	CredentialStatus(Context, uuid.UUID) (SCIMTokenStatus, error)
	RotateCredential(Context, SCIMCredential) error
	RevokeCredential(Context, uuid.UUID, time.Time) error
	CreateUser(Context, uuid.UUID, SCIMUser, time.Time) (SCIMUser, error)
	GetUser(Context, uuid.UUID, uuid.UUID) (SCIMUser, error)
	ListUsers(Context, uuid.UUID, SCIMFilter, int, int) (SCIMPage[SCIMUser], error)
	ReplaceUser(Context, uuid.UUID, uuid.UUID, string, SCIMUser, time.Time) (SCIMUser, error)
	DeleteUser(Context, uuid.UUID, uuid.UUID, string, time.Time) (SCIMUser, error)
	AdoptUser(Context, uuid.UUID, uuid.UUID, time.Time) (SCIMUser, error)
	CreateGroup(Context, uuid.UUID, SCIMGroup, time.Time) (SCIMGroup, error)
	GetGroup(Context, uuid.UUID, uuid.UUID) (SCIMGroup, error)
	ListGroups(Context, uuid.UUID, SCIMFilter, int, int) (SCIMPage[SCIMGroup], error)
	ReplaceGroup(Context, uuid.UUID, uuid.UUID, string, SCIMGroup, time.Time) (SCIMGroup, error)
	DeleteGroup(Context, uuid.UUID, uuid.UUID, string, time.Time) error
}

type SCIMUsecase interface {
	RotateToken(Context, Principal, uuid.UUID) (SCIMTokenResult, error)
	TokenStatus(Context, Principal, uuid.UUID) (SCIMTokenStatus, error)
	RevokeToken(Context, Principal, uuid.UUID) error
	CreateUser(Context, uuid.UUID, SCIMUser) (SCIMUser, error)
	GetUser(Context, uuid.UUID, uuid.UUID) (SCIMUser, error)
	ListUsers(Context, uuid.UUID, string, int, int) (SCIMPage[SCIMUser], error)
	ReplaceUser(Context, uuid.UUID, uuid.UUID, string, SCIMUser) (SCIMUser, error)
	PatchUser(Context, uuid.UUID, uuid.UUID, string, SCIMPatchRequest) (SCIMUser, error)
	DeleteUser(Context, uuid.UUID, uuid.UUID, string) (SCIMUser, error)
	AdoptUser(Context, Principal, uuid.UUID, uuid.UUID) (SCIMUser, error)
	CreateGroup(Context, uuid.UUID, SCIMGroup) (SCIMGroup, error)
	GetGroup(Context, uuid.UUID, uuid.UUID) (SCIMGroup, error)
	ListGroups(Context, uuid.UUID, string, int, int) (SCIMPage[SCIMGroup], error)
	ReplaceGroup(Context, uuid.UUID, uuid.UUID, string, SCIMGroup) (SCIMGroup, error)
	PatchGroup(Context, uuid.UUID, uuid.UUID, string, SCIMPatchRequest) (SCIMGroup, error)
	DeleteGroup(Context, uuid.UUID, uuid.UUID, string) error
}
