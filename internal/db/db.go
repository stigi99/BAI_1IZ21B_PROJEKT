package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

const (
	fallbackAdminUsername = "admin"
	fallbackAdminPassword = "admin"
	fallbackAdminEmail    = "admin@example.com"
)

func seededAdminCredentials() (username, password, email string) {
	username = os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = fallbackAdminUsername
	}

	password = os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = fallbackAdminPassword
	}

	email = os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = fallbackAdminEmail
	}

	return username, password, email
}

// InitDB opens (or creates) a SQLite database file at filepath and returns
// the *sql.DB handle.
func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	return db
}

// MigrateDB creates the blog and users tables if they do not already exist.
func MigrateDB(db *sql.DB) {
	createBlog := `
	CREATE TABLE IF NOT EXISTS blog (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		title        VARCHAR(255),
		post_content TEXT,
		published    TINYINT(1),
		author_username VARCHAR(50)
	);`

	createUsers := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		username      VARCHAR(50)  UNIQUE,
		password_hash VARCHAR(255),
		email         VARCHAR(100) UNIQUE,
		role          VARCHAR(20) NOT NULL DEFAULT 'user'
	);`

	for _, stmt := range []string{createBlog, createUsers} {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("MigrateDB error: %v", err)
		}
	}

	if _, err := db.Exec("ALTER TABLE blog ADD COLUMN author_username VARCHAR(50)"); err != nil {
		log.Printf("MigrateDB add blog.author_username note: %v", err)
	}

	if _, err := db.Exec("ALTER TABLE users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'user'"); err != nil {
		log.Printf("MigrateDB add users.role note: %v", err)
	}
}

// SeedDB inserts dummy data into blog and users tables when they are empty.
func SeedDB(db *sql.DB) {
	adminUsername, adminPassword, adminEmail := seededAdminCredentials()

	// Seed blog posts
	var postCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM blog").Scan(&postCount); err != nil {
		log.Printf("SeedDB count blog error: %v", err)
		return
	}
	if postCount == 0 {
		posts := []struct {
			title, content, author string
			published              int
		}{
			{"Hello World", "This is the first blog post.", adminUsername, 1},
			{"Go is great", "Go makes it easy to build reliable software.", adminUsername, 1},
		}
		for _, p := range posts {
			if _, err := db.Exec(
				"INSERT INTO blog (title, post_content, published, author_username) VALUES (?, ?, ?, ?)",
				p.title, p.content, p.published, p.author,
			); err != nil {
				log.Printf("SeedDB blog error: %v", err)
			}
		}
	}

	// Seed users
	var userCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		log.Printf("SeedDB count users error: %v", err)
		return
	}
	if userCount == 0 {
		users := []struct{ username, passwordHash, email, role string }{
			{adminUsername, adminPassword, adminEmail, "admin"},
			{"user1", "user1pass", "user1@example.com", "user"},
		}
		for _, u := range users {
			if _, err := db.Exec(
				"INSERT INTO users (username, password_hash, email, role) VALUES (?, ?, ?, ?)",
				u.username, u.passwordHash, u.email, u.role,
			); err != nil {
				log.Printf("SeedDB users error: %v", err)
			}
		}
	}

	if _, err := db.Exec(
		"INSERT OR IGNORE INTO users (username, password_hash, email, role) VALUES (?, ?, ?, 'admin')",
		adminUsername, adminPassword, adminEmail,
	); err != nil {
		log.Printf("SeedDB ensure admin error: %v", err)
	}

	if _, err := db.Exec(
		"UPDATE users SET role = 'admin' WHERE username = ?",
		adminUsername,
	); err != nil {
		log.Printf("SeedDB enforce admin role error: %v", err)
	}
}
