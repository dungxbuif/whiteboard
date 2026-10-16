package usecase

import (
	"context"
	"devtool/internal/domain"
	"encoding/base64"
	"github.com/google/uuid"
	"testing"
)

type exportAccess struct{ domain.AccessService }

func (exportAccess) WithWorkspaceLock(ctx domain.Context, _ uuid.UUID, _ uuid.UUID, fn func(domain.Context) error) error {
	return fn(ctx)
}
func (exportAccess) Resolve(domain.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error) {
	return domain.AccessDecision{Access: domain.AccessView}, nil
}

type exportKeys struct {
	domain.AgentRepository
	key     domain.AgentKey
	revoked bool
}

func (k *exportKeys) FindActive(domain.Context, uuid.UUID) (domain.AgentKey, error) {
	if k.revoked {
		return domain.AgentKey{}, domain.ErrAgentCredential
	}
	return k.key, nil
}

type exportRows struct {
	domain.SceneAssetRepository
	row domain.SceneAsset
}

func (r exportRows) GetByFileIDs(domain.Context, uuid.UUID, uuid.UUID, []string) ([]domain.SceneAsset, error) {
	return []domain.SceneAsset{r.row}, nil
}

type exportObjects struct {
	bytes []byte
	calls int
	after func()
}

func (o *exportObjects) ReadExportObject(domain.Context, string, string, int64) ([]byte, error) {
	o.calls++
	if o.after != nil {
		o.after()
	}
	return o.bytes, nil
}
func TestExportAssetsScopeAndRevocation(t *testing.T) {
	keyID, w, s, a := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	file := "file"
	p := domain.Principal{Kind: domain.PrincipalAgent, ID: "agent:" + keyID.String(), GrantID: keyID.String()}
	ref := domain.SceneRef{WorkspaceID: w, SceneID: s}
	keys := &exportKeys{key: domain.AgentKey{ID: keyID, WorkspaceID: w, OwnerID: uuid.New(), Scopes: []string{domain.ScopeSceneExport}}}
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	objects := &exportObjects{bytes: png}
	row := domain.SceneAsset{ID: a, WorkspaceID: w, SceneID: &s, FileID: &file, MimeType: "image/png", ByteSize: int64(len(png)), StorageBucket: "server", StorageKey: "owned"}
	service := NewExportAssets(NewAgentService(keys, exportAccess{}), exportRows{row: row}, objects)
	out, e := service.Get(context.Background(), p, ref, file, a.String())
	if e != nil || out.MimeType != "image/png" || len(out.Data) != len(png) {
		t.Fatalf("export-only download: %v", e)
	}
	keys.key.Scopes = []string{domain.ScopeSceneRead}
	before := objects.calls
	if _, e = service.Get(context.Background(), p, ref, file, a.String()); e == nil || objects.calls != before {
		t.Fatal("read scope must not fetch export bytes")
	}
	keys.key.Scopes = []string{domain.ScopeSceneExport}
	objects.after = func() { keys.revoked = true }
	if _, e = service.Get(context.Background(), p, ref, file, a.String()); e == nil {
		t.Fatal("revoke during storage read must deny result")
	}
}
