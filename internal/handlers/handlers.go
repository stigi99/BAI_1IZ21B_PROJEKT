package handlers

import (
	"fmt"
	"log"
	"net/http"

	"BAI_1IZ21B_PROJEKT/internal/service"
	"BAI_1IZ21B_PROJEKT/internal/views"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
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

		exists, err := h.svc.UserExists(req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		// NOTE: password is not verified in this phase (SecurityEnabled = false).
		// Password verification will be enforced when SecurityEnabled is set to true.
		c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
	}
}

func (h *Handler) PagePosts() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, err := h.svc.GetPublishedPosts()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to render posts page")
			return
		}

		component := views.PostsPage(posts, h.securityEnabled)
		renderHTML(c, http.StatusOK, "posts", component)
	}
}

func (h *Handler) PageLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		component := views.LoginPage(h.securityEnabled, "", false)
		renderHTML(c, http.StatusOK, "login", component)
	}
}

func (h *Handler) PagePostsPartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		posts, err := h.svc.GetPublishedPosts()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to render posts partial")
			return
		}

		component := views.PostsList(posts)
		renderHTML(c, http.StatusOK, "posts_partial", component)
	}
}

func (h *Handler) evaluateLogin(username, password string) (message string, isError bool, status int) {
	if username == "" || password == "" {
		return "Username and password are required", true, http.StatusBadRequest
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

func (h *Handler) PageLoginSubmit() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		message, isError, status := h.evaluateLogin(username, password)

		component := views.LoginPage(h.securityEnabled, message, isError)
		renderHTML(c, status, "login_submit", component)
	}
}

func (h *Handler) PageLoginPartial() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		message, isError, status := h.evaluateLogin(username, password)

		component := views.LoginResult(message, isError)
		renderHTML(c, status, "login_partial", component)
	}
}
