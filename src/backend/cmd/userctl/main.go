package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"devtool/internal/domain"
	postgresrepo "devtool/internal/repository/postgres"
	"devtool/internal/usecase"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	email := flag.String("email", "", "user email")
	password := flag.String("password", "", "user password")
	name := flag.String("name", "", "display name (defaults to email)")
	breakGlassWorkspace := flag.String("break-glass-workspace", "", "workspace UUID for operator recovery")
	breakGlassAction := flag.String("break-glass-action", "", "create, test, recover, or revoke an operator recovery credential")
	breakGlassCredentialFD := flag.Int("break-glass-credential-fd", 0, "file descriptor containing the recovery credential; terminal input is hidden")
	flag.Parse()
	if strings.TrimSpace(*email) == "" || (*breakGlassAction == "" && *password == "") {
		log.Fatal("--email and --password are required unless managing break-glass recovery")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	ctx := context.Background()
	emailValue := strings.ToLower(strings.TrimSpace(*email))
	var user domain.User
	err = db.WithContext(ctx).Where("email = ?", emailValue).First(&user).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		user = domain.User{ID: uuid.New(), Email: emailValue}
	case err != nil:
		log.Fatalf("find user: %v", err)
	}
	if *breakGlassAction != "" {
		workspaceID, parseErr := uuid.Parse(*breakGlassWorkspace)
		if parseErr != nil || workspaceID == uuid.Nil {
			log.Fatal("--break-glass-workspace must be a workspace UUID")
		}
		if user.ID == uuid.Nil {
			log.Fatal("break-glass user must already exist")
		}
		credential := ""
		if *breakGlassAction == "test" || *breakGlassAction == "recover" {
			credential, err = readRecoveryCredential(*breakGlassCredentialFD)
			if err != nil {
				log.Fatal(err)
			}
		}
		message, err := manageBreakGlass(ctx, db, user, workspaceID, *breakGlassAction, credential)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(message)
		return
	}
	if strings.TrimSpace(*name) != "" {
		user.Name = strings.TrimSpace(*name)
	} else if user.Name == "" {
		user.Name = emailValue
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}
	hashValue := string(hash)
	user.PasswordHash = &hashValue
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("save user: %w", err)
		}
		return usecase.EnsureDefaultWorkspace(ctx, postgresrepo.NewUserProvisioningRepository(tx), user.ID)
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("user upserted and default workspace ensured: %s (%s)\n", user.Email, user.ID)
}

func readRecoveryCredential(fd int) (string, error) {
	if fd < 0 {
		return "", errors.New("--break-glass-credential-fd must be non-negative")
	}
	if term.IsTerminal(fd) {
		_, _ = fmt.Fprint(os.Stderr, "Break-glass credential: ")
		value, err := term.ReadPassword(fd)
		_, _ = fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		credential := strings.TrimSpace(string(value))
		if credential == "" {
			return "", errors.New("break-glass credential is empty")
		}
		return credential, nil
	}
	file := os.NewFile(uintptr(fd), "break-glass-credential")
	if file == nil {
		return "", errors.New("invalid break-glass credential file descriptor")
	}
	value, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil {
		return "", err
	}
	if len(value) > 4096 {
		return "", errors.New("break-glass credential exceeds 4096 bytes")
	}
	credential := strings.TrimSpace(string(value))
	if credential == "" {
		return "", errors.New("break-glass credential is empty")
	}
	return credential, nil
}

func manageBreakGlass(ctx context.Context, db *gorm.DB, user domain.User, workspaceID uuid.UUID, action, credential string) (string, error) {
	var message string
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owners int64
		if err := tx.Table("workspaces").Where("id=? AND owner_id=? AND deleted_at IS NULL", workspaceID, user.ID).Count(&owners).Error; err != nil || owners != 1 {
			return errors.New("break-glass user must be the current workspace owner")
		}
		now := time.Now().UTC()
		switch action {
		case "create":
			var secret [32]byte
			if _, err := rand.Read(secret[:]); err != nil {
				return err
			}
			raw := base64.RawURLEncoding.EncodeToString(secret[:])
			digest := sha256.Sum256([]byte(raw))
			if err := tx.Exec(`INSERT INTO workspace_break_glass(id,workspace_id,user_id,credential_hash,created_at,tested_at,revoked_at) VALUES (?,?,?,?,?,NULL,NULL) ON CONFLICT (workspace_id,user_id) DO UPDATE SET credential_hash=EXCLUDED.credential_hash,created_at=EXCLUDED.created_at,tested_at=NULL,revoked_at=NULL`, uuid.New(), workspaceID, user.ID, digest[:], now).Error; err != nil {
				return err
			}
			if err := tx.Exec("INSERT INTO workspace_break_glass_audit(workspace_id,user_id,action) VALUES (?,?,'created')", workspaceID, user.ID).Error; err != nil {
				return err
			}
			message = "break-glass credential (shown once): " + raw
		case "test", "recover":
			if credential == "" {
				return errors.New("break-glass credential input is required for test or recovery")
			}
			var recovery struct {
				CredentialHash []byte
				TestedAt       *time.Time
			}
			if err := tx.Raw("SELECT credential_hash,tested_at FROM workspace_break_glass WHERE workspace_id=? AND user_id=? AND revoked_at IS NULL FOR UPDATE", workspaceID, user.ID).Scan(&recovery).Error; err != nil {
				return err
			}
			digest := sha256.Sum256([]byte(credential))
			if len(recovery.CredentialHash) != len(digest) || subtle.ConstantTimeCompare(recovery.CredentialHash, digest[:]) != 1 {
				return errors.New("invalid break-glass credential")
			}
			if action == "test" {
				if err := tx.Exec("UPDATE workspace_break_glass SET tested_at=? WHERE workspace_id=? AND user_id=?", now, workspaceID, user.ID).Error; err != nil {
					return err
				}
				if err := tx.Exec("INSERT INTO workspace_break_glass_audit(workspace_id,user_id,action) VALUES (?,?,'tested')", workspaceID, user.ID).Error; err != nil {
					return err
				}
				message = "break-glass recovery tested"
				return nil
			}
			if recovery.TestedAt == nil || recovery.TestedAt.Before(now.Add(-30*24*time.Hour)) {
				return domain.ErrBreakGlassRequired
			}
			jwtSecret := []byte(os.Getenv("JWT_SECRET"))
			sessions, err := usecase.NewSessionService(postgresrepo.NewAuthSessionRepository(tx), postgresrepo.NewSSORepository(tx), jwtSecret, func() time.Time { return now })
			if err != nil {
				return err
			}
			session, err := sessions.IssueBreakGlassSession(ctx, user.ID, workspaceID)
			if err != nil {
				return err
			}
			token, err := sessions.IssueLoginToken(&user, session)
			if err != nil {
				return err
			}
			if err := tx.Exec("INSERT INTO workspace_break_glass_audit(workspace_id,user_id,action) VALUES (?,?,'used')", workspaceID, user.ID).Error; err != nil {
				return err
			}
			message = "break-glass bearer token (shown once): " + token
		case "revoke":
			if err := tx.Exec("UPDATE workspace_break_glass SET revoked_at=? WHERE workspace_id=? AND user_id=? AND revoked_at IS NULL", now, workspaceID, user.ID).Error; err != nil {
				return err
			}
			if err := tx.Exec("INSERT INTO workspace_break_glass_audit(workspace_id,user_id,action) VALUES (?,?,'revoked')", workspaceID, user.ID).Error; err != nil {
				return err
			}
			message = "break-glass recovery revoked"
		default:
			return errors.New("--break-glass-action must be create, test, recover, or revoke")
		}
		return nil
	})
	return message, err
}
