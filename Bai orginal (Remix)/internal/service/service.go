package service

import "database/sql"

type Service struct {
	db *sql.DB
}

type Post struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	PostContent string `json:"post_content"`
	Published   int    `json:"published"`
	Author      string `json:"author_username,omitempty"`
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetPublishedPosts() ([]Post, error) {
	rows, err := s.db.Query(
		"SELECT id, title, post_content, published, COALESCE(author_username, '') FROM blog WHERE published = 1",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.PostContent, &p.Published, &p.Author); err != nil {
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
	rows, err := s.db.Query("SELECT id, title, post_content, published, COALESCE(author_username, '') FROM blog ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.PostContent, &p.Published, &p.Author); err != nil {
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
		"SELECT id, title, post_content, published, COALESCE(author_username, '') FROM blog WHERE id = ?",
		id,
	).Scan(&p.ID, &p.Title, &p.PostContent, &p.Published, &p.Author)
	return p, err
}

func (s *Service) CreatePost(title, content string, published int, author string) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO blog (title, post_content, published, author_username) VALUES (?, ?, ?, ?)",
		title, content, published, author,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Service) UpdatePost(id int, title, content string, published int) error {
	_, err := s.db.Exec(
		"UPDATE blog SET title = ?, post_content = ?, published = ? WHERE id = ?",
		title, content, published, id,
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

func (s *Service) CreateUser(username, passwordHash, email string) error {
	_, err := s.db.Exec(
		"INSERT INTO users (username, password_hash, email, role) VALUES (?, ?, ?, 'user')",
		username, passwordHash, email,
	)
	return err
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

func (s *Service) ValidateUserCredentials(username, password string) (bool, error) {
	var passwordHash string
	err := s.db.QueryRow(
		"SELECT password_hash FROM users WHERE username = ?",
		username,
	).Scan(&passwordHash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return passwordHash == password, nil

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
