package db

import (
	"database/sql"
	"html"
	"log"
	"os"
	"regexp"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

const (
	fallbackAdminUsername = "admin"
	fallbackAdminPassword = "admin"
	fallbackAdminEmail    = "admin@example.com"
)

var (
	seedScriptTagRE        = regexp.MustCompile(`(?i)</?\s*script[^>]*>`)
	seedEventAttributeRE   = regexp.MustCompile(`(?i)\s+on[a-z0-9_-]+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	seedJavascriptSchemeRE = regexp.MustCompile(`(?i)javascript\s*:`)
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

// MigrateDB creates the blog, users, and comments tables if they do not already exist.
func MigrateDB(db *sql.DB) {
	createBlog := `
	CREATE TABLE IF NOT EXISTS blog (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		title           VARCHAR(255),
		post_content    TEXT,
		published       TINYINT(1),
		author_username VARCHAR(50),
		attachment_path VARCHAR(255),
		attachment_name VARCHAR(255)
	);`

	createUsers := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		username      VARCHAR(50)  UNIQUE,
		password_hash VARCHAR(255),
		email         VARCHAR(100) UNIQUE,
		role          VARCHAR(20) NOT NULL DEFAULT 'user'
	);`

	createComments := `
	CREATE TABLE IF NOT EXISTS comments (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id    INTEGER NOT NULL,
		author     VARCHAR(50) NOT NULL DEFAULT 'anonymous',
		body       TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	for _, stmt := range []string{createBlog, createUsers, createComments} {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("MigrateDB error: %v", err)
		}
	}

	// Idempotent column additions for existing databases.
	addColumns := []string{
		"ALTER TABLE blog ADD COLUMN author_username VARCHAR(50)",
		"ALTER TABLE blog ADD COLUMN attachment_path VARCHAR(255)",
		"ALTER TABLE blog ADD COLUMN attachment_name VARCHAR(255)",
		"ALTER TABLE users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'user'",
	}
	for _, stmt := range addColumns {
		if _, err := db.Exec(stmt); err != nil {
			log.Printf("MigrateDB note (column may already exist): %v", err)
		}
	}
}

// SeedDB inserts dummy data into blog and users tables when they are empty.
// In secure mode user passwords are stored as bcrypt hashes; in insecure mode
// they are stored in plaintext (so the Sensitive Data Exposure scenario can be
// demonstrated by inspecting the database directly).
func SeedDB(db *sql.DB, securityEnabled bool) {
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
			// Post by user1 — used for the IDOR / Broken Access Control demo.
			{"User1's Post", "This post belongs to user1. In insecure mode any authenticated user can delete it.", "user1", 1},
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

	// Seed a demo XSS comment on post 1 when the comments table is empty.
	var commentCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM comments").Scan(&commentCount); err != nil {
		log.Printf("SeedDB count comments error: %v", err)
	} else if commentCount == 0 {
		// This stored XSS payload is intentionally inserted in insecure mode so
		// the demo works out of the box. In secure mode the body is HTML-escaped
		// before storage, so the script tag is inert.
		xssBody := `<img src=x onerror="alert('Stored XSS! cookie='+document.cookie)"> — demo payload`
		xssBody = encodeCommentForSeed(xssBody, securityEnabled)
		if _, err := db.Exec(
			"INSERT INTO comments (post_id, author, body) VALUES (1, 'attacker', ?)",
			xssBody,
		); err != nil {
			log.Printf("SeedDB comments error: %v", err)
		}
	}

	// Seed users
	var userCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		log.Printf("SeedDB count users error: %v", err)
		return
	}
	if userCount == 0 {
		users := []struct{ username, password, email, role string }{
			{adminUsername, adminPassword, adminEmail, "admin"},
			{"user1", "user1pass", "user1@example.com", "user"},
		}
		for _, u := range users {
			stored := encodePasswordForSeed(u.password, securityEnabled)
			if _, err := db.Exec(
				"INSERT INTO users (username, password_hash, email, role) VALUES (?, ?, ?, ?)",
				u.username, stored, u.email, u.role,
			); err != nil {
				log.Printf("SeedDB users error: %v", err)
			}
		}
	}

	// Always ensure an admin row exists with the configured password format
	// matching the current security mode.
	stored := encodePasswordForSeed(adminPassword, securityEnabled)
	if _, err := db.Exec(
		"INSERT OR IGNORE INTO users (username, password_hash, email, role) VALUES (?, ?, ?, 'admin')",
		adminUsername, stored, adminEmail,
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

// encodePasswordForSeed returns the plaintext password in insecure mode and
// a bcrypt hash in secure mode.
func encodePasswordForSeed(password string, securityEnabled bool) string {
	if !securityEnabled {
		return password
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("SeedDB bcrypt error (falling back to plaintext): %v", err)
		return password
	}
	return string(hash)
}

func encodeCommentForSeed(body string, securityEnabled bool) string {
	if !securityEnabled {
		return body
	}
	cleaned := seedScriptTagRE.ReplaceAllString(body, "")
	cleaned = seedEventAttributeRE.ReplaceAllString(cleaned, "")
	cleaned = seedJavascriptSchemeRE.ReplaceAllString(cleaned, "")
	return html.EscapeString(cleaned)
}
