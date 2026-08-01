package resetpass

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/VedantJJA/devpnl/internal/db"
)

const fixedSalt = "devpanel_salt_v1_8237"

func hashPassword(password string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password + fixedSalt))
	return hex.EncodeToString(hasher.Sum(nil))
}

func main() {
	database, err := db.Open("devpnl.db")
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer database.Close()

	password := "admin"
	hash := hashPassword(password)

	err = database.SetSetting(context.Background(), "admin_password_hash", hash)
	if err != nil {
		log.Fatalf("Failed to set password: %v", err)
	}

	_ = database.SetSetting(context.Background(), "admin_session_token", "")

	fmt.Printf("Successfully set admin password to: %s\n", password)
}
