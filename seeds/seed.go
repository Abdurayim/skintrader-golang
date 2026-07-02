package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/pkg/hash"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://skintrader:skintrader@localhost:5432/skintrader?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Connected to database")

	if err := seedGames(ctx, pool); err != nil {
		log.Fatalf("Failed to seed games: %v", err)
	}

	if err := seedAdmin(ctx, pool); err != nil {
		log.Fatalf("Failed to seed admin: %v", err)
	}

	fmt.Println("\nSeeding completed successfully!")
}

func seedGames(ctx context.Context, pool *pgxpool.Pool) error {
	fmt.Println("\nSeeding games...")

	games := []struct {
		Name   string
		Icon   string
		Genres []string
	}{
		{"Counter-Strike 2", "cs2.png", []string{"FPS", "Action"}},
		{"Dota 2", "dota2.png", []string{"MOBA", "Strategy"}},
		{"PUBG: Battlegrounds", "pubg.png", []string{"Battle Royale", "FPS"}},
		{"Fortnite", "fortnite.png", []string{"Battle Royale", "Action"}},
		{"Valorant", "valorant.png", []string{"FPS", "Action"}},
		{"League of Legends", "lol.png", []string{"MOBA", "Strategy"}},
		{"Rust", "rust.png", []string{"Survival", "Action"}},
		{"Apex Legends", "apex.png", []string{"Battle Royale", "FPS"}},
		{"GTA V Online", "gtav.png", []string{"Action", "Adventure", "Driving"}},
		{"Roblox", "roblox.png", []string{"Sandbox", "Social"}},
		{"Minecraft", "minecraft.png", []string{"Sandbox", "Survival", "Adventure"}},
		{"FIFA / EA FC", "eafc.png", []string{"Sports", "Football"}},
		{"Mobile Legends", "mlbb.png", []string{"MOBA", "Mobile"}},
		{"Genshin Impact", "genshin.png", []string{"RPG", "Adventure", "Action"}},
		{"Call of Duty: Warzone", "warzone.png", []string{"Battle Royale", "FPS"}},
		{"World of Warcraft", "wow.png", []string{"MMO", "RPG"}},
		{"Diablo IV", "diablo4.png", []string{"RPG", "Action"}},
		{"Rocket League", "rocketleague.png", []string{"Sports", "Racing"}},
		{"Rainbow Six Siege", "r6.png", []string{"FPS", "Strategy"}},
		{"Overwatch 2", "ow2.png", []string{"FPS", "Action"}},
		{"Team Fortress 2", "tf2.png", []string{"FPS", "Action"}},
		{"Path of Exile", "poe.png", []string{"RPG", "Action"}},
		{"Brawl Stars", "brawlstars.png", []string{"Mobile", "Action"}},
		{"Clash Royale", "clashroyale.png", []string{"Mobile", "Strategy", "Card"}},
		{"Standoff 2", "standoff2.png", []string{"FPS", "Mobile"}},
	}

	for _, g := range games {
		id := uuid.New()
		slug := slugify(g.Name)

		var exists bool
		err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM games WHERE slug = $1)", slug).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check game %s: %w", g.Name, err)
		}
		if exists {
			fmt.Printf("  [skip] %s (already exists)\n", g.Name)
			continue
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO games (id, name, slug, icon, genres, is_active, posts_count, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, true, 0, NOW(), NOW())`,
			id, g.Name, slug, g.Icon, g.Genres,
		)
		if err != nil {
			return fmt.Errorf("insert game %s: %w", g.Name, err)
		}
		fmt.Printf("  [ok]   %s\n", g.Name)
	}

	var count int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM games").Scan(&count)
	fmt.Printf("Total games: %d\n", count)
	return nil
}

func seedAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	fmt.Println("\nSeeding superadmin...")

	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@skintrader.uz"
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "Admin123!"
	}
	name := os.Getenv("ADMIN_NAME")
	if name == "" {
		name = "Super Admin"
	}

	var exists bool
	err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM admins WHERE role = 'superadmin')").Scan(&exists)
	if err != nil {
		return fmt.Errorf("check admin: %w", err)
	}
	if exists {
		fmt.Println("  [skip] Superadmin already exists")
		return nil
	}

	passwordHash, err := hash.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	permissions := []string{
		"manage_users", "manage_posts", "manage_admins",
		"view_kyc", "approve_kyc", "view_logs", "view_stats",
		"manage_games", "manage_subscriptions", "manage_reports",
	}

	id := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO admins (id, email, password_hash, name, role, permissions, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'superadmin', $5, true, NOW(), NOW())`,
		id, email, passwordHash, name, permissions,
	)
	if err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}

	fmt.Printf("  [ok]   Superadmin created\n")
	fmt.Printf("         Email:    %s\n", email)
	fmt.Printf("         Password: %s\n", password)
	return nil
}

func slugify(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			result = append(result, c)
		case c >= 'A' && c <= 'Z':
			result = append(result, c+32)
		case c == ' ' || c == '-' || c == '_':
			if len(result) > 0 && result[len(result)-1] != '-' {
				result = append(result, '-')
			}
		}
	}
	if len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	return string(result)
}
