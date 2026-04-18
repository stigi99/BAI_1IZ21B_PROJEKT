//go:build integration

package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dbpkg "BAI_1IZ21B_PROJEKT/internal/db"

	"github.com/gin-gonic/gin"
)

type postDTO struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	PostContent string `json:"post_content"`
	Published   int    `json:"published"`
}

func setupIntegrationTestApp(t *testing.T) (*gin.Engine, *sql.DB, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration.db")
	db := dbpkg.InitDB(dbPath)

	dbpkg.MigrateDB(db)
	dbpkg.SeedDB(db)

	router := buildRouter(db)

	return router, db, dbPath
}

func doRequest(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func doFormRequest(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestIntegration_Ping(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	resp := doRequest(t, router, http.MethodGet, "/ping", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed reading response body: %v", err)
	}

	if string(body) != "{\"message\":\"pong\"}" {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestIntegration_Posts(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	resp := doRequest(t, router, http.MethodGet, "/posts", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var posts []postDTO
	if err := json.Unmarshal(resp.Body.Bytes(), &posts); err != nil {
		t.Fatalf("failed to decode posts response: %v", err)
	}

	if len(posts) < 2 {
		t.Fatalf("expected at least 2 seeded posts, got %d", len(posts))
	}

	for _, p := range posts {
		if p.Published != 1 {
			t.Fatalf("expected only published posts, found post id=%d with published=%d", p.ID, p.Published)
		}
	}
}

func TestIntegration_Login(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	t.Run("existing user", func(t *testing.T) {
		resp := doRequest(t, router, http.MethodPost, "/login", `{"username":"admin","password":"whatever"}`)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		resp := doRequest(t, router, http.MethodPost, "/login", `{"username":"ghost","password":"x"}`)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		resp := doRequest(t, router, http.MethodPost, "/login", `{bad json}`)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
		}
	})
}

func TestIntegration_DatabaseFileIsCreated(t *testing.T) {
	_, db, dbPath := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected DB file to exist at %s: %v", dbPath, err)
	}
}

func TestIntegration_UI_PostsPage(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	resp := doRequest(t, router, http.MethodGet, "/ui/posts", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	if !strings.Contains(resp.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected text/html content-type, got %s", resp.Header().Get("Content-Type"))
	}

	body := resp.Body.String()
	if !strings.Contains(body, "Published Posts") {
		t.Fatalf("expected posts page marker in body, got: %s", body)
	}
}

func TestIntegration_UI_LoginPage(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	resp := doRequest(t, router, http.MethodGet, "/ui/login", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	if !strings.Contains(resp.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected text/html content-type, got %s", resp.Header().Get("Content-Type"))
	}

	body := resp.Body.String()
	if !strings.Contains(body, "<form") {
		t.Fatalf("expected login form in body, got: %s", body)
	}
}

func TestIntegration_UI_LoginSubmit(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	t.Run("existing user", func(t *testing.T) {
		resp := doFormRequest(t, router, http.MethodPost, "/ui/login", "username=admin&password=anything")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
		}
		if !strings.Contains(resp.Body.String(), "Login successful") {
			t.Fatalf("expected success message in body, got: %s", resp.Body.String())
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		resp := doFormRequest(t, router, http.MethodPost, "/ui/login", "username=ghost&password=x")
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
		}
		if !strings.Contains(resp.Body.String(), "User not found") {
			t.Fatalf("expected not found message in body, got: %s", resp.Body.String())
		}
	})
}

func TestIntegration_UI_PostsPartial(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	resp := doRequest(t, router, http.MethodGet, "/ui/partials/posts", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	if !strings.Contains(resp.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected text/html content-type, got %s", resp.Header().Get("Content-Type"))
	}

	body := resp.Body.String()
	if !strings.Contains(body, "id=\"posts-list\"") {
		t.Fatalf("expected posts-list fragment in body, got: %s", body)
	}
}

func TestIntegration_UI_LoginPartial(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	t.Run("existing user", func(t *testing.T) {
		resp := doFormRequest(t, router, http.MethodPost, "/ui/partials/login", "username=admin&password=anything")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
		}
		if !strings.Contains(resp.Body.String(), "Login successful") {
			t.Fatalf("expected success message in body, got: %s", resp.Body.String())
		}
	})

	t.Run("missing credentials", func(t *testing.T) {
		resp := doFormRequest(t, router, http.MethodPost, "/ui/partials/login", "username=&password=")
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
		}
		if !strings.Contains(resp.Body.String(), "required") {
			t.Fatalf("expected validation message in body, got: %s", resp.Body.String())
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		resp := doFormRequest(t, router, http.MethodPost, "/ui/partials/login", "username=ghost&password=x")
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
		}
		if !strings.Contains(resp.Body.String(), "User not found") {
			t.Fatalf("expected not found message in body, got: %s", resp.Body.String())
		}
	})
}
