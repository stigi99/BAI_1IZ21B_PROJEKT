package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"BAI_1IZ21B_PROJEKT/internal/config"
	"BAI_1IZ21B_PROJEKT/internal/db"
	"BAI_1IZ21B_PROJEKT/internal/handlers"
	"BAI_1IZ21B_PROJEKT/internal/security"
	"BAI_1IZ21B_PROJEKT/internal/service"

	"github.com/gin-gonic/gin"
)

// uploadsDir is the shared on-disk location for user-uploaded attachments.
const uploadsDir = "./uploads"

// ---- Entry point ------------------------------------------------------------

// buildRouter creates the Gin engine, wires static assets, and registers every
// application route.
//
// Inputs:
//   - dbConn: database handle shared by handlers and services.
//   - initialSecurityEnabled: initial lab mode used to seed the mode store.
//
// Output:
//   - *gin.Engine: fully configured HTTP router ready to serve requests.
func buildRouter(dbConn *sql.DB, initialSecurityEnabled bool) *gin.Engine {
	router := gin.Default()
	router.Static("/static", "./static")
	router.Static("/uploads", uploadsDir)

	mode := security.NewModeStore(initialSecurityEnabled)
	router.Use(handlers.SecurityHeadersMiddleware(mode.Enabled))
	router.Use(handlers.ErrorSanitizerMiddleware(mode.Enabled))

	svc := service.NewWithMode(dbConn, mode)
	h := handlers.New(svc, initialSecurityEnabled)

	registerHealthRoutes(router)
	registerAPIRoutes(router, h)
	registerUIRoutes(router, h, mode)
	registerLabRoutes(router, h)

	return router
}

// registerHealthRoutes exposes a minimal liveness endpoint for local checks.
//
// Input:
//   - router: Gin engine that receives the /ping route.
func registerHealthRoutes(router *gin.Engine) {
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
}

// registerAPIRoutes binds the JSON API endpoints, including the intentionally
// vulnerable demo routes used by the security lab.
//
// Inputs:
//   - router: Gin engine that receives API routes.
//   - h: handler bundle that implements the request logic.
func registerAPIRoutes(router *gin.Engine, h *handlers.Handler) {
	router.GET("/posts", h.GetPosts())
	router.POST("/posts", h.PostCreate())
	router.PUT("/posts/:id", h.PostUpdate())
	router.DELETE("/posts/:id", h.PostDelete())
	router.POST("/login", h.PostLogin())
	router.POST("/register", h.PostRegister())
	router.GET("/ui/logout", h.Logout())
	router.POST("/logout", h.Logout())

	// Library search honors SECURITY_ENABLED, while /api/search-vulnerable is
	// kept as a forced vulnerable endpoint for repeatable curl/Burp checks.
	router.GET("/api/search", h.Search())
	router.GET("/api/search-vulnerable", h.SearchVulnerable())
	router.POST("/api/comments-vulnerable", h.CommentsVulnerable())
	router.POST("/api/comments-secure", h.CommentsSecure())
	router.GET("/csrf-vulnerable-form", h.CsrfFormVulnerable())
	router.POST("/csrf-vulnerable-form", h.CsrfFormVulnerable())
	router.GET("/debug/crash", h.DebugCrash())
}

// registerUIRoutes binds the browser-facing routes and the live security mode
// toggle used by the demo UI.
//
// Inputs:
//   - router: Gin engine that receives the HTML routes.
//   - h: handler bundle that renders pages and processes forms.
//   - mode: shared security-mode store used to flip secure/vulnerable state.
func registerUIRoutes(router *gin.Engine, h *handlers.Handler, mode *security.ModeStore) {
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
	router.GET("/ui/posts/:id", h.PagePostDetail())
	router.POST("/ui/posts/:id/comment", h.PagePostCommentSubmit())
	router.GET("/ui/login", h.PageLogin())
	router.POST("/ui/login", h.PageLoginSubmit())
	router.GET("/ui/register", h.PageRegister())
	router.POST("/ui/register", h.PageRegisterSubmit())
	router.GET("/ui/search", h.PageSearch())
	router.GET("/ui/library", h.PageSearch())
	router.GET("/ui/security-map", h.PageSecurityMap())
	router.GET("/ui/vuln-demos", h.PageVulnDemos())
	router.GET("/ui/partials/posts", h.PagePostsPartial())
	router.POST("/ui/partials/posts/create", h.PagePostsCreatePartial())
	router.POST("/ui/partials/login", h.PageLoginPartial())
	router.POST("/ui/partials/register", h.PageRegisterPartial())
	router.POST("/ui/partials/search", h.PageSearchPartial())
	router.POST("/ui/mode/toggle", func(c *gin.Context) {
		h.SetSecurityEnabled(mode.Toggle())

		next := c.PostForm("next")
		if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
			next = "/ui/security-map"
		}
		c.Redirect(http.StatusSeeOther, next)
	})
}

// registerLabRoutes keeps the legacy demo URLs and the feature-first UI routes
// available side by side so existing scripts and report links keep working.
//
// Inputs:
//   - router: Gin engine that receives the lab and alias routes.
//   - h: handler bundle that serves the security lab scenarios.
func registerLabRoutes(router *gin.Engine, h *handlers.Handler) {
	// Feature-first UI routes. Legacy "demo" routes remain registered as aliases
	// so older scripts and report links keep working.
	router.GET("/ui/profile", h.ProfileSettings())
	router.POST("/ui/profile", h.ProfileSettings())
	router.GET("/ui/profile-secure", h.CsrfSecureForm())
	router.POST("/ui/profile-secure", h.CsrfSecureForm())
	router.GET("/ui/moderation", h.PageIDOR())
	router.GET("/ui/members", h.PageDBExpose())
	router.GET("/ui/gallery", h.PagePathTraversal())
	router.GET("/ui/stream-check", h.PageCmdInjection())

	router.GET("/ui/account", h.ProfileSettings())
	router.POST("/ui/account", h.ProfileSettings())
	router.GET("/ui/account-secure", h.CsrfSecureForm())
	router.POST("/ui/account-secure", h.CsrfSecureForm())
	router.GET("/ui/access-control", h.PageIDOR())
	router.GET("/ui/database", h.PageDBExpose())
	router.GET("/ui/files", h.PagePathTraversal())
	router.GET("/ui/tools/ping", h.PageCmdInjection())
	router.GET("/ui/csrf-demo", h.CsrfFormVulnerable())
	router.POST("/ui/csrf-demo", h.CsrfFormVulnerable())
	router.GET("/ui/csrf-secure", h.CsrfSecureForm())
	router.POST("/ui/csrf-secure", h.CsrfSecureForm())
	router.GET("/ui/idor-demo", h.PageIDOR())
	router.GET("/ui/db-expose", h.PageDBExpose())
	router.GET("/ui/path-traversal", h.PagePathTraversal())
	router.GET("/api/files-vulnerable", h.FilesVulnerable())
	router.GET("/api/files-secure", h.FilesSecure())
	router.GET("/ui/cmd-injection", h.PageCmdInjection())
	router.GET("/api/ping-vulnerable", h.PingVulnerable())
	router.GET("/api/ping-secure", h.PingSecure())
}

// main loads configuration, initializes storage, seeds demo data, builds the
// router, and starts the HTTP server.
func main() {
	cfg := config.Load()

	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		log.Fatalf("Failed to create uploads dir: %v", err)
	}

	dbConn := db.InitDB(cfg.DBPath)
	defer dbConn.Close()

	db.MigrateDB(dbConn)
	db.SeedDB(dbConn, cfg.SecurityEnabled)

	router := buildRouter(dbConn, cfg.SecurityEnabled)

	log.Printf("Starting server on %s (SECURITY_ENABLED=%v)", cfg.Port, cfg.SecurityEnabled)
	if err := router.Run(cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
