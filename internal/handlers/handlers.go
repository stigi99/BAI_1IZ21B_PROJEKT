package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"BAI_1IZ21B_PROJEKT/internal/service"
	"BAI_1IZ21B_PROJEKT/internal/views"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

const (
	authCookieName = "bai_auth_user"
	csrfCookieName = "bai_csrf_token"
	uploadsDir     = "./uploads"
	uploadsURLPath = "/uploads"
	maxUploadBytes = 5 << 20 // 5 MiB
	// fallbackSessionSecret is used only when BAI_SESSION_SECRET is unset.
	// In a production system this must come from a securely stored secret.
	fallbackSessionSecret = "bai-lab-demo-secret-do-not-use-in-prod"
	sessionSecretEnv      = "BAI_SESSION_SECRET"
	securePingTimeout     = 3 * time.Second
)

type Handler struct {
	svc *service.Service
}

// New constructs a handler bundle backed by the provided service and initial
// security mode.
//
// Inputs:
//   - svc: shared application service used by all route handlers.
//   - securityEnabled: initial lab mode to apply to the service.
//
// Output:
//   - *Handler: ready-to-use HTTP handler bundle.
func New(svc *service.Service, securityEnabled bool) *Handler {
	svc.SetSecurityEnabled(securityEnabled)
	return &Handler{svc: svc}
}

// securityEnabled reports the current lab mode through the underlying service.
//
// Output:
//   - bool: true when the secure code paths are active.
func (h *Handler) securityEnabled() bool {
	return h.svc.SecurityEnabled()
}

// SetSecurityEnabled updates the active lab mode for subsequent requests.
//
// Input:
//   - enabled: new security state to store.
func (h *Handler) SetSecurityEnabled(enabled bool) {
	h.svc.SetSecurityEnabled(enabled)
}

// renderHTML writes a templ component to the response writer and falls back to
// a plain-text 500 response when rendering fails.
//
// Inputs:
//   - c: Gin context that owns the HTTP response.
//   - status: HTTP status code to send before rendering.
//   - pageName: short label used only for logging.
//   - component: templ component to render.
func renderHTML(c *gin.Context, status int, pageName string, component templ.Component) {
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		log.Printf("render %s failed: %v", pageName, err)
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusInternalServerError, "Failed to render page")
	}
}

// signCookieValue creates a signed cookie value: "username|hmac(username)".
// Used in secure mode so the cookie value cannot be trivially forged.
func signCookieValue(username string) string {
	mac := hmac.New(sha256.New, []byte(sessionSecret()))
	mac.Write([]byte(username))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return username + "|" + sig
}

// sessionSecret returns the secret used to sign secure-mode cookies.
//
// Output:
//   - string: configured secret or the built-in demo fallback.
func sessionSecret() string {
	if secret := os.Getenv(sessionSecretEnv); secret != "" {
		return secret
	}
	return fallbackSessionSecret
}

// verifyCookieValue checks the HMAC signature and returns the username.
func verifyCookieValue(value string) (string, bool) {
	idx := strings.LastIndex(value, "|")
	if idx < 0 {
		return "", false
	}
	username := value[:idx]
	expected := signCookieValue(username)
	if !hmac.Equal([]byte(value), []byte(expected)) {
		return "", false
	}
	return username, true
}

// setAuthCookie stores the login cookie using either plaintext or signed mode.
//
// Inputs:
//   - c: Gin context that owns the response.
//   - username: authenticated username to store in the cookie.
func (h *Handler) setAuthCookie(c *gin.Context, username string) {
	cookieVal := username
	if h.securityEnabled() {
		// Secure mode: HMAC-signed cookie + SameSite=Strict.
		cookieVal = signCookieValue(username)
		c.SetSameSite(http.SameSiteStrictMode)
	}
	// HttpOnly=true in both modes; Secure=false since the lab runs on plain HTTP.
	c.SetCookie(authCookieName, cookieVal, 60*60*8, "/", "", false, true)
}

// currentUsername resolves the logged-in username from the auth cookie.
//
// Inputs:
//   - c: Gin context that may contain the auth cookie.
//
// Outputs:
//   - string: authenticated username when present and valid.
//   - bool: true when a usable login cookie was found.
func (h *Handler) currentUsername(c *gin.Context) (string, bool) {
	raw, err := c.Cookie(authCookieName)
	if err != nil || raw == "" {
		return "", false
	}

	var username string
	if h.securityEnabled() {
		var ok bool
		username, ok = verifyCookieValue(raw)
		if !ok {
			return "", false
		}
	} else {
		// VULNERABLE: plaintext cookie value — easily forged by editing the cookie.
		username = raw
	}

	exists, err := h.svc.UserExists(username)
	if err != nil || !exists {
		return "", false
	}

	return username, true
}

// requireLoginJSON returns false and sends a 401 JSON response when the
// request is not authenticated.
//
// Input:
//   - c: Gin context that may already carry authentication state.
//
// Output:
//   - bool: true when the request is authenticated.
func (h *Handler) requireLoginJSON(c *gin.Context) bool {
	if _, ok := h.currentUsername(c); ok {
		return true
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Login required"})
	return false
}

// requireLoginUI redirects unauthenticated requests to the supplied fallback
// URL and returns false.
//
// Inputs:
//   - c: Gin context that may already carry authentication state.
//   - fallback: redirect target used when the request is not logged in.
//
// Output:
//   - bool: true when the request is authenticated.
func (h *Handler) requireLoginUI(c *gin.Context, fallback string) bool {
	if _, ok := h.currentUsername(c); ok {
		return true
	}

	if fallback == "" {
		fallback = "/ui/login?err=1&msg=Please+log+in+to+continue"
	}
	c.Redirect(http.StatusSeeOther, fallback)
	return false
}

// GetPosts returns all published blog posts as JSON.
func (h *Handler) GetPosts() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, err := h.svc.GetPublishedPosts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
			return
		}

		c.JSON(http.StatusOK, posts)
	}
}

// PostLogin accepts JSON credentials, applies the current security mode, and
// returns a login result as JSON.
func (h *Handler) PostLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		message, isError, status := h.evaluateLogin(req.Username, req.Password)
		if isError {
			c.JSON(status, gin.H{"error": message})
			return
		}

		h.setAuthCookie(c, req.Username)
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Login successful for %s", req.Username)})
	}
}

// PostRegister accepts JSON registration data and creates a new user record.
func (h *Handler) PostRegister() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Email    string `json:"email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if req.Username == "" || req.Password == "" || req.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username, password and email are required"})
			return
		}

		if err := h.svc.CreateUser(req.Username, req.Password, req.Email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to register user"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "User registered"})
	}
}

// PostCreate accepts JSON post data and persists a new blog entry.
func (h *Handler) PostCreate() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Title       string `json:"title"`
			PostContent string `json:"post_content"`
			Published   int    `json:"published"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if !h.requireLoginJSON(c) {
			return
		}

		if req.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
			return
		}

		author := ""
		if username, ok := h.currentUsername(c); ok {
			author = username
		}

		id, err := h.svc.CreatePost(req.Title, req.PostContent, req.Published, author, "", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Post created"})
	}
}

// PostUpdate accepts JSON post data and updates an existing entry.
func (h *Handler) PostUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post id"})
			return
		}

		var req struct {
			Title       string `json:"title"`
			PostContent string `json:"post_content"`
			Published   int    `json:"published"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if req.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
			return
		}

		if !h.requireLoginJSON(c) {
			return
		}

		if err := h.svc.UpdatePost(id, req.Title, req.PostContent, req.Published, "", ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Post updated"})
	}
}

// PostDelete deletes a post after the current mode-specific authorization check.
func (h *Handler) PostDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireLoginJSON(c) {
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post id"})
			return
		}

		if h.securityEnabled() {
			username, ok := h.currentUsername(c)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Login required"})
				return
			}

			allowed, authErr := h.canDeletePost(username, id)
			if authErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
				return
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own posts"})
				return
			}
		}

		if err := h.svc.DeletePost(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete post"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Post deleted"})
	}
}

// PagePosts renders the full blog list page with the current login state.
func (h *Handler) PagePosts() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, err := h.svc.GetAllPosts()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to render posts page")
			return
		}

		// Flash message comes only from query params (e.g. after redirect).
		// The "please log in" hint is rendered by the template itself in the
		// !loggedIn branch — duplicating it here produced two stacked banners.
		message := c.Query("msg")
		isError := c.Query("err") == "1"
		username, loggedIn := h.currentUsername(c)
		component := views.PostsPage(posts, h.securityEnabled(), loggedIn, username, message, isError)
		renderHTML(c, http.StatusOK, "posts", component)
	}
}

// PageLogin renders the login page.
func (h *Handler) PageLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		component := views.LoginPage(h.securityEnabled(), loggedIn, username, "", false)
		renderHTML(c, http.StatusOK, "login", component)
	}
}

// PageRegister renders the registration page.
func (h *Handler) PageRegister() gin.HandlerFunc {
	return func(c *gin.Context) {
		message := c.Query("msg")
		isError := c.Query("err") == "1"
		username, loggedIn := h.currentUsername(c)
		component := views.RegisterPage(h.securityEnabled(), loggedIn, username, message, isError)
		renderHTML(c, http.StatusOK, "register", component)
	}
}

// PageRegisterSubmit handles the classic form-based registration flow.
func (h *Handler) PageRegisterSubmit() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		email := c.PostForm("email")

		if username == "" || password == "" || email == "" {
			c.Redirect(http.StatusSeeOther, "/ui/register?err=1&msg=All+fields+are+required")
			return
		}

		if err := h.svc.CreateUser(username, password, email); err != nil {
			c.Redirect(http.StatusSeeOther, "/ui/register?err=1&msg=Failed+to+register+user")
			return
		}

		c.Redirect(http.StatusSeeOther, "/ui/register?msg=User+registered")
	}
}

// PageRegisterPartial is the HTMX endpoint for in-place registration.
// On success it instructs the browser (via HX-Redirect) to navigate to /ui/login,
// so the message banner is shown there and the user can log in immediately.
func (h *Handler) PageRegisterPartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		email := c.PostForm("email")

		if username == "" || password == "" || email == "" {
			component := views.ResultMessage("All fields are required", true)
			renderHTML(c, http.StatusBadRequest, "register_partial", component)
			return
		}

		if err := h.svc.CreateUser(username, password, email); err != nil {
			component := views.ResultMessage("Failed to register user (username or email may already exist)", true)
			renderHTML(c, http.StatusBadRequest, "register_partial", component)
			return
		}

		// HX-Redirect causes htmx to do a full client-side navigation,
		// which reloads the layout and refreshes the auth-aware navbar.
		c.Header("HX-Redirect", "/ui/login?msg=Account+created.+You+can+log+in+now")
		c.Status(http.StatusOK)
	}
}

// PagePostsPartial returns the posts list fragment used by HTMX updates.
func (h *Handler) PagePostsPartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, err := h.svc.GetAllPosts()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to render posts partial")
			return
		}

		username, loggedIn := h.currentUsername(c)
		component := views.PostsListContainer(posts, loggedIn, username, h.securityEnabled())
		renderHTML(c, http.StatusOK, "posts_partial", component)
	}
}

// readPublishedFromForm normalizes common checkbox and select values to 0 or 1.
func readPublishedFromForm(c *gin.Context) int {
	v := c.PostForm("published")
	if v == "1" || v == "on" || v == "true" {
		return 1
	}
	return 0
}

// saveUploadedAttachment stores an uploaded file under uploadsDir and returns
// its public path and original filename. Returns ("", "", nil) when no file
// was provided.
func (h *Handler) saveUploadedAttachment(c *gin.Context, formField string) (publicPath, originalName string, err error) {
	file, err := c.FormFile(formField)
	if err != nil {
		// No file uploaded – not an error condition.
		return "", "", nil
	}

	if file.Size > maxUploadBytes {
		return "", "", fmt.Errorf("file too large (max %d bytes)", maxUploadBytes)
	}

	cleanName := sanitizeFilename(file.Filename)
	if cleanName == "" {
		return "", "", fmt.Errorf("invalid filename")
	}

	storedName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), cleanName)
	dst := filepath.Join(uploadsDir, storedName)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		return "", "", err
	}

	return uploadsURLPath + "/" + storedName, file.Filename, nil
}

// sanitizeFilename strips path separators and disallows traversal sequences.
// Returns "" for invalid names.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	if name == "" || name == "." || name == ".." {
		return ""
	}
	if strings.ContainsAny(name, `/\`) {
		return ""
	}
	return name
}

// PagePostsCreate handles the classic form-based create-post flow.
func (h *Handler) PagePostsCreate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireLoginUI(c, "/ui/login?err=1&msg=Please+log+in+to+add+posts") {
			return
		}

		title := c.PostForm("title")
		content := c.PostForm("post_content")
		published := readPublishedFromForm(c)

		if title == "" {
			c.Redirect(http.StatusSeeOther, "/ui/posts?err=1&msg=Title+is+required")
			return
		}

		author := ""
		if username, ok := h.currentUsername(c); ok {
			author = username
		}

		attachmentPath, attachmentName, err := h.saveUploadedAttachment(c, "attachment")
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/ui/posts?err=1&msg=Failed+to+save+attachment")
			return
		}

		if _, err := h.svc.CreatePost(title, content, published, author, attachmentPath, attachmentName); err != nil {
			c.Redirect(http.StatusSeeOther, "/ui/posts?err=1&msg=Failed+to+create+post")
			return
		}

		c.Redirect(http.StatusSeeOther, "/ui/posts?msg=Post+created")
	}
}

// PagePostsCreatePartial handles HTMX form submissions and returns the freshly
// rendered posts list so the page updates without a full reload.
// PagePostsCreatePartial handles HTMX post creation and re-renders the list.
func (h *Handler) PagePostsCreatePartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		if !loggedIn {
			c.Header("HX-Redirect", "/ui/login?err=1&msg=Please+log+in+to+add+posts")
			c.Status(http.StatusUnauthorized)
			return
		}

		title := c.PostForm("title")
		content := c.PostForm("post_content")
		published := readPublishedFromForm(c)

		if title == "" {
			posts, _ := h.svc.GetAllPosts()
			component := views.PostsListWithBanner(posts, loggedIn, username, h.securityEnabled(), "Title is required", true)
			renderHTML(c, http.StatusBadRequest, "posts_create_partial", component)
			return
		}

		attachmentPath, attachmentName, err := h.saveUploadedAttachment(c, "attachment")
		if err != nil {
			posts, _ := h.svc.GetAllPosts()
			component := views.PostsListWithBanner(posts, loggedIn, username, h.securityEnabled(), "Failed to save attachment: "+err.Error(), true)
			renderHTML(c, http.StatusBadRequest, "posts_create_partial", component)
			return
		}

		if _, err := h.svc.CreatePost(title, content, published, username, attachmentPath, attachmentName); err != nil {
			posts, _ := h.svc.GetAllPosts()
			component := views.PostsListWithBanner(posts, loggedIn, username, h.securityEnabled(), "Failed to create post", true)
			renderHTML(c, http.StatusInternalServerError, "posts_create_partial", component)
			return
		}

		posts, _ := h.svc.GetAllPosts()
		component := views.PostsListWithBanner(posts, loggedIn, username, h.securityEnabled(), "Post created", false)
		c.Header("HX-Trigger", "post-created")
		renderHTML(c, http.StatusOK, "posts_create_partial", component)
	}
}

// PagePostEdit renders the edit form for a single post.
func (h *Handler) PagePostEdit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireLoginUI(c, "/ui/login?err=1&msg=Please+log+in+to+edit+posts") {
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid post id")
			return
		}

		post, err := h.svc.GetPostByID(id)
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Post not found")
			return
		}
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load post")
			return
		}

		message := c.Query("msg")
		isError := c.Query("err") == "1"
		username, loggedIn := h.currentUsername(c)
		component := views.EditPostPage(h.securityEnabled(), loggedIn, username, post, message, isError)
		renderHTML(c, http.StatusOK, "post_edit", component)
	}
}

// PagePostEditSubmit processes the edit form and persists the updated post.
func (h *Handler) PagePostEditSubmit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireLoginUI(c, "/ui/login?err=1&msg=Please+log+in+to+edit+posts") {
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid post id")
			return
		}

		title := c.PostForm("title")
		content := c.PostForm("post_content")
		published := readPublishedFromForm(c)

		if title == "" {
			c.Redirect(http.StatusSeeOther, fmt.Sprintf("/ui/posts/edit/%d?err=1&msg=Title+is+required", id))
			return
		}

		attachmentPath, attachmentName, err := h.saveUploadedAttachment(c, "attachment")
		if err != nil {
			c.Redirect(http.StatusSeeOther, fmt.Sprintf("/ui/posts/edit/%d?err=1&msg=Failed+to+save+attachment", id))
			return
		}

		if err := h.svc.UpdatePost(id, title, content, published, attachmentPath, attachmentName); err != nil {
			c.Redirect(http.StatusSeeOther, fmt.Sprintf("/ui/posts/edit/%d?err=1&msg=Failed+to+update+post", id))
			return
		}

		c.Redirect(http.StatusSeeOther, "/ui/posts?msg=Post+updated")
	}
}

// PagePostDelete handles the browser form used to delete a post.
func (h *Handler) PagePostDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireLoginUI(c, "/ui/login?err=1&msg=Please+log+in+to+delete+posts") {
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid post id")
			return
		}

		if h.securityEnabled() {
			username, ok := h.currentUsername(c)
			if !ok {
				c.Redirect(http.StatusSeeOther, "/ui/login?err=1&msg=Please+log+in+to+delete+posts")
				return
			}

			allowed, authErr := h.canDeletePost(username, id)
			if authErr != nil {
				c.Redirect(http.StatusSeeOther, "/ui/posts?err=1&msg=Failed+to+authorize+delete")
				return
			}
			if !allowed {
				c.Redirect(http.StatusSeeOther, "/ui/posts?err=1&msg=You+can+delete+only+your+own+posts")
				return
			}
		}

		if err := h.svc.DeletePost(id); err != nil {
			c.Redirect(http.StatusSeeOther, "/ui/posts?err=1&msg=Failed+to+delete+post")
			return
		}

		c.Redirect(http.StatusSeeOther, "/ui/posts?msg=Post+deleted")
	}
}

// evaluateLogin validates the login request and returns the message, error
// state, and HTTP status that the caller should send back.
//
// Inputs:
//   - username: submitted username.
//   - password: submitted password.
//
// Outputs:
//   - message: human-readable result for the UI or API.
//   - isError: true when the login failed.
//   - status: HTTP status code that matches the result.
func (h *Handler) evaluateLogin(username, password string) (message string, isError bool, status int) {
	if username == "" || password == "" {
		return "Username and password are required", true, http.StatusBadRequest
	}

	// Secure mode: enforce rate limiting to block brute-force attacks.
	if h.securityEnabled() && !h.svc.CheckRateLimit(username) {
		return "Too many failed attempts. Please wait 60 seconds.", true, http.StatusTooManyRequests
	}

	if h.securityEnabled() {
		valid, err := h.svc.ValidateUserCredentials(username, password)
		if err != nil {
			return "Database error", true, http.StatusInternalServerError
		}
		if !valid {
			h.svc.RecordLoginFailure(username)
			return "Invalid username or password", true, http.StatusUnauthorized
		}
		return fmt.Sprintf("Login successful for %s", username), false, http.StatusOK
	}

	// VULNERABLE (insecure mode): Broken Authentication — only check user
	// existence; password is ignored. This preserves the lab demonstration of
	// the vulnerability.
	exists, err := h.svc.UserExists(username)
	if err != nil {
		return "Database error", true, http.StatusInternalServerError
	}
	if !exists {
		return "User not found", true, http.StatusUnauthorized
	}

	return fmt.Sprintf("Login successful for %s", username), false, http.StatusOK
}

// Logout clears the auth cookie and redirects to posts UI.
// Logout clears the auth cookie and redirects to the posts page.
func (h *Handler) Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie(authCookieName, "", -1, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/ui/posts")
	}
}

// canDeletePost reports whether the current user may delete the selected post.
//
// Inputs:
//   - username: authenticated username.
//   - postID: identifier of the post to delete.
//
// Outputs:
//   - bool: true when deletion is allowed.
//   - error: lookup failure while checking role or ownership.
func (h *Handler) canDeletePost(username string, postID int) (bool, error) {
	isAdmin, err := h.svc.IsUserAdmin(username)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	author, err := h.svc.GetPostAuthor(postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return author == username, nil
}

// PageLoginSubmit processes the browser login form and renders the result page.
func (h *Handler) PageLoginSubmit() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		message, isError, status := h.evaluateLogin(username, password)
		if !isError {
			h.setAuthCookie(c, username)
		}

		loggedIn := !isError
		component := views.LoginPage(h.securityEnabled(), loggedIn, username, message, isError)
		renderHTML(c, status, "login_submit", component)
	}
}

// PageLoginPartial is the HTMX endpoint. On success we send HX-Redirect so the
// whole layout refreshes (showing the green username badge in the navbar).
func (h *Handler) PageLoginPartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		message, isError, status := h.evaluateLogin(username, password)
		if !isError {
			h.setAuthCookie(c, username)
			c.Header("HX-Redirect", "/ui/posts?msg=Welcome+back%2C+"+username)
			c.Status(http.StatusOK)
			return
		}

		component := views.LoginResult(message, isError)
		renderHTML(c, status, "login_partial", component)
	}
}

// ============================================================================
// VULNERABLE ENDPOINTS FOR DEMONSTRATION (Disabled when SECURITY_ENABLED=true)
// ============================================================================

// Search runs a blog search through the SECURITY_ENABLED toggle:
//   - secure mode → parameterized LIKE (input cannot break out)
//   - insecure mode → direct string concatenation (SQL Injection)
//
// Returns posts as JSON.
func (h *Handler) Search() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' required"})
			return
		}

		posts, err := h.svc.SearchPosts(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed", "detail": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"query":   query,
			"mode":    modeLabel(h.securityEnabled()),
			"results": posts,
		})
	}
}

// SearchVulnerable is a force-vulnerable SQL Injection endpoint that always
// uses string concatenation, regardless of SECURITY_ENABLED. Kept for the
// side-by-side defense demo.
//
// Try: /api/search-vulnerable?q=zz' ORDER BY 8 --
// Or:  /api/search-vulnerable?q=zz' OR EXISTS(SELECT 1 FROM users WHERE username='admin' AND substr(password_hash,1,1)='a') --
// Or:  /api/search-vulnerable?q=zz' UNION SELECT 1, name, sql, 1, char(32), char(32), char(32) FROM sqlite_master WHERE type='table' --
// Or:  /api/search-vulnerable?q=zz' UNION SELECT 9000, '[intel] database map', 'users=' || (SELECT COUNT(*) FROM users) || ' tables=' || (SELECT group_concat(name, ', ') FROM sqlite_master WHERE type='table'), 1, 'sqli-bot', char(32), char(32) --
func (h *Handler) SearchVulnerable() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' required"})
			return
		}

		posts, err := h.svc.SearchPostsVulnerable(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed", "detail": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"query":   query,
			"mode":    "vulnerable (forced)",
			"results": posts,
		})
	}
}

// PageSearch renders the full search page with the form.
func (h *Handler) PageSearch() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		query := c.Query("q")
		var posts []service.Post
		var err error
		if query != "" {
			posts, err = h.svc.SearchPosts(query)
			if err != nil {
				component := views.SearchPage(h.securityEnabled(), loggedIn, username, query, nil, "Search failed: "+err.Error(), true)
				renderHTML(c, http.StatusOK, "search", component)
				return
			}
		}
		component := views.SearchPage(h.securityEnabled(), loggedIn, username, query, posts, "", false)
		renderHTML(c, http.StatusOK, "search", component)
	}
}

// PageSearchPartial returns just the results fragment for HTMX.
func (h *Handler) PageSearchPartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.PostForm("q")
		if query == "" {
			query = c.Query("q")
		}
		if query == "" {
			component := views.SearchResults("", nil, "Type something to search", true, h.securityEnabled())
			renderHTML(c, http.StatusOK, "search_partial", component)
			return
		}

		posts, err := h.svc.SearchPosts(query)
		if err != nil {
			component := views.SearchResults(query, nil, "Search failed: "+err.Error(), true, h.securityEnabled())
			renderHTML(c, http.StatusInternalServerError, "search_partial", component)
			return
		}
		component := views.SearchResults(query, posts, "", false, h.securityEnabled())
		renderHTML(c, http.StatusOK, "search_partial", component)
	}
}

// modeLabel converts the security flag into a short human-readable label.
func modeLabel(secure bool) string {
	if secure {
		return "secure (parameterized)"
	}
	return "vulnerable (concatenated)"
}

// PageSecurityMap renders the lab overview and current security posture.
func (h *Handler) PageSecurityMap() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		component := views.SecurityMapPage(h.securityEnabled(), loggedIn, username)
		renderHTML(c, http.StatusOK, "security_map", component)
	}
}

// PageVulnDemos renders the security demo index page.
func (h *Handler) PageVulnDemos() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		demos := []views.VulnDemo{
			{
				Emoji:       "💉",
				Title:       "SQL Injection",
				CWE:         "CWE-89",
				OWASP:       "A03:2021",
				Status:      "ready",
				Description: "The Vocaloid library search concatenates the search term into a SELECT in vulnerable mode; the payload builds a live DB intelligence row.",
				Href:        "/ui/library",
				Payload:     `zz' UNION SELECT 9000, '[intel] database map', 'users=' || (SELECT COUNT(*) FROM users) || ' tables=' || (SELECT group_concat(name, ', ') FROM sqlite_master WHERE type='table'), 1, 'sqli-bot', '', '' --`,
			},
			{
				Emoji:       "🪲",
				Title:       "Stored XSS",
				CWE:         "CWE-79",
				OWASP:       "A03:2021",
				Status:      "ready",
				Description: "Fan comments are rendered as raw HTML in vulnerable mode, so a malicious comment can take over the visible page UI for every reader.",
				Href:        "/ui/posts/view/1",
				Payload:     `<img src=x onerror="document.body.insertAdjacentHTML('afterbegin','<div style=&quot;position:fixed;inset:0;z-index:9999;background:#111827;color:white;padding:40px&quot;><h1>Session expired</h1><p>Fake re-login overlay injected from a comment.</p></div>')">`,
			},
			{
				Emoji:       "🔐",
				Title:       "Broken Authentication",
				CWE:         "CWE-287",
				OWASP:       "A07:2021",
				Status:      "ready",
				Description: "The normal login accepts any password for an existing member in vulnerable mode; that can be chained with moderation actions.",
				Href:        "/ui/login",
				Payload:     `username=user1&password=totally-wrong → POST /ui/posts/delete/{adminPostId}`,
			},
			{
				Emoji:       "🔓",
				Title:       "Broken Access Control",
				CWE:         "CWE-639",
				OWASP:       "A01:2021",
				Status:      "ready",
				Description: "The moderation queue exposes post delete actions. Vulnerable mode checks only login, while secure mode checks owner/admin rights.",
				Href:        "/ui/moderation",
				Payload:     `POST /ui/posts/delete/{otherUsersPostId}`,
			},
			{
				Emoji:       "🗝️",
				Title:       "Sensitive Data Exposure",
				CWE:         "CWE-200",
				OWASP:       "A02:2021",
				Status:      "ready",
				Description: "The member directory shows the persistence layer: plaintext credentials in vulnerable mode, bcrypt hashes in secure mode.",
				Href:        "/ui/members",
				Payload:     `sqlite> SELECT username, password_hash FROM users;`,
			},
			{
				Emoji:       "🪪",
				Title:       "CSRF",
				CWE:         "CWE-352",
				OWASP:       "A01:2021",
				Status:      "ready",
				Description: "Profile settings update the notification email. Without an anti-CSRF token, another page can auto-submit the state-changing POST.",
				Href:        "/ui/profile",
				Payload:     `<form id=x action="/ui/profile" method="POST"><input name="new_email" value="attacker+csrf@evil.test"></form><script>x.submit()</script>`,
			},
			{
				Emoji:       "📂",
				Title:       "Path Traversal / LFI",
				CWE:         "CWE-22",
				OWASP:       "A01:2021",
				Status:      "ready",
				Description: "The fanart vault previews uploaded files. Vulnerable mode trusts the filename, so ../ can expose source code or the SQLite database.",
				Href:        "/ui/gallery",
				Payload:     `GET /api/files-vulnerable?name=../internal/db/db.go`,
			},
			{
				Emoji:       "💻",
				Title:       "Command Injection",
				CWE:         "CWE-78",
				OWASP:       "A03:2021",
				Status:      "ready",
				Description: "The stream health check pings relay hosts. Vulnerable mode passes the host through sh -c, so a relay check becomes shell execution.",
				Href:        "/ui/stream-check",
				Payload:     `GET /api/ping-vulnerable?host=127.0.0.1; whoami; uname -a`,
			},
			{
				Emoji:       "🛠️",
				Title:       "Security Misconfiguration",
				CWE:         "CWE-16",
				OWASP:       "A05:2021",
				Status:      "ready",
				Description: "Vulnerable mode: Gin default Recovery leaks stack trace on /debug/crash; no security headers. Secure mode: CSP/HSTS/X-Frame-Options + sanitized 500.",
				Href:        "/debug/crash",
				Payload:     `curl -i http://localhost:8080/debug/crash # vuln: full stack trace; secure: {"error":"Internal server error"}`,
			},
		}
		component := views.VulnDemosPage(h.securityEnabled(), loggedIn, username, demos)
		renderHTML(c, http.StatusOK, "vuln_demos", component)
	}
}

// CommentsVulnerable stores comments without sanitization for the XSS demo.
func (h *Handler) CommentsVulnerable() gin.HandlerFunc {
	return func(c *gin.Context) {
		postID, _ := strconv.Atoi(c.PostForm("post_id"))
		body := c.PostForm("body")
		author := c.PostForm("author")

		if postID == 0 || body == "" {
			var req struct {
				PostID int    `json:"post_id"`
				Body   string `json:"body"`
				Author string `json:"author"`
			}
			if err := c.ShouldBindJSON(&req); err == nil {
				postID = req.PostID
				body = req.Body
				author = req.Author
			}
		}

		if postID == 0 || body == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "post_id and body required"})
			return
		}

		if _, err := h.svc.CreateCommentVulnerable(postID, author, body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store comment"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"message": "Comment stored verbatim (force-vulnerable)",
			"post_id": postID,
			"body":    body,
		})
	}
}

// DebugCrash deliberately panics so the Security Misconfiguration demo can
// show how Gin's default Recovery leaks the stack trace in vulnerable mode and
// how ErrorSanitizerMiddleware swallows it in secure mode.
func (h *Handler) DebugCrash() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Pretend the user passed bad input that we did not validate. Triggers
		// a runtime panic with a colourful stack trace.
		var posts []string
		_ = posts[42] // index out of range
		c.JSON(http.StatusOK, gin.H{"unreachable": true})
	}
}

// CsrfFormVulnerable returns and accepts a form without CSRF protection.
// The POST handler actually updates the logged-in user's email address,
// making the state-change exploitable via a cross-origin request.
func (h *Handler) CsrfFormVulnerable() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		if c.Request.Method == http.MethodGet {
			email := ""
			if loggedIn {
				email, _ = h.svc.GetUserEmail(username)
			}
			component := views.CSRFDemoPage(h.securityEnabled(), loggedIn, username, "", email, "", false)
			renderHTML(c, http.StatusOK, "csrf_demo", component)
			return
		}

		// POST — vulnerable: no CSRF token check.
		newEmail := c.PostForm("new_email")
		if !loggedIn {
			c.Redirect(http.StatusSeeOther, "/ui/login?err=1&msg=Please+log+in+to+update+email")
			return
		}
		if newEmail == "" {
			email, _ := h.svc.GetUserEmail(username)
			component := views.CSRFDemoPage(h.securityEnabled(), loggedIn, username, "", email, "Email cannot be empty", true)
			renderHTML(c, http.StatusBadRequest, "csrf_demo", component)
			return
		}
		if err := h.svc.UpdateUserEmail(username, newEmail); err != nil {
			component := views.CSRFDemoPage(h.securityEnabled(), loggedIn, username, "", newEmail, "Failed to update email", true)
			renderHTML(c, http.StatusInternalServerError, "csrf_demo", component)
			return
		}
		component := views.CSRFDemoPage(h.securityEnabled(), loggedIn, username, "", newEmail,
			fmt.Sprintf("Email updated to %s (no CSRF token was checked!)", newEmail), false)
		renderHTML(c, http.StatusOK, "csrf_demo", component)
	}
}

// ProfileSettings dispatches to the secure or vulnerable profile workflow.
func (h *Handler) ProfileSettings() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.securityEnabled() {
			h.CsrfSecureForm()(c)
			return
		}
		h.CsrfFormVulnerable()(c)
	}
}

// ============================================================================
// NEW HANDLERS — XSS, CSRF secure, IDOR, DB Expose, Path Traversal
// ============================================================================

// CommentsSecure stores a comment after HTML-escaping the body.
func (h *Handler) CommentsSecure() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			PostID  int    `json:"post_id"`
			Comment string `json:"comment"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		if req.Comment == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "comment cannot be empty"})
			return
		}

		author := "anonymous"
		if u, ok := h.currentUsername(c); ok {
			author = u
		}

		// Secure: CreateComment HTML-escapes the body before storage.
		id, err := h.svc.CreateComment(req.PostID, author, req.Comment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store comment"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Comment stored (secure — body HTML-escaped)",
			"id":      id,
			"post_id": req.PostID,
		})
	}
}

// PagePostDetail renders a post together with its comment thread.
func (h *Handler) PagePostDetail() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid post id")
			return
		}

		post, err := h.svc.GetPostByID(id)
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Post not found")
			return
		}
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load post")
			return
		}

		comments, err := h.svc.GetCommentsByPostID(id)
		if err != nil {
			comments = []service.Comment{}
		}

		message := c.Query("msg")
		isError := c.Query("err") == "1"
		username, loggedIn := h.currentUsername(c)
		component := views.PostDetailPage(post, comments, h.securityEnabled(), loggedIn, username, message, isError)
		renderHTML(c, http.StatusOK, "post_detail", component)
	}
}

// PagePostCommentSubmit processes a form-based comment submission.
func (h *Handler) PagePostCommentSubmit() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid post id")
			return
		}

		body := c.PostForm("body")
		if body == "" {
			c.Redirect(http.StatusSeeOther, fmt.Sprintf("/ui/posts/%d?err=1&msg=Comment+cannot+be+empty", id))
			return
		}

		author := "anonymous"
		if u, ok := h.currentUsername(c); ok {
			author = u
		}

		if _, err := h.svc.CreateComment(id, author, body); err != nil {
			c.Redirect(http.StatusSeeOther, fmt.Sprintf("/ui/posts/%d?err=1&msg=Failed+to+save+comment", id))
			return
		}

		comments, err := h.svc.GetCommentsByPostID(id)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load comments")
			return
		}

		component := views.CommentsList(comments, h.securityEnabled())
		renderHTML(c, http.StatusOK, "comments_list", component)
	}
}

// generateCSRFToken creates a cryptographically random base64url token.
func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp — not cryptographically safe, but ensures
		// the handler does not crash in the unlikely event rand.Read fails.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// setCSRFCookie stores the CSRF token in a SameSite=Strict cookie.
//
// Inputs:
//   - c: Gin context that owns the response.
//   - token: CSRF token to store.
func setCSRFCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(csrfCookieName, token, 60*60, "/", "", false, true)
}

// CsrfSecureForm handles the CSRF-protected email-update form.
// GET issues a new token; POST validates the token before updating the email.
func (h *Handler) CsrfSecureForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)

		if c.Request.Method == http.MethodGet {
			token := generateCSRFToken()
			setCSRFCookie(c, token)
			email := ""
			if loggedIn {
				email, _ = h.svc.GetUserEmail(username)
			}
			component := views.CSRFDemoPage(h.securityEnabled(), loggedIn, username, token, email, "", false)
			renderHTML(c, http.StatusOK, "csrf_secure", component)
			return
		}

		// POST — secure: validate CSRF token.
		formToken := c.PostForm("csrf_token")
		cookieToken, cookieErr := c.Cookie(csrfCookieName)
		if cookieErr != nil || formToken == "" || formToken != cookieToken {
			token := generateCSRFToken()
			setCSRFCookie(c, token)
			email := ""
			if loggedIn {
				email, _ = h.svc.GetUserEmail(username)
			}
			component := views.CSRFDemoPage(h.securityEnabled(), loggedIn, username, token, email, "CSRF token validation failed", true)
			renderHTML(c, http.StatusForbidden, "csrf_secure", component)
			return
		}

		if !loggedIn {
			c.Redirect(http.StatusSeeOther, "/ui/login?err=1&msg=Please+log+in")
			return
		}

		newEmail := c.PostForm("new_email")
		if newEmail == "" {
			email, _ := h.svc.GetUserEmail(username)
			token := generateCSRFToken()
			setCSRFCookie(c, token)
			component := views.CSRFDemoPage(h.securityEnabled(), loggedIn, username, token, email, "Email cannot be empty", true)
			renderHTML(c, http.StatusBadRequest, "csrf_secure", component)
			return
		}

		if err := h.svc.UpdateUserEmail(username, newEmail); err != nil {
			token := generateCSRFToken()
			setCSRFCookie(c, token)
			component := views.CSRFDemoPage(h.securityEnabled(), loggedIn, username, token, newEmail, "Failed to update email", true)
			renderHTML(c, http.StatusInternalServerError, "csrf_secure", component)
			return
		}

		token := generateCSRFToken()
		setCSRFCookie(c, token)
		component := views.CSRFDemoPage(h.securityEnabled(), loggedIn, username, token, newEmail,
			fmt.Sprintf("Email updated to %s (CSRF token was valid ✓)", newEmail), false)
		renderHTML(c, http.StatusOK, "csrf_secure", component)
	}
}

// PageIDOR renders the broken-access-control demo page.
func (h *Handler) PageIDOR() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, _ := h.svc.GetAllPosts()
		username, loggedIn := h.currentUsername(c)
		message := c.Query("msg")
		isError := c.Query("err") == "1"
		component := views.IDORDemoPage(posts, h.securityEnabled(), loggedIn, username, message, isError)
		renderHTML(c, http.StatusOK, "idor_demo", component)
	}
}

// PageDBExpose renders the sensitive-data-exposure demo page.
func (h *Handler) PageDBExpose() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		users, err := h.svc.GetAllUsers()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load users")
			return
		}
		component := views.DBExposePage(users, h.securityEnabled(), loggedIn, username)
		renderHTML(c, http.StatusOK, "db_expose", component)
	}
}

// FilesVulnerable reads a file directly from user-supplied input.
func (h *Handler) FilesVulnerable() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name parameter required"})
			return
		}

		// VULNERABLE: no path validation — attacker can read any file the
		// server process has access to via relative sequences like "../../etc/passwd".
		path := uploadsDir + "/" + name
		data, err := os.ReadFile(path)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found", "path": path})
			return
		}

		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, string(data))
	}
}

// FilesSecure reads a file only after validating that it remains in uploads.
func (h *Handler) FilesSecure() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name parameter required"})
			return
		}

		candidate, ok := safeUploadPath(name)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "path traversal detected — access denied",
				"detail": "resolved path escapes the uploads directory",
			})
			return
		}

		data, err := os.ReadFile(candidate)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}

		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, string(data))
	}
}

// safeUploadPath resolves a filename and rejects traversal outside uploads.
//
// Input:
//   - name: user-supplied file name or relative path.
//
// Outputs:
//   - string: absolute path inside uploads when valid.
//   - bool: true when the resolved path is safe.
func safeUploadPath(name string) (string, bool) {
	if filepath.IsAbs(name) {
		return "", false
	}

	cleaned := filepath.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", false
	}

	base, err := filepath.Abs(uploadsDir)
	if err != nil {
		return "", false
	}
	candidate := filepath.Join(base, cleaned)
	if !strings.HasPrefix(candidate, base+string(filepath.Separator)) && candidate != base {
		return "", false
	}

	return candidate, true
}

// PagePathTraversal renders the path traversal demo page.
func (h *Handler) PagePathTraversal() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		filename := c.Query("name")
		var content, message string
		isError := false

		if filename != "" {
			endpoint := "/api/files-vulnerable"
			if h.securityEnabled() {
				endpoint = "/api/files-secure"
			}
			_ = endpoint // displayed in view; actual read below

			// Read via the appropriate path validation strategy.
			if !h.securityEnabled() {
				path := uploadsDir + "/" + filename
				data, err := os.ReadFile(path)
				if err != nil {
					message = fmt.Sprintf("Error: %v", err)
					isError = true
				} else {
					content = string(data)
					if len(content) > 4096 {
						content = content[:4096] + "\n... (truncated)"
					}
				}
			} else {
				candidate, ok := safeUploadPath(filename)
				if !ok {
					message = "⛔ Path traversal blocked: resolved path escapes the uploads directory"
					isError = true
				} else {
					data, err := os.ReadFile(candidate)
					if err != nil {
						message = fmt.Sprintf("Error: %v", err)
						isError = true
					} else {
						content = string(data)
						if len(content) > 4096 {
							content = content[:4096] + "\n... (truncated)"
						}
					}
				}
			}
		}

		component := views.PathTraversalPage(h.securityEnabled(), loggedIn, username, filename, content, message, isError)
		renderHTML(c, http.StatusOK, "path_traversal", component)
	}
}

// validHostRE matches only safe hostname/IP characters; metacharacters are excluded.
// validHostRE matches only safe host characters for the secure ping path.
var validHostRE = regexp.MustCompile(`^[a-zA-Z0-9.\-]+$`)

// validatePingHost rejects empty values and shell metacharacters.
func validatePingHost(host string) error {
	if host == "" {
		return fmt.Errorf("host parameter required")
	}
	if !validHostRE.MatchString(host) {
		return fmt.Errorf("invalid host: only [a-zA-Z0-9.-] allowed")
	}
	return nil
}

// runSecurePing executes ping without a shell after validating the host.
//
// Inputs:
//   - ctx: request context used for cancellation.
//   - host: target host to ping.
//
// Outputs:
//   - string: combined command output.
//   - error: validation failure, timeout, or command error.
func runSecurePing(ctx context.Context, host string) (string, error) {
	if err := validatePingHost(host); err != nil {
		return "", err
	}

	pingCtx, cancel := context.WithTimeout(ctx, securePingTimeout)
	defer cancel()

	out, err := exec.CommandContext(pingCtx, "ping", "-c1", host).CombinedOutput() // #nosec G204
	if pingCtx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("ping timed out after %s", securePingTimeout)
	}
	// A non-zero ping exit code is still a valid command result to display.
	// The security invariant here is input validation + no shell + timeout.
	_ = err
	return string(out), nil
}

// PingVulnerable executes ping through a shell and is intentionally unsafe.
func (h *Handler) PingVulnerable() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Query("host")
		if host == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "host parameter required"})
			return
		}

		// VULNERABLE: no input validation — shell metacharacters execute.
		cmd := exec.Command("sh", "-c", "ping -c1 "+host) // #nosec G204
		out, _ := cmd.CombinedOutput()

		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, string(out))
	}
}

// PingSecure validates the host and executes ping without a shell.
func (h *Handler) PingSecure() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Query("host")
		if host == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "host parameter required"})
			return
		}

		// Secure: reject input that contains shell metacharacters.
		if err := validatePingHost(host); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  err.Error(),
				"detail": "shell metacharacters (;, &, |, $, `, etc.) are rejected",
			})
			return
		}

		out, err := runSecurePing(c.Request.Context(), host)
		if err != nil {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			c.String(http.StatusGatewayTimeout, out+"\n"+err.Error())
			return
		}
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, out)
	}
}

// PageCmdInjection renders the command-injection demo page.
func (h *Handler) PageCmdInjection() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		host := c.Query("host")
		var output, message string
		isError := false

		if host != "" {
			if !h.securityEnabled() {
				// VULNERABLE mode: run via shell — command injection possible.
				cmd := exec.Command("sh", "-c", "ping -c1 "+host) // #nosec G204
				out, _ := cmd.CombinedOutput()
				output = string(out)
				if len(output) > 4096 {
					output = output[:4096] + "\n... (truncated)"
				}
			} else {
				// Secure mode: validate first, then invoke without a shell.
				if err := validatePingHost(host); err != nil {
					message = "⛔ Command injection blocked: host contains disallowed characters — only [a-zA-Z0-9.-] are permitted"
					isError = true
				} else {
					var err error
					output, err = runSecurePing(c.Request.Context(), host)
					if err != nil {
						message = fmt.Sprintf("Ping failed: %v", err)
						isError = true
					}
					if len(output) > 4096 {
						output = output[:4096] + "\n... (truncated)"
					}
				}
			}
		}

		component := views.CmdInjectionPage(h.securityEnabled(), loggedIn, username, host, output, message, isError)
		renderHTML(c, http.StatusOK, "cmd_injection", component)
	}
}
