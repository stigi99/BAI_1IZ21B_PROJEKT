package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

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
		published    TINYINT(1)
	);`

	createUsers := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		username      VARCHAR(50)  UNIQUE,
		password_hash VARCHAR(255),
		email         VARCHAR(100) UNIQUE
	);`

	for _, stmt := range []string{createBlog, createUsers} {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("MigrateDB error: %v", err)
		}
	}
}

// SeedDB inserts dummy data into blog and users tables when they are empty.
func SeedDB(db *sql.DB) {
	// Seed blog posts
	var postCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM blog").Scan(&postCount); err != nil {
		log.Printf("SeedDB count blog error: %v", err)
		return
	}
	if postCount == 0 {
		posts := []struct {
			title, content string
			published      int
		}{
			{"Hello World", "This is the first blog post.", 1},
			{"Go is great", "Go makes it easy to build reliable software.", 1},
		}
		for _, p := range posts {
			if _, err := db.Exec(
				"INSERT INTO blog (title, post_content, published) VALUES (?, ?, ?)",
				p.title, p.content, p.published,
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
		users := []struct{ username, passwordHash, email string }{
			{"admin", "hashed_password_admin", "admin@example.com"},
			{"user1", "hashed_password_user1", "user1@example.com"},
		}
		for _, u := range users {
			if _, err := db.Exec(
				"INSERT INTO users (username, password_hash, email) VALUES (?, ?, ?)",
				u.username, u.passwordHash, u.email,
			); err != nil {
				log.Printf("SeedDB users error: %v", err)
			}
		}
	}
}
