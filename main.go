package main

import (
	"database/sql"
	"log"
	"net/http"

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

// ---- Entry point ------------------------------------------------------------

func buildRouter(dbConn *sql.DB) *gin.Engine {
	router := gin.Default()
	router.Static("/static", "./static")

	svc := service.New(dbConn)
	h := handlers.New(svc, SecurityEnabled)

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	router.GET("/posts", h.GetPosts())
	router.POST("/login", h.PostLogin())

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/posts")
	})
	router.GET("/ui/posts", h.PagePosts())
	router.GET("/ui/login", h.PageLogin())
	router.POST("/ui/login", h.PageLoginSubmit())
	router.GET("/ui/partials/posts", h.PagePostsPartial())
	router.POST("/ui/partials/login", h.PageLoginPartial())

	return router
}

func main() {
	cfg := config.Load()
	SecurityEnabled = cfg.SecurityEnabled

	dbConn := db.InitDB(cfg.DBPath)
	defer dbConn.Close()

	db.MigrateDB(dbConn)
	db.SeedDB(dbConn)

	router := buildRouter(dbConn)

	log.Printf("Starting server on %s", cfg.Port)
	if err := router.Run(cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
