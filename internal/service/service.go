package service

import (
	"database/sql"
	"html"
	"regexp"
	"sync"
	"time"

	"BAI_1IZ21B_PROJEKT/internal/security"

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

// newRateLimiter creates an empty in-memory rate limiter for one service.
func newRateLimiter() *rateLimiter {
	return &rateLimiter{records: make(map[string]*loginRecord)}
}

// Service coordinates database access, security-mode checks, and in-memory
// throttling for authentication.
type Service struct {
	db   *sql.DB
	mode *security.ModeStore
	rl   *rateLimiter
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

// Post represents a blog post, including optional attachment metadata.
type Post struct {
	ID             int    `json:"id"`
	Title          string `json:"title"`
	PostContent    string `json:"post_content"`
	Published      int    `json:"published"`
	Author         string `json:"author_username,omitempty"`
	AttachmentPath string `json:"attachment_path,omitempty"`
	AttachmentName string `json:"attachment_name,omitempty"`
}

var (
	scriptTagRE        = regexp.MustCompile(`(?i)</?\s*script[^>]*>`)
	eventAttributeRE   = regexp.MustCompile(`(?i)\s+on[a-z0-9_-]+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	javascriptSchemeRE = regexp.MustCompile(`(?i)javascript\s*:`)
)

// New constructs a Service using a fresh mode store seeded with the provided
// security state.
//
// Inputs:
//   - db: database handle used by all service methods.
//   - securityEnabled: initial lab mode.
//
// Output:
//   - *Service: ready-to-use service instance.
func New(db *sql.DB, securityEnabled bool) *Service {
	return NewWithMode(db, security.NewModeStore(securityEnabled))
}

// NewWithMode constructs a Service that shares an existing security-mode store.
//
// Inputs:
//   - db: database handle used by all service methods.
//   - mode: shared mode store used for runtime secure/vulnerable toggling.
//
// Output:
//   - *Service: ready-to-use service instance bound to the supplied mode store.
func NewWithMode(db *sql.DB, mode *security.ModeStore) *Service {
	return &Service{db: db, mode: mode, rl: newRateLimiter()}
}

// SecurityEnabled reports whether the service should execute secure code paths.
//
// Output:
//   - bool: current security state.
func (s *Service) SecurityEnabled() bool {
	return s.mode.Enabled()
}

// SetSecurityEnabled updates the active security mode used by the service.
//
// Input:
//   - enabled: new security state.
func (s *Service) SetSecurityEnabled(enabled bool) {
	s.mode.Set(enabled)
}

// GetPublishedPosts returns all posts whose published flag is enabled.
//
// Output:
//   - []Post: published posts ordered as returned by SQLite.
//   - error: query or row scanning failure.
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

// GetAllPosts returns every post regardless of publication state.
//
// Output:
//   - []Post: all posts ordered from newest to oldest.
//   - error: query or row scanning failure.
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

// GetPostByID loads a single post by database identifier.
//
// Input:
//   - id: post identifier to look up.
//
// Output:
//   - Post: matching row when found.
//   - error: lookup or scan failure.
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
	if s.SecurityEnabled() {
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
//   - `zz' ORDER BY 8 --`
//     -> leaks the SELECT column count through a database error in vulnerable mode
//   - `zz' OR EXISTS(SELECT 1 FROM users WHERE username='admin' AND substr(password_hash,1,1)='a') --`
//     -> proves boolean inference against the users table without a visible login form error
//   - `zz' UNION SELECT 1, name, sql, 1, char(32), char(32), char(32) FROM sqlite_master WHERE type='table' --`
//     -> enumerates SQLite table names and CREATE statements
//   - `zz' UNION SELECT id, '[user] ' || username, 'role=' || role || ' email=' || email || ' secret=' || password_hash, 1, username, char(32), char(32) FROM users --`
//     -> leaks user records through the title/content columns
//   - `zz' UNION SELECT 9000, '[intel] database map', 'users=' || (SELECT COUNT(*) FROM users) || ' tables=' || (SELECT group_concat(name, ', ') FROM sqlite_master WHERE type='table'), 1, 'sqli-bot', char(32), char(32) --`
//     -> creates an attacker-controlled intelligence summary row
//   - `zz' UNION SELECT id, '[draft] ' || title, post_content, published, author_username, char(32), char(32) FROM blog WHERE published=0 --`
//     -> leaks hidden drafts that secure search intentionally filters out
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

// scanPostRows converts a raw SQL rows iterator into a slice of Post values.
//
// Input:
//   - rows: open result set with the blog schema columns in order.
//
// Output:
//   - []Post: posts scanned from the result set.
//   - error: scan or iteration failure.
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

// DeletePost removes a post by identifier.
//
// Input:
//   - id: post identifier to delete.
//
// Output:
//   - error: SQL execution failure, if any.
func (s *Service) DeletePost(id int) error {
	_, err := s.db.Exec("DELETE FROM blog WHERE id = ?", id)
	return err
}

// GetPostAuthor returns the author username stored for a post.
//
// Input:
//   - id: post identifier to inspect.
//
// Outputs:
//   - string: author username when present.
//   - error: lookup or scan failure.
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

// preparePassword hashes a password in secure mode and returns it unchanged in
// insecure mode.
//
// Input:
//   - plain: plaintext password supplied during registration or seeding.
//
// Outputs:
//   - string: value to persist in the database.
//   - error: bcrypt failure in secure mode.
func (s *Service) preparePassword(plain string) (string, error) {
	if !s.SecurityEnabled() {
		// VULNERABLE: plaintext storage (Sensitive Data Exposure)
		return plain, nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// UserExists reports whether a username is already stored in the users table.
//
// Input:
//   - username: account name to look up.
//
// Outputs:
//   - bool: true when the user exists.
//   - error: lookup or scan failure.
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

	if !s.SecurityEnabled() {
		// VULNERABLE: Broken Authentication — accept any password for an existing user
		return true, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)); err != nil {
		return false, nil
	}
	return true, nil
}

// GetDB returns the database connection for use in vulnerable endpoints (demo only)
// GetDB exposes the underlying database handle for the demo-only vulnerable
// endpoints that intentionally bypass the service abstraction.
//
// Output:
//   - *sql.DB: shared database connection.
func (s *Service) GetDB() *sql.DB {
	return s.db
}

// IsUserAdmin reports whether the given username belongs to an admin account.
//
// Input:
//   - username: account name to inspect.
//
// Outputs:
//   - bool: true when the stored role is admin.
//   - error: lookup or scan failure.
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
	if s.SecurityEnabled() {
		stored = html.EscapeString(stripUnsafeHTML(body))
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

// stripUnsafeHTML removes the most common script-bearing HTML patterns before
// the secure comment path escapes the remaining text.
//
// Input:
//   - body: raw HTML payload supplied by the user.
//
// Output:
//   - string: text with inline script constructs removed.
func stripUnsafeHTML(body string) string {
	cleaned := scriptTagRE.ReplaceAllString(body, "")
	cleaned = eventAttributeRE.ReplaceAllString(cleaned, "")
	cleaned = javascriptSchemeRE.ReplaceAllString(cleaned, "")
	return cleaned
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
	if !s.SecurityEnabled() {
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
	if !s.SecurityEnabled() {
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
