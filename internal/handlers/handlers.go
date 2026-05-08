package handlers

import (
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
	authCookieName  = "bai_auth_user"
	csrfCookieName  = "bai_csrf_token"
	uploadsDir      = "./uploads"
	uploadsURLPath  = "/uploads"
	maxUploadBytes  = 5 << 20 // 5 MiB
	// sessionSecret is used to HMAC-sign the auth cookie in secure mode.
	// In a production system this must come from a securely stored secret.
	sessionSecret = "bai-lab-demo-secret-do-not-use-in-prod"
)

type Handler struct {
	svc             *service.Service
	securityEnabled bool
}

func New(svc *service.Service, securityEnabled bool) *Handler {
	return &Handler{svc: svc, securityEnabled: securityEnabled}
}

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
	mac := hmac.New(sha256.New, []byte(sessionSecret))
	mac.Write([]byte(username))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return username + "|" + sig
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

func (h *Handler) setAuthCookie(c *gin.Context, username string) {
	cookieVal := username
	if h.securityEnabled {
		// Secure mode: HMAC-signed cookie + SameSite=Strict.
		cookieVal = signCookieValue(username)
	}
	// HttpOnly=true in both modes; Secure=false since the lab runs on plain HTTP.
	c.SetCookie(authCookieName, cookieVal, 60*60*8, "/", "", false, true)
	if h.securityEnabled {
		// Append SameSite=Strict for the secure mode cookie.
		existing := c.Writer.Header().Get("Set-Cookie")
		c.Writer.Header().Set("Set-Cookie", existing+"; SameSite=Strict")
	}
}

func (h *Handler) currentUsername(c *gin.Context) (string, bool) {
	raw, err := c.Cookie(authCookieName)
	if err != nil || raw == "" {
		return "", false
	}

	var username string
	if h.securityEnabled {
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

func (h *Handler) requireLoginJSON(c *gin.Context) bool {
	if _, ok := h.currentUsername(c); ok {
		return true
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Login required"})
	return false
}

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

// PostLogin accepts JSON with username and password. Honours the SECURITY_ENABLED toggle.
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

		if err := h.svc.UpdatePost(id, req.Title, req.PostContent, req.Published, "", ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Post updated"})
	}
}

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

		if h.securityEnabled {
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

func (h *Handler) PagePosts() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, err := h.svc.GetAllPosts()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to render posts page")
			return
		}

		message := c.Query("msg")
		isError := c.Query("err") == "1"
		username, loggedIn := h.currentUsername(c)
		if !loggedIn && message == "" {
			message = "Please log in to add, edit, or delete posts"
			isError = true
		}
		component := views.PostsPage(posts, h.securityEnabled, loggedIn, username, message, isError)
		renderHTML(c, http.StatusOK, "posts", component)
	}
}

func (h *Handler) PageLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		component := views.LoginPage(h.securityEnabled, loggedIn, username, "", false)
		renderHTML(c, http.StatusOK, "login", component)
	}
}

func (h *Handler) PageRegister() gin.HandlerFunc {
	return func(c *gin.Context) {
		message := c.Query("msg")
		isError := c.Query("err") == "1"
		username, loggedIn := h.currentUsername(c)
		component := views.RegisterPage(h.securityEnabled, loggedIn, username, message, isError)
		renderHTML(c, http.StatusOK, "register", component)
	}
}

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

func (h *Handler) PagePostsPartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, err := h.svc.GetAllPosts()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to render posts partial")
			return
		}

		username, loggedIn := h.currentUsername(c)
		component := views.PostsListContainer(posts, loggedIn, username, h.securityEnabled)
		renderHTML(c, http.StatusOK, "posts_partial", component)
	}
}

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
			component := views.PostsListWithBanner(posts, loggedIn, username, h.securityEnabled, "Title is required", true)
			renderHTML(c, http.StatusBadRequest, "posts_create_partial", component)
			return
		}

		attachmentPath, attachmentName, err := h.saveUploadedAttachment(c, "attachment")
		if err != nil {
			posts, _ := h.svc.GetAllPosts()
			component := views.PostsListWithBanner(posts, loggedIn, username, h.securityEnabled, "Failed to save attachment: "+err.Error(), true)
			renderHTML(c, http.StatusBadRequest, "posts_create_partial", component)
			return
		}

		if _, err := h.svc.CreatePost(title, content, published, username, attachmentPath, attachmentName); err != nil {
			posts, _ := h.svc.GetAllPosts()
			component := views.PostsListWithBanner(posts, loggedIn, username, h.securityEnabled, "Failed to create post", true)
			renderHTML(c, http.StatusInternalServerError, "posts_create_partial", component)
			return
		}

		posts, _ := h.svc.GetAllPosts()
		component := views.PostsListWithBanner(posts, loggedIn, username, h.securityEnabled, "Post created", false)
		c.Header("HX-Trigger", "post-created")
		renderHTML(c, http.StatusOK, "posts_create_partial", component)
	}
}

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
		component := views.EditPostPage(h.securityEnabled, loggedIn, username, post, message, isError)
		renderHTML(c, http.StatusOK, "post_edit", component)
	}
}

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

		if h.securityEnabled {
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

func (h *Handler) evaluateLogin(username, password string) (message string, isError bool, status int) {
	if username == "" || password == "" {
		return "Username and password are required", true, http.StatusBadRequest
	}

	// Secure mode: enforce rate limiting to block brute-force attacks.
	if h.securityEnabled && !h.svc.CheckRateLimit(username) {
		return "Too many failed attempts. Please wait 60 seconds.", true, http.StatusTooManyRequests
	}

	if h.securityEnabled {
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
func (h *Handler) Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie(authCookieName, "", -1, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/ui/posts")
	}
}

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

func (h *Handler) PageLoginSubmit() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		message, isError, status := h.evaluateLogin(username, password)
		if !isError {
			h.setAuthCookie(c, username)
		}

		loggedIn := !isError
		component := views.LoginPage(h.securityEnabled, loggedIn, username, message, isError)
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
			"mode":    modeLabel(h.securityEnabled),
			"results": posts,
		})
	}
}

// SearchVulnerable is a force-vulnerable SQL Injection endpoint that always
// uses string concatenation, regardless of SECURITY_ENABLED. Kept for the
// side-by-side defense demo.
//
// Try: /api/search-vulnerable?q=' OR 1=1 --
// Or:  /api/search-vulnerable?q=' UNION SELECT id, username, password_hash, 1, '', '', '' FROM users --
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
				component := views.SearchPage(h.securityEnabled, loggedIn, username, query, nil, "Search failed: "+err.Error(), true)
				renderHTML(c, http.StatusOK, "search", component)
				return
			}
		}
		component := views.SearchPage(h.securityEnabled, loggedIn, username, query, posts, "", false)
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
			component := views.SearchResults("", nil, "Type something to search", true, h.securityEnabled)
			renderHTML(c, http.StatusOK, "search_partial", component)
			return
		}

		posts, err := h.svc.SearchPosts(query)
		if err != nil {
			component := views.SearchResults(query, nil, "Search failed: "+err.Error(), true, h.securityEnabled)
			renderHTML(c, http.StatusInternalServerError, "search_partial", component)
			return
		}
		component := views.SearchResults(query, posts, "", false, h.securityEnabled)
		renderHTML(c, http.StatusOK, "search_partial", component)
	}
}

func modeLabel(secure bool) string {
	if secure {
		return "secure (parameterized)"
	}
	return "vulnerable (concatenated)"
}

// PageVulnDemos renders the security demos hub.
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
				Description: "Vulnerable mode concatenates the search term into a SELECT — try ' OR 1=1 -- to leak drafts.",
				Href:        "/ui/search",
				Payload:     `' UNION SELECT id, username, password_hash, 1, '', '', '' FROM users --`,
			},
			{
				Emoji:       "🪲",
				Title:       "Stored XSS",
				CWE:         "CWE-79",
				OWASP:       "A03:2021",
				Status:      "ready",
				Description: "Comments rendered as raw HTML in vulnerable mode. Open the Stored XSS demo post and submit a payload.",
				Href:        "/ui/posts/view/1",
				Payload:     `<script>alert('XSS-' + document.cookie)</script>`,
			},
			{
				Emoji:       "🔐",
				Title:       "Broken Authentication",
				CWE:         "CWE-287",
				OWASP:       "A07:2021",
				Status:      "ready",
				Description: "Vulnerable mode accepts any password for an existing user. Secure mode validates with bcrypt.",
				Href:        "/ui/login",
				Payload:     `username=admin&password=anything`,
			},
			{
				Emoji:       "🔓",
				Title:       "Broken Access Control",
				CWE:         "CWE-639",
				OWASP:       "A01:2021",
				Status:      "ready",
				Description: "In vulnerable mode any logged-in user can delete any post. Secure mode enforces author/admin only.",
				Href:        "/ui/idor-demo",
				Payload:     `POST /ui/posts/delete/{otherUsersPostId}`,
			},
			{
				Emoji:       "🗝️",
				Title:       "Sensitive Data Exposure",
				CWE:         "CWE-200",
				OWASP:       "A02:2021",
				Status:      "ready",
				Description: "Vulnerable mode stores plaintext passwords in SQLite. Secure mode persists bcrypt hashes only.",
				Href:        "/ui/db-expose",
				Payload:     `sqlite> SELECT username, password_hash FROM users;`,
			},
			{
				Emoji:       "🪪",
				Title:       "CSRF",
				CWE:         "CWE-352",
				OWASP:       "A01:2021",
				Status:      "ready",
				Description: "Form POST without anti-CSRF token can be forged cross-origin.",
				Href:        "/ui/csrf-demo",
				Payload:     `<form action="/ui/csrf-demo" method="POST"><input name="new_email" value="hacked@evil.com"></form>`,
			},
		}
		component := views.VulnDemosPage(h.securityEnabled, loggedIn, username, demos)
		renderHTML(c, http.StatusOK, "vuln_demos", component)
	}
}

// CommentsVulnerable stores comments without sanitization (Stored XSS demo).
// The raw HTML body is persisted directly so <script> tags execute on render.
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
			component := views.CSRFDemoPage(h.securityEnabled, loggedIn, username, "", email, "", false)
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
			component := views.CSRFDemoPage(h.securityEnabled, loggedIn, username, "", email, "Email cannot be empty", true)
			renderHTML(c, http.StatusBadRequest, "csrf_demo", component)
			return
		}
		if err := h.svc.UpdateUserEmail(username, newEmail); err != nil {
			component := views.CSRFDemoPage(h.securityEnabled, loggedIn, username, "", newEmail, "Failed to update email", true)
			renderHTML(c, http.StatusInternalServerError, "csrf_demo", component)
			return
		}
		component := views.CSRFDemoPage(h.securityEnabled, loggedIn, username, "", newEmail,
			fmt.Sprintf("Email updated to %s (no CSRF token was checked!)", newEmail), false)
		renderHTML(c, http.StatusOK, "csrf_demo", component)
	}
}

// ============================================================================
// NEW HANDLERS — XSS, CSRF secure, IDOR, DB Expose, Path Traversal
// ============================================================================

// CommentsSecure stores a comment after HTML-escaping the body.
// Used by the secure JSON API endpoint for the XSS demo.
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

// PagePostDetail renders a single post with its comment thread.
// Comments are rendered with templ.Raw in insecure mode (Stored XSS) and as
// plain text in secure mode.
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
		component := views.PostDetailPage(post, comments, h.securityEnabled, loggedIn, username, message, isError)
		renderHTML(c, http.StatusOK, "post_detail", component)
	}
}

// PagePostCommentSubmit processes a form-based comment submission from the
// post detail page.
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

		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/ui/posts/%d?msg=Comment+added", id))
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

// CsrfSecureForm handles the CSRF-protected email-update form.
// GET: issues a new CSRF token (stored in a cookie and embedded in the form).
// POST: validates the token before performing the email update.
func (h *Handler) CsrfSecureForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)

		if c.Request.Method == http.MethodGet {
			token := generateCSRFToken()
			c.SetCookie(csrfCookieName, token, 60*60, "/", "", false, true)
			email := ""
			if loggedIn {
				email, _ = h.svc.GetUserEmail(username)
			}
			component := views.CSRFDemoPage(h.securityEnabled, loggedIn, username, token, email, "", false)
			renderHTML(c, http.StatusOK, "csrf_secure", component)
			return
		}

		// POST — secure: validate CSRF token.
		formToken := c.PostForm("csrf_token")
		cookieToken, cookieErr := c.Cookie(csrfCookieName)
		if cookieErr != nil || formToken == "" || formToken != cookieToken {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token validation failed"})
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
			c.SetCookie(csrfCookieName, token, 60*60, "/", "", false, true)
			component := views.CSRFDemoPage(h.securityEnabled, loggedIn, username, token, email, "Email cannot be empty", true)
			renderHTML(c, http.StatusBadRequest, "csrf_secure", component)
			return
		}

		if err := h.svc.UpdateUserEmail(username, newEmail); err != nil {
			token := generateCSRFToken()
			c.SetCookie(csrfCookieName, token, 60*60, "/", "", false, true)
			component := views.CSRFDemoPage(h.securityEnabled, loggedIn, username, token, newEmail, "Failed to update email", true)
			renderHTML(c, http.StatusInternalServerError, "csrf_secure", component)
			return
		}

		token := generateCSRFToken()
		c.SetCookie(csrfCookieName, token, 60*60, "/", "", false, true)
		component := views.CSRFDemoPage(h.securityEnabled, loggedIn, username, token, newEmail,
			fmt.Sprintf("Email updated to %s (CSRF token was valid ✓)", newEmail), false)
		renderHTML(c, http.StatusOK, "csrf_secure", component)
	}
}

// PageIDOR renders the Broken Access Control / IDOR demo page.
func (h *Handler) PageIDOR() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, _ := h.svc.GetAllPosts()
		username, loggedIn := h.currentUsername(c)
		message := c.Query("msg")
		isError := c.Query("err") == "1"
		component := views.IDORDemoPage(posts, h.securityEnabled, loggedIn, username, message, isError)
		renderHTML(c, http.StatusOK, "idor_demo", component)
	}
}

// PageDBExpose renders the Sensitive Data Exposure demo page.
// In insecure mode it lists all users with their plaintext passwords.
// In secure mode it returns a 403 page showing only hashed passwords.
func (h *Handler) PageDBExpose() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		users, err := h.svc.GetAllUsers()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load users")
			return
		}
		component := views.DBExposePage(users, h.securityEnabled, loggedIn, username)
		renderHTML(c, http.StatusOK, "db_expose", component)
	}
}

// FilesVulnerable reads a file from the uploads directory by directly
// concatenating the user-supplied filename — exploitable with path traversal.
//
// Example: /api/files-vulnerable?name=../app.db
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

// FilesSecure reads a file from the uploads directory after validating that the
// resolved path stays within the uploads directory.
//
// Example (blocked): /api/files-secure?name=../app.db → 400
func (h *Handler) FilesSecure() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name parameter required"})
			return
		}

		// Secure: canonicalize and verify the path stays within uploadsDir.
		base, _ := filepath.Abs(uploadsDir)
		candidate := filepath.Join(base, filepath.Clean("/"+name))
		if !strings.HasPrefix(candidate, base+string(filepath.Separator)) && candidate != base {
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

// PagePathTraversal renders the Path Traversal / LFI demo page.
func (h *Handler) PagePathTraversal() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, loggedIn := h.currentUsername(c)
		filename := c.Query("name")
		var content, message string
		isError := false

		if filename != "" {
			endpoint := "/api/files-vulnerable"
			if h.securityEnabled {
				endpoint = "/api/files-secure"
			}
			_ = endpoint // displayed in view; actual read below

			// Read via the appropriate path validation strategy.
			if !h.securityEnabled {
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
				base, _ := filepath.Abs(uploadsDir)
				candidate := filepath.Join(base, filepath.Clean("/"+filename))
				if !strings.HasPrefix(candidate, base+string(filepath.Separator)) && candidate != base {
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

		component := views.PathTraversalPage(h.securityEnabled, loggedIn, username, filename, content, message, isError)
		renderHTML(c, http.StatusOK, "path_traversal", component)
	}
}

// validHostRE matches only safe hostname/IP characters; metacharacters are excluded.
var validHostRE = regexp.MustCompile(`^[a-zA-Z0-9.\-]+$`)

// PingVulnerable runs ping by concatenating user input into sh -c.
// This is intentionally vulnerable to command injection for the lab demo.
//
// Example exploit: /api/ping-vulnerable?host=8.8.8.8+;+cat+/etc/passwd
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

// PingSecure runs ping after validating that the host contains only safe characters.
// The command is invoked directly without a shell so metacharacters have no effect.
//
// Example (blocked): /api/ping-secure?host=8.8.8.8+;+cat+/etc/passwd → 400
func (h *Handler) PingSecure() gin.HandlerFunc {
return func(c *gin.Context) {
host := c.Query("host")
if host == "" {
c.JSON(http.StatusBadRequest, gin.H{"error": "host parameter required"})
return
}

// Secure: reject input that contains shell metacharacters.
if !validHostRE.MatchString(host) {
c.JSON(http.StatusBadRequest, gin.H{
"error":  "invalid host: only [a-zA-Z0-9.-] allowed",
"detail": "shell metacharacters (;, &, |, $, `, etc.) are rejected",
})
return
}

out, _ := exec.Command("ping", "-c1", host).CombinedOutput() // #nosec G204
c.Header("Content-Type", "text/plain; charset=utf-8")
c.String(http.StatusOK, string(out))
}
}

// PageCmdInjection renders the Command Injection demo page.
// When a host query parameter is present it runs the ping and shows the output.
func (h *Handler) PageCmdInjection() gin.HandlerFunc {
return func(c *gin.Context) {
username, loggedIn := h.currentUsername(c)
host := c.Query("host")
var output, message string
isError := false

if host != "" {
if !h.securityEnabled {
// VULNERABLE mode: run via shell — command injection possible.
cmd := exec.Command("sh", "-c", "ping -c1 "+host) // #nosec G204
out, _ := cmd.CombinedOutput()
output = string(out)
if len(output) > 4096 {
output = output[:4096] + "\n... (truncated)"
}
} else {
// Secure mode: validate first, then invoke without a shell.
if !validHostRE.MatchString(host) {
message = "⛔ Command injection blocked: host contains disallowed characters — only [a-zA-Z0-9.-] are permitted"
isError = true
} else {
out, _ := exec.Command("ping", "-c1", host).CombinedOutput() // #nosec G204
output = string(out)
if len(output) > 4096 {
output = output[:4096] + "\n... (truncated)"
}
}
}
}

component := views.CmdInjectionPage(h.securityEnabled, loggedIn, username, host, output, message, isError)
renderHTML(c, http.StatusOK, "cmd_injection", component)
}
}
