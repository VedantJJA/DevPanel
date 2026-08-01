package main

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
		log.Fatal(err)
	}
	defer database.Close()

	hash, _ := database.GetSetting(context.Background(), "admin_password_hash")
	fmt.Println("Stored Hash:", hash)

	testPasswords := []string{"admin", "admin123", "password", "password123", "VedantJJA", "devpanel", "devpanel123", "12345678"}
	for _, p := range testPasswords {
		if hashPassword(p) == hash {
			fmt.Println("MATCH FOUND! Password is:", p)
			return
		}
	}
	fmt.Println("No common match found in list.")
}
