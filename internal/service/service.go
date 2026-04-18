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
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetPublishedPosts() ([]Post, error) {
	rows, err := s.db.Query(
		"SELECT id, title, post_content, published FROM blog WHERE published = 1",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.PostContent, &p.Published); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
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
