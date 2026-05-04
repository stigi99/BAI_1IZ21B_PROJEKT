package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"BAI_1IZ21B_PROJEKT/internal/config"
	"BAI_1IZ21B_PROJEKT/internal/db"
	"BAI_1IZ21B_PROJEKT/internal/handlers"
	"BAI_1IZ21B_PROJEKT/internal/service"

	"github.com/gin-gonic/gin"
)

// SecurityEnabled acts as a global security toggle.
// When false the application runs without security controls (insecure mode).
// When true proper authentication and input validation are enforced.
var SecurityEnabled = false

const uploadsDir = "./uploads"

// ---- Entry point ------------------------------------------------------------

func buildRouter(dbConn *sql.DB) *gin.Engine {
	router := gin.Default()
	router.Static("/static", "./static")
	router.Static("/uploads", uploadsDir)

	svc := service.New(dbConn, SecurityEnabled)
	h := handlers.New(svc, SecurityEnabled)

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	router.GET("/posts", h.GetPosts())
	router.POST("/posts", h.PostCreate())
	router.PUT("/posts/:id", h.PostUpdate())
	router.DELETE("/posts/:id", h.PostDelete())
	router.POST("/login", h.PostLogin())
	router.POST("/register", h.PostRegister())
	router.GET("/ui/logout", h.Logout())
	router.POST("/logout", h.Logout())

	// SQL Injection demo: /api/search honors SECURITY_ENABLED, while
	// /api/search-vulnerable is always concatenation-based (forced vulnerable).
	router.GET("/api/search", h.Search())
	router.GET("/api/search-vulnerable", h.SearchVulnerable())
	router.POST("/api/comments-vulnerable", h.CommentsVulnerable())
	router.GET("/csrf-vulnerable-form", h.CsrfFormVulnerable())
	router.POST("/csrf-vulnerable-form", h.CsrfFormVulnerable())

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/posts")
	})
	router.GET("/ui/posts", h.PagePosts())
	router.POST("/ui/posts/create", h.PagePostsCreate())
	router.GET("/ui/posts/view/:id", h.PagePostDetail())
	router.POST("/ui/posts/view/:id/comments", h.PagePostCommentSubmit())
	router.POST("/ui/partials/posts/view/:id/comments", h.PagePostCommentSubmit())
	router.GET("/ui/posts/edit/:id", h.PagePostEdit())
	router.POST("/ui/posts/edit/:id", h.PagePostEditSubmit())
	router.POST("/ui/posts/delete/:id", h.PagePostDelete())
	router.GET("/ui/login", h.PageLogin())
	router.POST("/ui/login", h.PageLoginSubmit())
	router.GET("/ui/register", h.PageRegister())
	router.POST("/ui/register", h.PageRegisterSubmit())
	router.GET("/ui/search", h.PageSearch())
	router.GET("/ui/vuln-demos", h.PageVulnDemos())
	router.GET("/ui/partials/posts", h.PagePostsPartial())
	router.POST("/ui/partials/posts/create", h.PagePostsCreatePartial())
	router.POST("/ui/partials/login", h.PageLoginPartial())
	router.POST("/ui/partials/register", h.PageRegisterPartial())
	router.POST("/ui/partials/search", h.PageSearchPartial())

	return router
}

func main() {
	cfg := config.Load()
	SecurityEnabled = cfg.SecurityEnabled

	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		log.Fatalf("Failed to create uploads dir: %v", err)
	}

	dbConn := db.InitDB(cfg.DBPath)
	defer dbConn.Close()

	db.MigrateDB(dbConn)
	db.SeedDB(dbConn, SecurityEnabled)

	router := buildRouter(dbConn)

	log.Printf("Starting server on %s (SECURITY_ENABLED=%v)", cfg.Port, SecurityEnabled)
	if err := router.Run(cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
