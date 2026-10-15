package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"devtool/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	email := flag.String("email", "", "user email")
	password := flag.String("password", "", "user password")
	name := flag.String("name", "", "display name (defaults to email)")
	flag.Parse()
	if strings.TrimSpace(*email) == "" || *password == "" {
		log.Fatal("--email and --password are required")
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
		var workspace domain.Workspace
		workspaceErr := tx.Where("owner_id = ? AND deleted_at IS NULL", user.ID).First(&workspace).Error
		if errors.Is(workspaceErr, gorm.ErrRecordNotFound) {
			if err := tx.Create(&domain.Workspace{ID: uuid.New(), Name: "My Workspace", OwnerID: user.ID}).Error; err != nil {
				return fmt.Errorf("create default workspace: %w", err)
			}
			return nil
		}
		if workspaceErr != nil {
			return fmt.Errorf("find default workspace: %w", workspaceErr)
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("user upserted and default workspace ensured: %s (%s)\n", user.Email, user.ID)
}
