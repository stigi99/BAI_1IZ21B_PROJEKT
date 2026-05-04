package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
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
	uploadsDir      = "./uploads"
	uploadsURLPath  = "/uploads"
	maxUploadBytes  = 5 << 20 // 5 MiB
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

func (h *Handler) setAuthCookie(c *gin.Context, username string) {
	c.SetCookie(authCookieName, username, 60*60*8, "/", "", false, true)
}

func (h *Handler) currentUsername(c *gin.Context) (string, bool) {
	username, err := c.Cookie(authCookieName)
	if err != nil || username == "" {
		return "", false
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

	if h.securityEnabled {
		valid, err := h.svc.ValidateUserCredentials(username, password)
		if err != nil {
			return "Database error", true, http.StatusInternalServerError
		}
		if !valid {
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

// PageVulnDemos renders the Vuln Demos hub — one card per scenario in the
// curriculum, with quick links and current status.
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
				Href:        "/ui/posts/view/3",
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
				Href:        "/ui/posts",
				Payload:     `POST /ui/posts/delete/{otherUsersPostId}`,
			},
			{
				Emoji:       "🗝️",
				Title:       "Sensitive Data Exposure",
				CWE:         "CWE-200",
				OWASP:       "A02:2021",
				Status:      "ready",
				Description: "Vulnerable mode stores plaintext passwords in SQLite. Secure mode persists bcrypt hashes only.",
				Href:        "/ui/register",
				Payload:     `sqlite> SELECT username, password_hash FROM users;`,
			},
			{
				Emoji:       "🪪",
				Title:       "CSRF",
				CWE:         "CWE-352",
				OWASP:       "A01:2021",
				Status:      "todo",
				Description: "Form POST without anti-CSRF token. Endpoint exists at /csrf-vulnerable-form for the demo.",
				Href:        "/csrf-vulnerable-form",
				Payload:     `<form action="/csrf-vulnerable-form" method="POST"><input name=action value=transfer_funds>...`,
			},
		}
		component := views.VulnDemosPage(h.securityEnabled, loggedIn, username, demos)
		renderHTML(c, http.StatusOK, "vuln_demos", component)
	}
}

// PagePostDetail renders a single post with its comments (Stored XSS demo).
// In secure mode comment bodies are HTML-escaped on render; in insecure mode
// they are rendered as raw HTML.
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
		comments, err := h.svc.GetCommentsForPost(id)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load comments")
			return
		}
		username, loggedIn := h.currentUsername(c)
		message := c.Query("msg")
		isError := c.Query("err") == "1"
		component := views.PostDetailPage(h.securityEnabled, loggedIn, username, post, comments, message, isError)
		renderHTML(c, http.StatusOK, "post_detail", component)
	}
}

// PagePostCommentSubmit accepts a form POST and renders the refreshed comments
// list. Uses CreateComment which honors SECURITY_ENABLED (sanitize on store).
func (h *Handler) PagePostCommentSubmit() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid post id")
			return
		}
		body := c.PostForm("body")
		if body == "" {
			c.String(http.StatusBadRequest, "body required")
			return
		}
		author := ""
		if username, ok := h.currentUsername(c); ok {
			author = username
		}
		if _, err := h.svc.CreateComment(id, author, body); err != nil {
			c.String(http.StatusInternalServerError, "Failed to save comment")
			return
		}
		comments, err := h.svc.GetCommentsForPost(id)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load comments")
			return
		}
		component := views.CommentsList(comments, h.securityEnabled)
		renderHTML(c, http.StatusOK, "comments_list", component)
	}
}

// CommentsVulnerable always stores comment bodies verbatim and returns the
// comments list as raw HTML so the XSS payload fires regardless of the
// SECURITY_ENABLED toggle. Used for the side-by-side defense demo.
//
// Body: form-encoded `post_id` + `body` (or JSON with the same fields).
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
func (h *Handler) CsrfFormVulnerable() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			html := `<!DOCTYPE html>
	<html>
	<head><title>Vulnerable Form (CSRF)</title></head>
	<body>
	<h1>VULNERABLE: This form has no CSRF protection</h1>
	<form method="POST" action="/csrf-vulnerable-form">
	  <input type="hidden" name="action" value="transfer_funds">
	  <input type="text" name="amount" placeholder="Amount" required>
	  <input type="text" name="to_account" placeholder="To Account" required>
	  <button type="submit">Transfer Funds</button>
	</form>
	</body>
	</html>`
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}

		action := c.PostForm("action")
		amount := c.PostForm("amount")
		toAccount := c.PostForm("to_account")

		c.JSON(http.StatusOK, gin.H{
			"message": "Action completed (no CSRF protection)",
			"action":  action,
			"amount":  amount,
			"to":      toAccount,
		})
	}
}
