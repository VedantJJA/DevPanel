package main

import (
	"context"
	"fmt"
	"log"

	"github.com/VedantJJA/devpnl/internal/db"
)

func main() {
	database, err := db.Open("devpnl.db")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	tok, _ := database.GetSetting(ctx, "github_token")
	user, _ := database.GetSetting(ctx, "github_username")
	session, _ := database.GetSetting(ctx, "admin_session_token")

	fmt.Println("=== DEVPANEL DB SETTINGS ===")
	fmt.Println("GitHub Token Set:", tok != "")
	fmt.Println("GitHub Username:", user)
	fmt.Println("Session Token:", session)
}
