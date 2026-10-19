package postgres

import (
	"github.com/google/uuid"
	"os"
	"testing"
)

func TestMigratePublicLinks(t *testing.T) {
	db := accessTestDB(t, 7)
	_, _, w := accessFixture(t, db)
	for i, active := range []bool{true, false} {
		scene, id := uuid.New(), uuid.New()
		mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,?)", scene, w, scene.String())
		mustExec(t, db, "INSERT INTO scene_permissions(id,workspace_id,scene_id,token,active,password_hash) VALUES (?,?,?,?,?,?)", id, w, scene, []string{"active-secret", "inactive-secret"}[i], active, "preserved-bcrypt-hash")
	}
	migration, e := os.ReadFile("../../../migrations/000008_share_links.up.sql")
	if os.IsNotExist(e) {
		t.Fatal("independent public links migration is missing")
	}
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, db, string(migration))
	var rows []struct {
		Access, PasswordHash, TokenHash string
		Active                          bool
	}
	if e = db.Table("share_links").Order("active DESC").Find(&rows).Error; e != nil {
		t.Fatal(e)
	}
	if len(rows) != 2 || rows[0].Access != "view" || !rows[0].Active || rows[1].Active || rows[0].PasswordHash != "preserved-bcrypt-hash" || rows[1].PasswordHash != "preserved-bcrypt-hash" {
		t.Fatalf("legacy grants changed: %+v", rows)
	}
	if rows[0].TokenHash != "ca45238f673736f4012b690150b84892b5bf69073f50aa124d4e2186e4cf50b4" {
		t.Fatalf("legacy token not hashed: %s", rows[0].TokenHash)
	}
	down, e := os.ReadFile("../../../migrations/000008_share_links.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, db, string(down))
	var active int64
	db.Table("scene_permissions").Where("active=true").Count(&active)
	if active != 1 {
		t.Fatal("legacy active state lost on safe rollback")
	}
}
