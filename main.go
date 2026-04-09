package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// SecurityEnabled acts as a global security toggle.
// When false the application runs without security controls (insecure mode).
// When true proper authentication and input validation are enforced.
var SecurityEnabled = false

// ---- Database helpers -------------------------------------------------------

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
		// NOTE: password_hash values below are placeholder strings used only for
		// seeding. Real hashing (e.g. bcrypt) will be applied when SecurityEnabled = true.
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

// ---- Handlers ---------------------------------------------------------------

// getPosts returns all published blog posts as JSON.
func getPosts(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Future security branch: if SecurityEnabled { /* auth check */ }

		rows, err := db.Query(
			"SELECT id, title, post_content, published FROM blog WHERE published = 1",
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
			return
		}
		defer rows.Close()

		type Post struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			PostContent string `json:"post_content"`
			Published   int    `json:"published"`
		}

		var posts []Post
		for rows.Next() {
			var p Post
			if err := rows.Scan(&p.ID, &p.Title, &p.PostContent, &p.Published); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read post"})
				return
			}
			posts = append(posts, p)
		}

		c.JSON(http.StatusOK, posts)
	}
}

// postLogin accepts JSON with username and password.
// In the current (insecure) implementation it returns success if the username
// exists in the database, without verifying the password.
// When SecurityEnabled is true, proper password verification will be added.
func postLogin(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Future security branch: if SecurityEnabled { /* verify password hash */ }

		var id int
		err := db.QueryRow(
			"SELECT id FROM users WHERE username = ?", req.Username,
		).Scan(&id)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		// NOTE: password is not verified in this phase (SecurityEnabled = false).
		// Password verification will be enforced when SecurityEnabled is set to true.
		c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
	}
}

// ---- Entry point ------------------------------------------------------------

func main() {
	db := InitDB("app.db")
	defer db.Close()

	MigrateDB(db)
	SeedDB(db)

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	router.GET("/posts", getPosts(db))
	router.POST("/login", postLogin(db))

	log.Println("Starting server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
