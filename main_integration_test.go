//go:build integration

package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
	return setupIntegrationTestAppWithSecurity(t, false)
}

func setupIntegrationTestAppWithSecurity(t *testing.T, securityEnabled bool) (*gin.Engine, *sql.DB, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	previousSecurity := SecurityEnabled
	SecurityEnabled = securityEnabled
	t.Cleanup(func() { SecurityEnabled = previousSecurity })

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration.db")
	db := dbpkg.InitDB(dbPath)

	dbpkg.MigrateDB(db)
	dbpkg.SeedDB(db, securityEnabled)

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

func doRequestWithCookie(t *testing.T, router *gin.Engine, method, path, body, cookie string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func extractCreatedID(t *testing.T, body []byte) int {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	idValue, ok := payload["id"]
	if !ok {
		t.Fatalf("missing id in response: %s", string(body))
	}

	switch v := idValue.(type) {
	case float64:
		return int(v)
	case string:
		id, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("id is not numeric: %v", err)
		}
		return id
	default:
		t.Fatalf("unexpected id type: %T", idValue)
	}

	return 0
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
	if !strings.Contains(body, "Create Post") {
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
		// On success the partial responds with an HX-Redirect header (empty body) so
		// htmx triggers a full-page navigation that refreshes the auth-aware navbar.
		if redirect := resp.Header().Get("HX-Redirect"); redirect == "" {
			t.Fatalf("expected HX-Redirect header on successful login, got body: %s", resp.Body.String())
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

func TestIntegration_Register(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	t.Run("api register user", func(t *testing.T) {
		resp := doRequest(t, router, http.MethodPost, "/register", `{"username":"newuser","password":"newpass123","email":"newuser@example.com"}`)
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.Code)
		}

		loginResp := doRequest(t, router, http.MethodPost, "/login", `{"username":"newuser","password":"any"}`)
		if loginResp.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, loginResp.Code)
		}
	})

	t.Run("ui register user", func(t *testing.T) {
		resp := doFormRequest(t, router, http.MethodPost, "/ui/register", "username=uiuser&password=uipass123&email=uiuser%40example.com")
		if resp.Code != http.StatusSeeOther {
			t.Fatalf("expected status %d, got %d", http.StatusSeeOther, resp.Code)
		}
		location := resp.Header().Get("Location")
		if !strings.Contains(location, "msg=User+registered") {
			t.Fatalf("expected success redirect, got %s", location)
		}
	})
}

func TestIntegration_DeleteAuthorization_SecurityEnabled(t *testing.T) {
	router, db, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = db.Close() })

	loginUserResp := doRequest(t, router, http.MethodPost, "/login", `{"username":"user1","password":"user1pass"}`)
	if loginUserResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, loginUserResp.Code)
	}
	userCookie := loginUserResp.Header().Get("Set-Cookie")
	if userCookie == "" {
		t.Fatalf("expected auth cookie for user")
	}

	loginAdminResp := doRequest(t, router, http.MethodPost, "/login", `{"username":"admin","password":"admin"}`)
	if loginAdminResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, loginAdminResp.Code)
	}
	adminCookie := loginAdminResp.Header().Get("Set-Cookie")
	if adminCookie == "" {
		t.Fatalf("expected auth cookie for admin")
	}

	userPostResp := doRequestWithCookie(t, router, http.MethodPost, "/posts", `{"title":"user post","post_content":"x","published":1}`, userCookie)
	if userPostResp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, userPostResp.Code)
	}
	userPostID := extractCreatedID(t, userPostResp.Body.Bytes())

	adminPostResp := doRequestWithCookie(t, router, http.MethodPost, "/posts", `{"title":"admin post","post_content":"x","published":1}`, adminCookie)
	if adminPostResp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, adminPostResp.Code)
	}
	adminPostID := extractCreatedID(t, adminPostResp.Body.Bytes())

	userDeleteOtherResp := doRequestWithCookie(t, router, http.MethodDelete, "/posts/"+strconv.Itoa(adminPostID), "", userCookie)
	if userDeleteOtherResp.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, userDeleteOtherResp.Code)
	}

	userDeleteOwnResp := doRequestWithCookie(t, router, http.MethodDelete, "/posts/"+strconv.Itoa(userPostID), "", userCookie)
	if userDeleteOwnResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, userDeleteOwnResp.Code)
	}

	adminDeleteOtherResp := doRequestWithCookie(t, router, http.MethodDelete, "/posts/"+strconv.Itoa(adminPostID), "", adminCookie)
	if adminDeleteOtherResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, adminDeleteOtherResp.Code)
	}
}

// ---------------------------------------------------------------------------
// SQL Injection — vulnerable vs secure mode
// ---------------------------------------------------------------------------

func decodeSearchResults(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var payload struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode search response: %v (body=%s)", err, body)
	}
	return payload.Results
}

func TestIntegration_SearchSQLi_VulnerableMode(t *testing.T) {
	router, db, _ := setupIntegrationTestAppWithSecurity(t, false)
	t.Cleanup(func() { _ = db.Close() })

	// Seed an unpublished (draft) post that the secure search must hide.
	if _, err := db.Exec(
		"INSERT INTO blog (title, post_content, published, author_username) VALUES (?, ?, ?, ?)",
		"Internal draft", "Secret notes never to be published", 0, "admin",
	); err != nil {
		t.Fatalf("seed draft: %v", err)
	}

	t.Run("payload returns matched rows literally (no injection performed)", func(t *testing.T) {
		// A normal-looking term that doesn't match any row.
		resp := doRequest(t, router, http.MethodGet, "/api/search?q=zzz_nope", "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) != 0 {
			t.Fatalf("expected 0 results for unrelated term, got %d", len(results))
		}
	})

	t.Run("OR 1=1 leaks every row (including drafts)", func(t *testing.T) {
		// `' OR 1=1 --` closes the title literal and turns WHERE into a tautology.
		payload := url.QueryEscape("' OR 1=1 --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) < 3 {
			t.Fatalf("expected SQLi to leak ≥3 rows (incl. draft), got %d (body=%s)", len(results), resp.Body.String())
		}
		// The injected query bypasses the implicit `published = 1` filter, so the
		// "Internal draft" row must surface — which is the security failure we
		// are demonstrating.
		gotDraft := false
		for _, r := range results {
			if title, _ := r["title"].(string); title == "Internal draft" {
				gotDraft = true
				break
			}
		}
		if !gotDraft {
			t.Fatalf("expected draft row to leak through SQLi, body=%s", resp.Body.String())
		}
	})

	t.Run("UNION SELECT exfiltrates user table", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT id, username, password_hash, 1, '', '', '' FROM users --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) == 0 {
			t.Fatalf("expected UNION to surface rows, got 0 (body=%s)", resp.Body.String())
		}
		// One of the surfaced "titles" must be a username (admin/user1).
		gotUser := false
		for _, r := range results {
			if title, _ := r["title"].(string); title == "admin" || title == "user1" {
				gotUser = true
				break
			}
		}
		if !gotUser {
			t.Fatalf("expected username to leak as title via UNION, body=%s", resp.Body.String())
		}
	})
}

func TestIntegration_SearchSQLi_SecureMode(t *testing.T) {
	router, db, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(
		"INSERT INTO blog (title, post_content, published, author_username) VALUES (?, ?, ?, ?)",
		"Internal draft", "Secret notes never to be published", 0, "admin",
	); err != nil {
		t.Fatalf("seed draft: %v", err)
	}

	t.Run("OR 1=1 is treated as a literal pattern (no leak)", func(t *testing.T) {
		payload := url.QueryEscape("' OR 1=1 --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) != 0 {
			t.Fatalf("expected 0 results in secure mode, got %d (body=%s)", len(results), resp.Body.String())
		}
	})

	t.Run("UNION payload also returns no rows (parameterized)", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT id, username, password_hash, 1, '', '', '' FROM users --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) != 0 {
			t.Fatalf("expected 0 results for UNION in secure mode, got %d", len(results))
		}
	})

	t.Run("draft is never surfaced in secure mode", func(t *testing.T) {
		resp := doRequest(t, router, http.MethodGet, "/api/search?q=Secret", "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		for _, r := range results {
			if title, _ := r["title"].(string); title == "Internal draft" {
				t.Fatalf("secure search must filter to published=1, draft leaked: body=%s", resp.Body.String())
			}
		}
	})

	t.Run("force-vulnerable endpoint stays vulnerable even in secure mode", func(t *testing.T) {
		// /api/search-vulnerable ignores SECURITY_ENABLED — it always concatenates,
		// so the SQLi still works (this is intentional for the side-by-side demo).
		payload := url.QueryEscape("' OR 1=1 --")
		resp := doRequest(t, router, http.MethodGet, "/api/search-vulnerable?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) < 2 {
			t.Fatalf("force-vulnerable endpoint should still leak rows, got %d (body=%s)", len(results), resp.Body.String())
		}
	})
}

func TestIntegration_CreateDeleteRequireLogin_DefaultMode(t *testing.T) {
	router, db, _ := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	unauthCreateResp := doRequest(t, router, http.MethodPost, "/posts", `{"title":"x","post_content":"y","published":1}`)
	if unauthCreateResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, unauthCreateResp.Code)
	}

	loginResp := doRequest(t, router, http.MethodPost, "/login", `{"username":"admin","password":"anything"}`)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, loginResp.Code)
	}
	cookie := loginResp.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatalf("expected auth cookie after login")
	}

	authCreateResp := doRequestWithCookie(t, router, http.MethodPost, "/posts", `{"title":"secure create","post_content":"ok","published":1}`, cookie)
	if authCreateResp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, authCreateResp.Code)
	}
	createdID := extractCreatedID(t, authCreateResp.Body.Bytes())

	unauthDeleteResp := doRequest(t, router, http.MethodDelete, "/posts/"+strconv.Itoa(createdID), "")
	if unauthDeleteResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, unauthDeleteResp.Code)
	}
}

// ---- Stored XSS demo --------------------------------------------------------

const xssScriptPayload = `<script>alert('XSS-' + document.cookie)</script>`
const xssImgPayload = `<img src=x onerror="alert(1)">`

func postCommentForm(t *testing.T, router *gin.Engine, postID int, body string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{}
	form.Set("body", body)
	return doFormRequest(t, router, http.MethodPost, "/ui/posts/view/"+strconv.Itoa(postID)+"/comments", form.Encode())
}

func TestIntegration_StoredXSS_VulnerableMode(t *testing.T) {
	router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, false)
	t.Cleanup(func() { _ = dbConn.Close() })

	// Seed post #1 exists from SeedDB ("Hello World").
	postID := 1

	t.Run("script_payload_stored_and_rendered_raw", func(t *testing.T) {
		resp := postCommentForm(t, router, postID, xssScriptPayload)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		// Partial response is the comments list — must contain the raw payload
		// (not escaped) so the browser actually executes it.
		if !strings.Contains(resp.Body.String(), xssScriptPayload) {
			t.Fatalf("expected raw <script> tag in vulnerable comments partial; body=%s", resp.Body.String())
		}

		// Full page render must also include it raw.
		page := doRequest(t, router, http.MethodGet, "/ui/posts/view/"+strconv.Itoa(postID), "")
		if page.Code != http.StatusOK {
			t.Fatalf("expected 200 on detail page, got %d", page.Code)
		}
		if !strings.Contains(page.Body.String(), xssScriptPayload) {
			t.Fatalf("vulnerable detail page should embed raw payload; not found")
		}
	})

	t.Run("img_onerror_payload_stored_and_rendered_raw", func(t *testing.T) {
		resp := postCommentForm(t, router, postID, xssImgPayload)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		if !strings.Contains(resp.Body.String(), `onerror="alert(1)"`) {
			t.Fatalf("expected onerror attribute preserved; body=%s", resp.Body.String())
		}
	})
}

func TestIntegration_StoredXSS_SecureMode(t *testing.T) {
	router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = dbConn.Close() })

	postID := 1

	t.Run("script_payload_escaped_or_stripped", func(t *testing.T) {
		resp := postCommentForm(t, router, postID, xssScriptPayload)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		body := resp.Body.String()
		// Must NOT contain a raw executable <script> opener with the payload.
		if strings.Contains(body, "<script>alert(") {
			t.Fatalf("secure mode leaked raw <script> tag; body=%s", body)
		}
		// Verify the response was rendered (contains the comments-list anchor).
		if !strings.Contains(body, `id="comments-list"`) {
			t.Fatalf("expected comments-list in response; body=%s", body)
		}
	})

	t.Run("img_onerror_stripped_on_storage", func(t *testing.T) {
		resp := postCommentForm(t, router, postID, xssImgPayload)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		body := resp.Body.String()
		if strings.Contains(body, `onerror=`) {
			t.Fatalf("secure mode must not preserve onerror= as raw HTML; body=%s", body)
		}
	})

	t.Run("force_vulnerable_endpoint_still_stores_raw", func(t *testing.T) {
		// /api/comments-vulnerable bypasses the toggle.
		form := url.Values{}
		form.Set("post_id", strconv.Itoa(postID))
		form.Set("body", xssScriptPayload)
		resp := doFormRequest(t, router, http.MethodPost, "/api/comments-vulnerable", form.Encode())
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		// Verify the verbatim payload landed in DB.
		var stored string
		if err := dbConn.QueryRow(
			"SELECT body FROM comments WHERE post_id = ? ORDER BY id DESC LIMIT 1",
			postID,
		).Scan(&stored); err != nil {
			t.Fatalf("failed to read back comment: %v", err)
		}
		if stored != xssScriptPayload {
			t.Fatalf("force-vulnerable should preserve payload verbatim; got %q", stored)
		}
	})
}

// io.Discard is referenced indirectly; keep import alive.
var _ = io.Discard
