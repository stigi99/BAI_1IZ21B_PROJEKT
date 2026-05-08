package service

import (
	"database/sql"
	"html"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// loginRecord tracks failed login attempts for a username (used in secure mode
// to enforce a rate limit and slow down brute-force attacks).
type loginRecord struct {
	failures int
	resetAt  time.Time
}

// rateLimiter is a simple in-memory store; each service instance keeps its own
// so that tests remain isolated.
type rateLimiter struct {
	mu      sync.Mutex
	records map[string]*loginRecord
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{records: make(map[string]*loginRecord)}
}

type Service struct {
	db              *sql.DB
	securityEnabled bool
	rl              *rateLimiter
}

// Comment is a blog comment (may contain raw HTML in insecure mode for the
// Stored XSS demo).
type Comment struct {
	ID        int    `json:"id"`
	PostID    int    `json:"post_id"`
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// UserRecord exposes user rows for the Sensitive Data Exposure demo page.
type UserRecord struct {
	ID           int
	Username     string
	PasswordHash string
	Email        string
	Role         string
}

type Post struct {
	ID             int    `json:"id"`
	Title          string `json:"title"`
	PostContent    string `json:"post_content"`
	Published      int    `json:"published"`
	Author         string `json:"author_username,omitempty"`
	AttachmentPath string `json:"attachment_path,omitempty"`
	AttachmentName string `json:"attachment_name,omitempty"`
}

func New(db *sql.DB, securityEnabled bool) *Service {
	return &Service{db: db, securityEnabled: securityEnabled, rl: newRateLimiter()}
}

func (s *Service) GetPublishedPosts() ([]Post, error) {
	rows, err := s.db.Query(
		`SELECT id, title, post_content, published,
		        COALESCE(author_username, ''),
		        COALESCE(attachment_path, ''),
		        COALESCE(attachment_name, '')
		 FROM blog WHERE published = 1`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.PostContent, &p.Published, &p.Author, &p.AttachmentPath, &p.AttachmentName); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (s *Service) GetAllPosts() ([]Post, error) {
	rows, err := s.db.Query(
		`SELECT id, title, post_content, published,
		        COALESCE(author_username, ''),
		        COALESCE(attachment_path, ''),
		        COALESCE(attachment_name, '')
		 FROM blog ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.PostContent, &p.Published, &p.Author, &p.AttachmentPath, &p.AttachmentName); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (s *Service) GetPostByID(id int) (Post, error) {
	var p Post
	err := s.db.QueryRow(
		`SELECT id, title, post_content, published,
		        COALESCE(author_username, ''),
		        COALESCE(attachment_path, ''),
		        COALESCE(attachment_name, '')
		 FROM blog WHERE id = ?`,
		id,
	).Scan(&p.ID, &p.Title, &p.PostContent, &p.Published, &p.Author, &p.AttachmentPath, &p.AttachmentName)
	return p, err
}

// SearchPosts searches blog posts by query. Honors the SECURITY_ENABLED toggle:
// secure mode uses parameterized LIKE; insecure mode concatenates user input
// into the SQL string (classic SQL Injection).
func (s *Service) SearchPosts(query string) ([]Post, error) {
	if s.securityEnabled {
		return s.SearchPostsSecure(query)
	}
	return s.SearchPostsVulnerable(query)
}

// SearchPostsVulnerable concatenates the user-supplied query directly into the
// SQL string. This is the demo SQL Injection sink used regardless of the
// SECURITY_ENABLED toggle (so the lab can always show a "force vulnerable"
// endpoint side-by-side).
//
// Example payloads:
//   - `' OR 1=1 --`   → returns every row (drafts included)
//   - `' UNION SELECT id, username, password_hash, 1, '', '', '' FROM users --`
//     → leaks credentials through the title/content columns
func (s *Service) SearchPostsVulnerable(query string) ([]Post, error) {
	// VULNERABLE: direct string concatenation into the SQL statement.
	sqlQuery := "SELECT id, title, post_content, published, " +
		"COALESCE(author_username, ''), COALESCE(attachment_path, ''), COALESCE(attachment_name, '') " +
		"FROM blog WHERE title LIKE '%" + query + "%' OR post_content LIKE '%" + query + "%'"

	rows, err := s.db.Query(sqlQuery)
	if err != nil {
		return nil, err
	}
	return scanPostRows(rows)
}

// SearchPostsSecure runs the same query through a parameterized LIKE so user
// input cannot break out of the string literal — special characters become
// part of the search term, not SQL syntax.
func (s *Service) SearchPostsSecure(query string) ([]Post, error) {
	pattern := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT id, title, post_content, published,
		        COALESCE(author_username, ''),
		        COALESCE(attachment_path, ''),
		        COALESCE(attachment_name, '')
		 FROM blog
		 WHERE published = 1
		   AND (title LIKE ? OR post_content LIKE ?)`,
		pattern, pattern,
	)
	if err != nil {
		return nil, err
	}
	return scanPostRows(rows)
}

func scanPostRows(rows *sql.Rows) ([]Post, error) {
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.PostContent, &p.Published, &p.Author, &p.AttachmentPath, &p.AttachmentName); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

// CreatePost inserts a new post. Use empty strings for attachmentPath/Name when
// no file is attached.
func (s *Service) CreatePost(title, content string, published int, author, attachmentPath, attachmentName string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO blog (title, post_content, published, author_username, attachment_path, attachment_name)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		title, content, published, author, attachmentPath, attachmentName,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdatePost updates a post. Pass empty strings for attachment params to leave
// the existing attachment unchanged; pass non-empty values to replace it.
func (s *Service) UpdatePost(id int, title, content string, published int, attachmentPath, attachmentName string) error {
	if attachmentPath == "" && attachmentName == "" {
		_, err := s.db.Exec(
			`UPDATE blog SET title = ?, post_content = ?, published = ? WHERE id = ?`,
			title, content, published, id,
		)
		return err
	}

	_, err := s.db.Exec(
		`UPDATE blog SET title = ?, post_content = ?, published = ?,
		                  attachment_path = ?, attachment_name = ?
		 WHERE id = ?`,
		title, content, published, attachmentPath, attachmentName, id,
	)
	return err
}

func (s *Service) DeletePost(id int) error {
	_, err := s.db.Exec("DELETE FROM blog WHERE id = ?", id)
	return err
}

func (s *Service) GetPostAuthor(id int) (string, error) {
	var author sql.NullString
	err := s.db.QueryRow("SELECT author_username FROM blog WHERE id = ?", id).Scan(&author)
	if err != nil {
		return "", err
	}
	if !author.Valid {
		return "", nil
	}
	return author.String, nil
}

// CreateUser registers a new user.
// In secure mode the password is hashed with bcrypt before storage.
// In insecure mode the password is stored in plaintext (Sensitive Data Exposure).
func (s *Service) CreateUser(username, password, email string) error {
	stored, err := s.preparePassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		"INSERT INTO users (username, password_hash, email, role) VALUES (?, ?, ?, 'user')",
		username, stored, email,
	)
	return err
}

func (s *Service) preparePassword(plain string) (string, error) {
	if !s.securityEnabled {
		// VULNERABLE: plaintext storage (Sensitive Data Exposure)
		return plain, nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *Service) UserExists(username string) (bool, error) {
	var id int
	err := s.db.QueryRow(
		"SELECT id FROM users WHERE username = ?", username,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ValidateUserCredentials checks whether username + password authenticate.
// Secure mode: bcrypt comparison.
// Insecure mode: only verifies the user exists (Broken Authentication —
// password is ignored to keep the legacy lab behavior).
func (s *Service) ValidateUserCredentials(username, password string) (bool, error) {
	var stored string
	err := s.db.QueryRow(
		"SELECT password_hash FROM users WHERE username = ?",
		username,
	).Scan(&stored)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if !s.securityEnabled {
		// VULNERABLE: Broken Authentication — accept any password for an existing user
		return true, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)); err != nil {
		return false, nil
	}
	return true, nil
}

// GetDB returns the database connection for use in vulnerable endpoints (demo only)
func (s *Service) GetDB() *sql.DB {
	return s.db
}

func (s *Service) IsUserAdmin(username string) (bool, error) {
	var role string
	err := s.db.QueryRow(
		"SELECT role FROM users WHERE username = ?",
		username,
	).Scan(&role)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return role == "admin", nil
}

// ============================================================================
// Comments — Stored XSS demo
// ============================================================================

// CreateComment inserts a comment.
// In insecure mode the body is stored verbatim (Stored XSS risk).
// In secure mode the body is HTML-escaped before storage so script tags cannot
// execute when the comment is rendered.
func (s *Service) CreateComment(postID int, author, body string) (int64, error) {
	stored := body
	if s.securityEnabled {
		stored = html.EscapeString(body)
	}
	res, err := s.db.Exec(
		"INSERT INTO comments (post_id, author, body) VALUES (?, ?, ?)",
		postID, author, stored,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// CreateCommentVulnerable always stores the body verbatim, regardless of the
// SECURITY_ENABLED toggle. This keeps the force-vulnerable endpoint useful for
// side-by-side demo comparisons.
func (s *Service) CreateCommentVulnerable(postID int, author, body string) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO comments (post_id, author, body) VALUES (?, ?, ?)",
		postID, author, body,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetCommentsByPostID returns all comments for a given post, ordered oldest
// first.
func (s *Service) GetCommentsByPostID(postID int) ([]Comment, error) {
	rows, err := s.db.Query(
		`SELECT id, post_id, author, body, created_at FROM comments WHERE post_id = ? ORDER BY id ASC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.Author, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// ============================================================================
// Users — Sensitive Data Exposure demo
// ============================================================================

// GetAllUsers returns all rows from the users table.
// Used by the /ui/db-expose route to demonstrate Sensitive Data Exposure:
// in insecure mode password_hash contains the plaintext password.
func (s *Service) GetAllUsers() ([]UserRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, username, password_hash, email, role FROM users ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserRecord
	for rows.Next() {
		var u UserRecord
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateUserEmail changes the email for the given username.
// Used by the CSRF demo to perform a meaningful state change.
func (s *Service) UpdateUserEmail(username, newEmail string) error {
	_, err := s.db.Exec(
		"UPDATE users SET email = ? WHERE username = ?",
		newEmail, username,
	)
	return err
}

// GetUserEmail returns the current email for a username.
func (s *Service) GetUserEmail(username string) (string, error) {
	var email sql.NullString
	err := s.db.QueryRow("SELECT email FROM users WHERE username = ?", username).Scan(&email)
	if err != nil {
		return "", err
	}
	return email.String, nil
}

// ============================================================================
// Rate limiting — Broken Authentication / Brute Force demo
// ============================================================================

const (
	maxFailures    = 5
	lockoutSeconds = 60
)

// CheckRateLimit returns true (allowed) when the username has not exceeded the
// failure threshold within the lockout window. Always returns true in insecure
// mode so brute-force is trivially possible.
func (s *Service) CheckRateLimit(username string) bool {
	if !s.securityEnabled {
		return true
	}
	s.rl.mu.Lock()
	defer s.rl.mu.Unlock()

	rec, ok := s.rl.records[username]
	if !ok {
		return true
	}
	if time.Now().After(rec.resetAt) {
		delete(s.rl.records, username)
		return true
	}
	return rec.failures < maxFailures
}

// RecordLoginFailure increments the failure counter for the username.
// No-op in insecure mode.
func (s *Service) RecordLoginFailure(username string) {
	if !s.securityEnabled {
		return
	}
	s.rl.mu.Lock()
	defer s.rl.mu.Unlock()

	rec, ok := s.rl.records[username]
	if !ok || time.Now().After(rec.resetAt) {
		s.rl.records[username] = &loginRecord{
			failures: 1,
			resetAt:  time.Now().Add(lockoutSeconds * time.Second),
		}
		return
	}
	rec.failures++
}
