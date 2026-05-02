package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"BAI_1IZ21B_PROJEKT/internal/service"
	"BAI_1IZ21B_PROJEKT/internal/views"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

const authCookieName = "bai_auth_user"

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
	if !h.securityEnabled {
		return
	}

	c.SetCookie(authCookieName, username, 60*60*8, "/", "", false, true)
}

func (h *Handler) currentUsername(c *gin.Context) (string, bool) {
	if !h.securityEnabled {
		return "", true
	}

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
	if !h.securityEnabled {
		return true
	}

	if _, ok := h.currentUsername(c); ok {
		return true
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Login required"})
	return false
}

func (h *Handler) requireLoginUI(c *gin.Context, fallback string) bool {
	if !h.securityEnabled {
		return true
	}

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

// PostLogin accepts JSON with username and password.
// In the current (insecure) implementation it returns success if the username
// exists in the database, without verifying the password.
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

		id, err := h.svc.CreatePost(req.Title, req.PostContent, req.Published, author)
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

		if err := h.svc.UpdatePost(id, req.Title, req.PostContent, req.Published); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Post updated"})
	}
}

func (h *Handler) PostDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		if h.securityEnabled && !loggedIn && message == "" {
			message = "Please log in to add, edit, or delete posts"
			isError = true
		}
		component := views.PostsPage(posts, h.securityEnabled, loggedIn, username, message, isError)
		renderHTML(c, http.StatusOK, "posts", component)
	}
}

func (h *Handler) PageLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		component := views.LoginPage(h.securityEnabled, "", false)
		renderHTML(c, http.StatusOK, "login", component)
	}
}

func (h *Handler) PageRegister() gin.HandlerFunc {
	return func(c *gin.Context) {
		message := c.Query("msg")
		isError := c.Query("err") == "1"
		component := views.RegisterPage(h.securityEnabled, message, isError)
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

func (h *Handler) PagePostsPartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, err := h.svc.GetAllPosts()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to render posts partial")
			return
		}

		_, loggedIn := h.currentUsername(c)
		component := views.PostsList(posts, !h.securityEnabled || loggedIn)
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

		if _, err := h.svc.CreatePost(title, content, published, author); err != nil {
			c.Redirect(http.StatusSeeOther, "/ui/posts?err=1&msg=Failed+to+create+post")
			return
		}

		c.Redirect(http.StatusSeeOther, "/ui/posts?msg=Post+created")
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
		component := views.EditPostPage(h.securityEnabled, post, message, isError)
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

		if err := h.svc.UpdatePost(id, title, content, published); err != nil {
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

	exists, err := h.svc.UserExists(username)
	if err != nil {
		return "Database error", true, http.StatusInternalServerError
	}
	if !exists {
		return "User not found", true, http.StatusUnauthorized
	}

	return fmt.Sprintf("Login successful for %s", username), false, http.StatusOK
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

		component := views.LoginPage(h.securityEnabled, message, isError)
		renderHTML(c, status, "login_submit", component)
	}
}

func (h *Handler) PageLoginPartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		message, isError, status := h.evaluateLogin(username, password)
		if !isError {
			h.setAuthCookie(c, username)
		}

		component := views.LoginResult(message, isError)
		renderHTML(c, status, "login_partial", component)
	}
}
