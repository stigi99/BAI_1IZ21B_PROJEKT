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
	"golang.org/x/crypto/bcrypt"
)

// postDTO mirrors the JSON payload used by the post API tests.
type postDTO struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	PostContent string `json:"post_content"`
	Published   int    `json:"published"`
}

// setupIntegrationTestApp creates an isolated test application with vulnerable
// mode disabled by default.
//
// Inputs:
//   - t: testing handle used for cleanup and fatal failures.
//
// Outputs:
//   - *gin.Engine: configured test router.
//   - *sql.DB: isolated SQLite connection used by the test.
//   - string: temporary database path.
func setupIntegrationTestApp(t *testing.T) (*gin.Engine, *sql.DB, string) {
	t.Helper()
	return setupIntegrationTestAppWithSecurity(t, false)
}

// setupIntegrationTestAppWithSecurity creates an isolated test application and
// lets the caller choose the active security mode.
//
// Inputs:
//   - t: testing handle used for cleanup and fatal failures.
//   - securityEnabled: initial security mode for the test application.
//
// Outputs:
//   - *gin.Engine: configured test router.
//   - *sql.DB: isolated SQLite connection used by the test.
//   - string: temporary database path.
func setupIntegrationTestAppWithSecurity(t *testing.T, securityEnabled bool) (*gin.Engine, *sql.DB, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration.db")
	db := dbpkg.InitDB(dbPath)

	dbpkg.MigrateDB(db)
	dbpkg.SeedDB(db, securityEnabled)

	router := buildRouter(db, securityEnabled)

	return router, db, dbPath
}

// doRequest sends a raw request against the test router and returns the recorder.
//
// Inputs:
//   - t: testing handle used for fatal failures.
//   - router: test Gin engine.
//   - method: HTTP method to send.
//   - path: request path.
//   - body: request body payload.
//
// Output:
//   - *httptest.ResponseRecorder: captured response.
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

// doFormRequest sends an application/x-www-form-urlencoded request.
//
// Inputs:
//   - t: testing handle used for fatal failures.
//   - router: test Gin engine.
//   - method: HTTP method to send.
//   - path: request path.
//   - body: form-encoded body payload.
//
// Output:
//   - *httptest.ResponseRecorder: captured response.
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

// doFormRequestWithCookie sends a form request with a Cookie header attached.
//
// Inputs:
//   - t: testing handle used for fatal failures.
//   - router: test Gin engine.
//   - method: HTTP method to send.
//   - path: request path.
//   - body: form-encoded body payload.
//   - cookie: Cookie header value to attach.
//
// Output:
//   - *httptest.ResponseRecorder: captured response.
func doFormRequestWithCookie(t *testing.T, router *gin.Engine, method, path, body, cookie string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doRequestWithCookie sends a raw request with a Cookie header attached.
//
// Inputs:
//   - t: testing handle used for fatal failures.
//   - router: test Gin engine.
//   - method: HTTP method to send.
//   - path: request path.
//   - body: request body payload.
//   - cookie: Cookie header value to attach.
//
// Output:
//   - *httptest.ResponseRecorder: captured response.
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

// responseCookieValue extracts a named cookie value from a response recorder.
//
// Inputs:
//   - t: testing handle used for fatal failures.
//   - resp: response recorder to inspect.
//   - name: cookie name to look up.
//
// Output:
//   - string: cookie value when present.
func responseCookieValue(t *testing.T, resp *httptest.ResponseRecorder, name string) string {
	t.Helper()
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	t.Fatalf("missing cookie %q in %v", name, resp.Header().Values("Set-Cookie"))
	return ""
}

// joinSetCookies turns Set-Cookie response headers into a single value suitable
// for re-sending as `Cookie: ...` on subsequent requests.
func joinSetCookies(resp *httptest.ResponseRecorder) string {
	parts := []string{}
	for _, c := range resp.Result().Cookies() {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

// extractCreatedID parses the numeric id field from a create-response payload.
//
// Inputs:
//   - t: testing handle used for fatal failures.
//   - body: response body containing JSON.
//
// Output:
//   - int: parsed identifier.
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

// TestIntegration_Ping verifies that the basic health endpoint responds with pong.
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

// TestIntegration_Posts verifies the published-posts API and create flow.
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

// TestIntegration_Login verifies login behavior through the JSON API.
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

// TestIntegration_DatabaseFileIsCreated verifies that initialization creates the DB file.
// TestIntegration_DatabaseFileIsCreated verifies that initialization creates the DB file.
func TestIntegration_DatabaseFileIsCreated(t *testing.T) {
	_, db, dbPath := setupIntegrationTestApp(t)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected DB file to exist at %s: %v", dbPath, err)
	}
}

// TestIntegration_UI_PostsPage verifies that the posts page renders successfully.
// TestIntegration_UI_PostsPage verifies that the posts page renders successfully.
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
	if !strings.Contains(body, "Create fan post") {
		t.Fatalf("expected posts page marker in body, got: %s", body)
	}
}

// TestIntegration_UI_LoginPage verifies that the login page renders successfully.
// TestIntegration_UI_LoginPage verifies that the login page renders successfully.
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

// TestIntegration_UI_LoginSubmit verifies the browser login submission flow.
// TestIntegration_UI_LoginSubmit verifies the browser login submission flow.
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

// TestIntegration_UI_PostsPartial verifies the HTMX posts fragment.
// TestIntegration_UI_PostsPartial verifies the HTMX posts fragment.
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

// TestIntegration_UI_LoginPartial verifies the HTMX login fragment.
// TestIntegration_UI_LoginPartial verifies the HTMX login fragment.
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

// TestIntegration_UIModeToggle verifies the runtime secure/vulnerable toggle.
// TestIntegration_UIModeToggle verifies the runtime secure/vulnerable toggle.
func TestIntegration_UIModeToggle(t *testing.T) {
	router, db, _ := setupIntegrationTestAppWithSecurity(t, false)
	t.Cleanup(func() { _ = db.Close() })

	page := doRequest(t, router, http.MethodGet, "/ui/search", "")
	if page.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, page.Code)
	}
	if !strings.Contains(page.Body.String(), "Vulnerable") || !strings.Contains(page.Body.String(), "↔ Secure") {
		t.Fatalf("expected vulnerable badge and secure switch button, got: %s", page.Body.String())
	}

	toggleResp := doFormRequest(t, router, http.MethodPost, "/ui/mode/toggle", "next=%2Fui%2Fsearch")
	if toggleResp.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, toggleResp.Code)
	}
	if location := toggleResp.Header().Get("Location"); location != "/ui/search" {
		t.Fatalf("expected redirect back to /ui/search, got %q", location)
	}

	securePage := doRequest(t, router, http.MethodGet, "/ui/search", "")
	if securePage.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, securePage.Code)
	}
	if !strings.Contains(securePage.Body.String(), "Secure") || !strings.Contains(securePage.Body.String(), "↔ Vulnerable") {
		t.Fatalf("expected secure badge and vulnerable switch button, got: %s", securePage.Body.String())
	}

	payload := url.QueryEscape("zz' OR EXISTS(SELECT 1 FROM users WHERE username='admin' AND substr(password_hash,1,1)='a') --")
	searchResp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
	if searchResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, searchResp.Code)
	}
	results := decodeSearchResults(t, searchResp.Body.Bytes())
	if len(results) != 0 {
		t.Fatalf("expected secure search after toggle to block SQLi, got %d results", len(results))
	}
}

// TestIntegration_SeedDB_RewritesDemoCredentialsForSecurityMode verifies that seeding respects the active mode.
// TestIntegration_SeedDB_RewritesDemoCredentialsForSecurityMode verifies that seeding respects the active mode.
func TestIntegration_SeedDB_RewritesDemoCredentialsForSecurityMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "seed-mode.db")
	dbConn := dbpkg.InitDB(dbPath)
	t.Cleanup(func() { _ = dbConn.Close() })

	dbpkg.MigrateDB(dbConn)
	dbpkg.SeedDB(dbConn, false)

	var vulnerableStored string
	if err := dbConn.QueryRow("SELECT password_hash FROM users WHERE username = 'admin'").Scan(&vulnerableStored); err != nil {
		t.Fatalf("read vulnerable admin password: %v", err)
	}
	if vulnerableStored != "admin" {
		t.Fatalf("expected vulnerable seed to store plaintext admin password, got %q", vulnerableStored)
	}

	dbpkg.SeedDB(dbConn, true)

	var secureStored string
	if err := dbConn.QueryRow("SELECT password_hash FROM users WHERE username = 'admin'").Scan(&secureStored); err != nil {
		t.Fatalf("read secure admin password: %v", err)
	}
	if secureStored == "admin" {
		t.Fatalf("secure re-seed must not leave plaintext admin password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(secureStored), []byte("admin")); err != nil {
		t.Fatalf("secure re-seed should store a bcrypt hash for admin password: %v", err)
	}
}

// TestIntegration_SecureAuthCookieIsSignedAndSameSiteStrict verifies the secure login cookie settings.
// TestIntegration_SecureAuthCookieIsSignedAndSameSiteStrict verifies the secure login cookie settings.
func TestIntegration_SecureAuthCookieIsSignedAndSameSiteStrict(t *testing.T) {
	router, db, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = db.Close() })

	resp := doRequest(t, router, http.MethodPost, "/login", `{"username":"admin","password":"admin"}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	setCookie := resp.Header().Get("Set-Cookie")
	authValue := responseCookieValue(t, resp, "bai_auth_user")
	decodedAuthValue, err := url.QueryUnescape(authValue)
	if err != nil {
		t.Fatalf("decode auth cookie value: %v", err)
	}
	if !strings.HasPrefix(decodedAuthValue, "admin|") {
		t.Fatalf("expected signed auth cookie value, got %q", authValue)
	}
	if !strings.Contains(setCookie, "HttpOnly") {
		t.Fatalf("expected HttpOnly auth cookie, got %q", setCookie)
	}
	if !strings.Contains(setCookie, "SameSite=Strict") {
		t.Fatalf("expected SameSite=Strict auth cookie, got %q", setCookie)
	}

	forgedCookie := "bai_auth_user=admin"
	forgedResp := doRequestWithCookie(t, router, http.MethodPost, "/posts", `{"title":"forged","post_content":"x","published":1}`, forgedCookie)
	if forgedResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected forged unsigned cookie to be rejected, got %d (body=%s)", forgedResp.Code, forgedResp.Body.String())
	}
}

// TestIntegration_CSRFSecureFormRejectsAndAcceptsTokens verifies CSRF token enforcement.
// TestIntegration_CSRFSecureFormRejectsAndAcceptsTokens verifies CSRF token enforcement.
func TestIntegration_CSRFSecureFormRejectsAndAcceptsTokens(t *testing.T) {
	router, db, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = db.Close() })

	loginResp := doRequest(t, router, http.MethodPost, "/login", `{"username":"admin","password":"admin"}`)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginResp.Code)
	}
	authValue := responseCookieValue(t, loginResp, "bai_auth_user")
	authCookie := "bai_auth_user=" + authValue

	rejectResp := doFormRequestWithCookie(t, router, http.MethodPost, "/ui/csrf-secure", "new_email=blocked%40example.com", authCookie)
	if rejectResp.Code != http.StatusForbidden {
		t.Fatalf("expected missing CSRF token to be rejected, got %d", rejectResp.Code)
	}
	if !strings.Contains(rejectResp.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected CSRF reject to render HTML UI, got %s", rejectResp.Header().Get("Content-Type"))
	}
	if !strings.Contains(rejectResp.Body.String(), "CSRF token validation failed") {
		t.Fatalf("expected CSRF validation message, got: %s", rejectResp.Body.String())
	}

	formResp := doRequestWithCookie(t, router, http.MethodGet, "/ui/csrf-secure", "", authCookie)
	if formResp.Code != http.StatusOK {
		t.Fatalf("expected CSRF form status %d, got %d", http.StatusOK, formResp.Code)
	}
	csrfValue := responseCookieValue(t, formResp, "bai_csrf_token")
	validCookie := authCookie + "; bai_csrf_token=" + csrfValue

	body := "new_email=secure%40example.com&csrf_token=" + url.QueryEscape(csrfValue)
	acceptResp := doFormRequestWithCookie(t, router, http.MethodPost, "/ui/csrf-secure", body, validCookie)
	if acceptResp.Code != http.StatusOK {
		t.Fatalf("expected valid CSRF token to be accepted, got %d (body=%s)", acceptResp.Code, acceptResp.Body.String())
	}
	if !strings.Contains(acceptResp.Body.String(), "Email updated to secure@example.com") {
		t.Fatalf("expected successful email update message, got: %s", acceptResp.Body.String())
	}
}

// TestIntegration_ProfileEndpointHonorsSecurityModeForCSRF verifies the profile form dispatch.
// TestIntegration_ProfileEndpointHonorsSecurityModeForCSRF verifies the profile form dispatch.
func TestIntegration_ProfileEndpointHonorsSecurityModeForCSRF(t *testing.T) {
	t.Run("vulnerable mode accepts forged profile POST without token", func(t *testing.T) {
		router, db, _ := setupIntegrationTestAppWithSecurity(t, false)
		t.Cleanup(func() { _ = db.Close() })

		loginResp := doRequest(t, router, http.MethodPost, "/login", `{"username":"admin","password":"anything"}`)
		if loginResp.Code != http.StatusOK {
			t.Fatalf("expected login status %d, got %d", http.StatusOK, loginResp.Code)
		}
		authCookie := loginResp.Header().Get("Set-Cookie")

		resp := doFormRequestWithCookie(t, router, http.MethodPost, "/ui/profile", "new_email=csrf-pwned%40evil.test", authCookie)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected vulnerable profile update to succeed, got %d body=%s", resp.Code, resp.Body.String())
		}

		var email string
		if err := db.QueryRow("SELECT email FROM users WHERE username = 'admin'").Scan(&email); err != nil {
			t.Fatalf("query admin email: %v", err)
		}
		if email != "csrf-pwned@evil.test" {
			t.Fatalf("expected forged profile POST to update email, got %q", email)
		}
	})

	t.Run("secure mode rejects same profile POST without token", func(t *testing.T) {
		router, db, _ := setupIntegrationTestAppWithSecurity(t, true)
		t.Cleanup(func() { _ = db.Close() })

		loginResp := doRequest(t, router, http.MethodPost, "/login", `{"username":"admin","password":"admin"}`)
		if loginResp.Code != http.StatusOK {
			t.Fatalf("expected login status %d, got %d", http.StatusOK, loginResp.Code)
		}
		authValue := responseCookieValue(t, loginResp, "bai_auth_user")
		authCookie := "bai_auth_user=" + authValue

		rejectResp := doFormRequestWithCookie(t, router, http.MethodPost, "/ui/profile", "new_email=blocked%40evil.test", authCookie)
		if rejectResp.Code != http.StatusForbidden {
			t.Fatalf("expected secure profile POST without token to be rejected, got %d body=%s", rejectResp.Code, rejectResp.Body.String())
		}

		var email string
		if err := db.QueryRow("SELECT email FROM users WHERE username = 'admin'").Scan(&email); err != nil {
			t.Fatalf("query admin email: %v", err)
		}
		if email == "blocked@evil.test" {
			t.Fatal("secure profile endpoint updated email despite missing CSRF token")
		}

		formResp := doRequestWithCookie(t, router, http.MethodGet, "/ui/profile", "", authCookie)
		if formResp.Code != http.StatusOK {
			t.Fatalf("expected profile form status %d, got %d", http.StatusOK, formResp.Code)
		}
		csrfValue := responseCookieValue(t, formResp, "bai_csrf_token")
		validCookie := authCookie + "; bai_csrf_token=" + csrfValue

		body := "new_email=secure-profile%40example.com&csrf_token=" + url.QueryEscape(csrfValue)
		acceptResp := doFormRequestWithCookie(t, router, http.MethodPost, "/ui/profile", body, validCookie)
		if acceptResp.Code != http.StatusOK {
			t.Fatalf("expected secure profile POST with token to succeed, got %d body=%s", acceptResp.Code, acceptResp.Body.String())
		}
	})
}

// TestIntegration_Register verifies user registration through the API.
// TestIntegration_Register verifies user registration through the API.
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

// TestIntegration_DeleteAuthorization_SecurityEnabled verifies owner/admin delete checks in secure mode.
// TestIntegration_DeleteAuthorization_SecurityEnabled verifies owner/admin delete checks in secure mode.
func TestIntegration_DeleteAuthorization_SecurityEnabled(t *testing.T) {
	router, db, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = db.Close() })

	loginUserResp := doRequest(t, router, http.MethodPost, "/login", `{"username":"user1","password":"user1pass"}`)
	if loginUserResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, loginUserResp.Code)
	}
	userCookie := joinSetCookies(loginUserResp)
	if userCookie == "" {
		t.Fatalf("expected auth cookie for user")
	}

	loginAdminResp := doRequest(t, router, http.MethodPost, "/login", `{"username":"admin","password":"admin"}`)
	if loginAdminResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, loginAdminResp.Code)
	}
	adminCookie := joinSetCookies(loginAdminResp)
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

// decodeSearchResults parses the JSON payload returned by the search endpoints.
//
// Inputs:
//   - t: testing handle used for fatal failures.
//   - body: response body containing JSON.
//
// Output:
//   - []map[string]any: decoded search result rows.
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

// TestIntegration_SearchSQLi_VulnerableMode verifies the force-vulnerable SQLi demo.
// TestIntegration_SearchSQLi_VulnerableMode verifies the force-vulnerable SQLi demo.
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

	t.Run("ORDER BY probe leaks SELECT shape through SQL error", func(t *testing.T) {
		payload := url.QueryEscape("zz' ORDER BY 8 --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected vulnerable ORDER BY probe to fail with 500, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		if !strings.Contains(resp.Body.String(), "ORDER BY") && !strings.Contains(resp.Body.String(), "range") {
			t.Fatalf("expected database error to reveal ORDER BY/column-count detail, body=%s", resp.Body.String())
		}
	})

	t.Run("boolean probe confirms admin password prefix and leaks rows", func(t *testing.T) {
		// This is more realistic than a bare tautology: the attacker asks the
		// database a yes/no question about another table. In vulnerable mode the
		// true predicate expands the result set and confirms the guessed prefix.
		payload := url.QueryEscape("zz' OR EXISTS(SELECT 1 FROM users WHERE username='admin' AND substr(password_hash,1,1)='a') --")
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
		payload := url.QueryEscape("zz' UNION SELECT id, '[user] ' || username, 'role=' || role || ' email=' || email || ' secret=' || password_hash, 1, username, '', '' FROM users --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) == 0 {
			t.Fatalf("expected UNION to surface rows, got 0 (body=%s)", resp.Body.String())
		}
		// One of the surfaced "titles" must be a tagged username (admin/user1).
		gotUser := false
		for _, r := range results {
			if title, _ := r["title"].(string); title == "[user] admin" || title == "[user] user1" {
				gotUser = true
				break
			}
		}
		if !gotUser {
			t.Fatalf("expected username to leak as title via UNION, body=%s", resp.Body.String())
		}
	})

	t.Run("UNION SELECT enumerates sqlite schema", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT 1, name, sql, 1, '', '', '' FROM sqlite_master WHERE type='table' --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		gotUsersTable := false
		for _, r := range results {
			if title, _ := r["title"].(string); title == "users" {
				gotUsersTable = true
				break
			}
		}
		if !gotUsersTable {
			t.Fatalf("expected sqlite_master UNION to reveal users table, body=%s", resp.Body.String())
		}
	})

	t.Run("UNION SELECT leaks hidden drafts directly", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT id, '[draft] ' || title, post_content, published, author_username, '', '' FROM blog WHERE published=0 --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		gotDraft := false
		for _, r := range results {
			if title, _ := r["title"].(string); title == "[draft] Internal draft" {
				gotDraft = true
				break
			}
		}
		if !gotDraft {
			t.Fatalf("expected direct draft UNION leak, body=%s", resp.Body.String())
		}
	})

	t.Run("UNION SELECT builds database intelligence summary row", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT 9000, '[intel] database map', 'users=' || (SELECT COUNT(*) FROM users) || ' tables=' || (SELECT group_concat(name, ', ') FROM sqlite_master WHERE type='table'), 1, 'sqli-bot', '', '' --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		gotSummary := false
		for _, r := range results {
			title, _ := r["title"].(string)
			content, _ := r["post_content"].(string)
			if title == "[intel] database map" && strings.Contains(content, "users=") && strings.Contains(content, "blog") {
				gotSummary = true
				break
			}
		}
		if !gotSummary {
			t.Fatalf("expected injected intelligence summary row, body=%s", resp.Body.String())
		}
	})

	t.Run("UNION SELECT fingerprints SQLite engine", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT 9001, '[fingerprint] SQLite ' || sqlite_version(), sqlite_source_id(), 1, 'sqli-bot', '', '' --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		gotFingerprint := false
		for _, r := range results {
			if title, _ := r["title"].(string); strings.HasPrefix(title, "[fingerprint] SQLite ") {
				gotFingerprint = true
				break
			}
		}
		if !gotFingerprint {
			t.Fatalf("expected SQLite fingerprint row, body=%s", resp.Body.String())
		}
	})

	t.Run("UNION SELECT pivots admin row into hidden draft summary", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT 9002, '[pivot] ' || username || ' owns drafts', (SELECT group_concat(title, ' | ') FROM blog WHERE published=0), 1, username, '', '' FROM users WHERE role='admin' --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		gotPivot := false
		for _, r := range results {
			title, _ := r["title"].(string)
			content, _ := r["post_content"].(string)
			if title == "[pivot] admin owns drafts" && strings.Contains(content, "Internal draft") {
				gotPivot = true
				break
			}
		}
		if !gotPivot {
			t.Fatalf("expected admin-to-draft pivot row, body=%s", resp.Body.String())
		}
	})
}

// TestIntegration_SearchSQLi_SecureMode verifies the parameterized secure search path.
// TestIntegration_SearchSQLi_SecureMode verifies the parameterized secure search path.
func TestIntegration_SearchSQLi_SecureMode(t *testing.T) {
	router, db, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(
		"INSERT INTO blog (title, post_content, published, author_username) VALUES (?, ?, ?, ?)",
		"Internal draft", "Secret notes never to be published", 0, "admin",
	); err != nil {
		t.Fatalf("seed draft: %v", err)
	}

	t.Run("boolean probe is treated as a literal pattern (no leak)", func(t *testing.T) {
		payload := url.QueryEscape("zz' OR EXISTS(SELECT 1 FROM users WHERE username='admin' AND substr(password_hash,1,1)='a') --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) != 0 {
			t.Fatalf("expected 0 results in secure mode, got %d (body=%s)", len(results), resp.Body.String())
		}
	})

	t.Run("ORDER BY probe is treated as literal text, not SQL syntax", func(t *testing.T) {
		payload := url.QueryEscape("zz' ORDER BY 8 --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 in secure mode, got %d (body=%s)", resp.Code, resp.Body.String())
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) != 0 {
			t.Fatalf("expected 0 results for literal ORDER BY probe in secure mode, got %d", len(results))
		}
	})

	t.Run("UNION payload also returns no rows (parameterized)", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT id, '[user] ' || username, 'role=' || role || ' email=' || email || ' secret=' || password_hash, 1, username, '', '' FROM users --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) != 0 {
			t.Fatalf("expected 0 results for UNION in secure mode, got %d", len(results))
		}
	})

	t.Run("sqlite_master UNION payload also returns no rows (parameterized)", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT 1, name, sql, 1, '', '', '' FROM sqlite_master WHERE type='table' --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) != 0 {
			t.Fatalf("expected 0 results for sqlite_master UNION in secure mode, got %d", len(results))
		}
	})

	t.Run("hidden draft UNION payload also returns no rows (parameterized)", func(t *testing.T) {
		payload := url.QueryEscape("zz' UNION SELECT id, '[draft] ' || title, post_content, published, author_username, '', '' FROM blog WHERE published=0 --")
		resp := doRequest(t, router, http.MethodGet, "/api/search?q="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := decodeSearchResults(t, resp.Body.Bytes())
		if len(results) != 0 {
			t.Fatalf("expected 0 results for hidden draft UNION in secure mode, got %d", len(results))
		}
	})

	t.Run("advanced UNION payloads also return no rows (parameterized)", func(t *testing.T) {
		payloads := []string{
			"zz' UNION SELECT 9000, '[intel] database map', 'users=' || (SELECT COUNT(*) FROM users) || ' tables=' || (SELECT group_concat(name, ', ') FROM sqlite_master WHERE type='table'), 1, 'sqli-bot', '', '' --",
			"zz' UNION SELECT 9001, '[fingerprint] SQLite ' || sqlite_version(), sqlite_source_id(), 1, 'sqli-bot', '', '' --",
			"zz' UNION SELECT 9002, '[pivot] ' || username || ' owns drafts', (SELECT group_concat(title, ' | ') FROM blog WHERE published=0), 1, username, '', '' FROM users WHERE role='admin' --",
		}
		for _, p := range payloads {
			resp := doRequest(t, router, http.MethodGet, "/api/search?q="+url.QueryEscape(p), "")
			if resp.Code != http.StatusOK {
				t.Fatalf("expected 200 for secure advanced payload, got %d (payload=%s body=%s)", resp.Code, p, resp.Body.String())
			}
			results := decodeSearchResults(t, resp.Body.Bytes())
			if len(results) != 0 {
				t.Fatalf("expected 0 results for secure advanced payload, got %d (payload=%s)", len(results), p)
			}
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
		payload := url.QueryEscape("zz' UNION SELECT id, '[user] ' || username, 'role=' || role || ' email=' || email || ' secret=' || password_hash, 1, username, '', '' FROM users --")
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

// TestIntegration_CreateDeleteRequireLogin_DefaultMode verifies login is required for write actions.
// TestIntegration_CreateDeleteRequireLogin_DefaultMode verifies login is required for write actions.
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
	cookie := joinSetCookies(loginResp)
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

	unauthUpdateResp := doRequest(t, router, http.MethodPut, "/posts/"+strconv.Itoa(createdID), `{"title":"updated","post_content":"changed","published":1}`)
	if unauthUpdateResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, unauthUpdateResp.Code)
	}

	authUpdateResp := doRequestWithCookie(t, router, http.MethodPut, "/posts/"+strconv.Itoa(createdID), `{"title":"updated","post_content":"changed","published":1}`, cookie)
	if authUpdateResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, authUpdateResp.Code)
	}
}

// ---- Stored XSS demo --------------------------------------------------------

const xssScriptPayload = `<script>alert('XSS-' + document.cookie)</script>`
const xssImgPayload = `<img src=x onerror="alert(1)">`
const xssOverlayPayload = `<img src=x onerror="document.body.insertAdjacentHTML('afterbegin','<div style=&quot;position:fixed;inset:0;z-index:9999;background:#111827;color:white;padding:40px&quot;><h1>Session expired</h1><p>Fake re-login overlay injected from a comment.</p></div>')">`

// postCommentForm submits a form-based comment against the post detail route.
//
// Inputs:
//   - t: testing handle used for fatal failures.
//   - router: test Gin engine.
//   - postID: target post identifier.
//   - body: comment body to submit.
//
// Output:
//   - *httptest.ResponseRecorder: captured response.
func postCommentForm(t *testing.T, router *gin.Engine, postID int, body string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{}
	form.Set("body", body)
	return doFormRequest(t, router, http.MethodPost, "/ui/posts/view/"+strconv.Itoa(postID)+"/comments", form.Encode())
}

// TestIntegration_StoredXSS_VulnerableMode verifies the raw-comment XSS demo.
// TestIntegration_StoredXSS_VulnerableMode verifies the raw-comment XSS demo.
func TestIntegration_StoredXSS_VulnerableMode(t *testing.T) {
	router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, false)
	t.Cleanup(func() { _ = dbConn.Close() })

	postID := 1

	t.Run("script_payload_stored_and_rendered_raw", func(t *testing.T) {
		resp := postCommentForm(t, router, postID, xssScriptPayload)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 comments fragment, got %d (body=%s)", resp.Code, resp.Body.String())
		}

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
			t.Fatalf("expected 200 comments fragment, got %d", resp.Code)
		}
		page := doRequest(t, router, http.MethodGet, "/ui/posts/view/"+strconv.Itoa(postID), "")
		if !strings.Contains(page.Body.String(), `onerror="alert(1)"`) {
			t.Fatalf("expected onerror attribute preserved on detail page")
		}
	})

	t.Run("overlay_payload_can_take_over_reader_ui", func(t *testing.T) {
		resp := postCommentForm(t, router, postID, xssOverlayPayload)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		body := resp.Body.String()
		if !strings.Contains(body, "insertAdjacentHTML") || !strings.Contains(body, "Fake re-login overlay") {
			t.Fatalf("expected UI takeover payload to be preserved raw; body=%s", body)
		}
	})
}

// TestIntegration_StoredXSS_SecureMode verifies that comments are escaped in secure mode.
// TestIntegration_StoredXSS_SecureMode verifies that comments are escaped in secure mode.
func TestIntegration_StoredXSS_SecureMode(t *testing.T) {
	router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = dbConn.Close() })

	postID := 1

	t.Run("script_payload_escaped_or_stripped", func(t *testing.T) {
		resp := postCommentForm(t, router, postID, xssScriptPayload)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 comments fragment, got %d", resp.Code)
		}
		page := doRequest(t, router, http.MethodGet, "/ui/posts/view/"+strconv.Itoa(postID), "")
		body := page.Body.String()
		// Must NOT contain a raw executable <script> opener with the payload.
		if strings.Contains(body, "<script>alert('XSS-") {
			t.Fatalf("secure mode leaked raw <script> tag in detail page body")
		}
	})

	t.Run("img_onerror_stripped_on_storage", func(t *testing.T) {
		resp := postCommentForm(t, router, postID, xssImgPayload)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 comments fragment, got %d", resp.Code)
		}
		page := doRequest(t, router, http.MethodGet, "/ui/posts/view/"+strconv.Itoa(postID), "")
		body := page.Body.String()
		if strings.Contains(body, `onerror="alert(1)"`) {
			t.Fatalf("secure mode must not render raw onerror= attribute on detail page")
		}
	})

	t.Run("overlay_payload_is_rendered_inert", func(t *testing.T) {
		resp := postCommentForm(t, router, postID, xssOverlayPayload)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		body := resp.Body.String()
		if strings.Contains(body, "insertAdjacentHTML") || strings.Contains(body, `onerror=`) {
			t.Fatalf("secure mode must not preserve UI takeover JavaScript; body=%s", body)
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

// TestIntegration_PathTraversal_LFI verifies the vulnerable and secure file read flows.
// TestIntegration_PathTraversal_LFI verifies the vulnerable and secure file read flows.
func TestIntegration_PathTraversal_LFI(t *testing.T) {
	t.Run("vulnerable endpoint reads outside uploads", func(t *testing.T) {
		router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, false)
		t.Cleanup(func() { _ = dbConn.Close() })

		resp := doRequest(t, router, http.MethodGet, "/api/files-vulnerable?name=../go.mod", "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d (body=%s)", http.StatusOK, resp.Code, resp.Body.String())
		}
		if !strings.Contains(resp.Body.String(), "module BAI_1IZ21B_PROJEKT") {
			t.Fatalf("expected vulnerable endpoint to read go.mod, got: %s", resp.Body.String())
		}
	})

	t.Run("vulnerable endpoint leaks source code through traversal", func(t *testing.T) {
		router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, false)
		t.Cleanup(func() { _ = dbConn.Close() })

		resp := doRequest(t, router, http.MethodGet, "/api/files-vulnerable?name=../internal/db/db.go", "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d (body=%s)", http.StatusOK, resp.Code, resp.Body.String())
		}
		if !strings.Contains(resp.Body.String(), "func SeedDB") || !strings.Contains(resp.Body.String(), "Unpublished Magical Mirai") {
			t.Fatalf("expected source disclosure to expose seed logic, got: %s", resp.Body.String())
		}
	})

	t.Run("secure endpoint blocks traversal", func(t *testing.T) {
		router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, true)
		t.Cleanup(func() { _ = dbConn.Close() })

		resp := doRequest(t, router, http.MethodGet, "/api/files-secure?name=../go.mod", "")
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d (body=%s)", http.StatusBadRequest, resp.Code, resp.Body.String())
		}
		if !strings.Contains(resp.Body.String(), "path traversal detected") {
			t.Fatalf("expected traversal block message, got: %s", resp.Body.String())
		}
	})

	t.Run("secure endpoint blocks source-code traversal", func(t *testing.T) {
		router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, true)
		t.Cleanup(func() { _ = dbConn.Close() })

		resp := doRequest(t, router, http.MethodGet, "/api/files-secure?name=../internal/db/db.go", "")
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d (body=%s)", http.StatusBadRequest, resp.Code, resp.Body.String())
		}
	})
}

// TestIntegration_CommandInjection verifies the vulnerable and secure ping flows.
// TestIntegration_CommandInjection verifies the vulnerable and secure ping flows.
func TestIntegration_CommandInjection(t *testing.T) {
	t.Run("vulnerable endpoint executes appended shell command", func(t *testing.T) {
		router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, false)
		t.Cleanup(func() { _ = dbConn.Close() })

		payload := url.QueryEscape("127.0.0.1; echo BAI_CMD_INJECTION")
		resp := doRequest(t, router, http.MethodGet, "/api/ping-vulnerable?host="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
		}
		if !strings.Contains(resp.Body.String(), "BAI_CMD_INJECTION") {
			t.Fatalf("expected injected command output, got: %s", resp.Body.String())
		}
	})

	t.Run("vulnerable endpoint executes chained reconnaissance commands", func(t *testing.T) {
		router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, false)
		t.Cleanup(func() { _ = dbConn.Close() })

		payload := url.QueryEscape("127.0.0.1; printf BAI_CHAIN_MARKER; uname -s")
		resp := doRequest(t, router, http.MethodGet, "/api/ping-vulnerable?host="+payload, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
		}
		if !strings.Contains(resp.Body.String(), "BAI_CHAIN_MARKER") {
			t.Fatalf("expected chained command marker, got: %s", resp.Body.String())
		}
	})

	t.Run("secure endpoint rejects shell metacharacters", func(t *testing.T) {
		router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, true)
		t.Cleanup(func() { _ = dbConn.Close() })

		payload := url.QueryEscape("127.0.0.1; echo BAI_CMD_INJECTION")
		resp := doRequest(t, router, http.MethodGet, "/api/ping-secure?host="+payload, "")
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d (body=%s)", http.StatusBadRequest, resp.Code, resp.Body.String())
		}
		if !strings.Contains(resp.Body.String(), "invalid host") {
			t.Fatalf("expected invalid host message, got: %s", resp.Body.String())
		}
	})
}

// io.Discard is referenced indirectly; keep import alive.
var _ = io.Discard

// ---- Security Misconfiguration ---------------------------------------------

func TestIntegration_SecurityHeaders_VulnerableMode(t *testing.T) {
	router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, false)
	t.Cleanup(func() { _ = dbConn.Close() })

	resp := doRequest(t, router, http.MethodGet, "/ui/posts", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	// In vuln mode none of the defensive headers should be set.
	for _, h := range []string{
		"Content-Security-Policy",
		"Strict-Transport-Security",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Permissions-Policy",
	} {
		if got := resp.Header().Get(h); got != "" {
			t.Errorf("vuln mode should NOT set %s, got %q", h, got)
		}
	}
}

func TestIntegration_SecurityHeaders_SecureMode(t *testing.T) {
	router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = dbConn.Close() })

	resp := doRequest(t, router, http.MethodGet, "/ui/posts", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	expectations := map[string]string{
		"X-Frame-Options":           "DENY",
		"X-Content-Type-Options":    "nosniff",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}
	for h, want := range expectations {
		if got := resp.Header().Get(h); got != want {
			t.Errorf("expected %s=%q, got %q", h, want, got)
		}
	}

	// CSP just needs to be present and contain a few key directives.
	csp := resp.Header().Get("Content-Security-Policy")
	for _, expect := range []string{"default-src 'self'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, expect) {
			t.Errorf("CSP missing %q; got: %s", expect, csp)
		}
	}
}

func TestIntegration_DebugCrash_LeaksInVulnerableMode(t *testing.T) {
	router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, false)
	t.Cleanup(func() { _ = dbConn.Close() })

	resp := doRequest(t, router, http.MethodGet, "/debug/crash", "")
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
	body := resp.Body.String()
	// Gin's default Recovery returns an empty body but writes a colourful
	// stack trace to stderr. Either way, the key fact for the demo is that
	// vuln mode does NOT return a sanitized JSON envelope.
	if strings.Contains(body, `"error":"Internal server error"`) {
		t.Fatalf("vuln mode should not return sanitized JSON; got: %s", body)
	}
}

func TestIntegration_DebugCrash_SanitizedInSecureMode(t *testing.T) {
	router, dbConn, _ := setupIntegrationTestAppWithSecurity(t, true)
	t.Cleanup(func() { _ = dbConn.Close() })

	resp := doRequest(t, router, http.MethodGet, "/debug/crash", "")
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"error":"Internal server error"`) {
		t.Fatalf("secure mode should return sanitized JSON envelope; got: %s", body)
	}
	// Must not leak stack trace fragments.
	for _, leak := range []string{"goroutine", "panic:", "runtime error", "/internal/handlers/"} {
		if strings.Contains(body, leak) {
			t.Errorf("secure mode response should not contain %q; got: %s", leak, body)
		}
	}
}
