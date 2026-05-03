package service

import (
	"database/sql"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db              *sql.DB
	securityEnabled bool
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
	return &Service{db: db, securityEnabled: securityEnabled}
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
