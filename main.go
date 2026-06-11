package main

import (
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"worldcup-stake/db"
	"worldcup-stake/frontend"
	"worldcup-stake/handlers"
)

func main() {
	database, err := db.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := db.Seed(database); err != nil {
		log.Fatal(err)
	}

	apiKey := os.Getenv("FOOTBALL_DATA_API_KEY")
	pushService := handlers.NewPushService(database)
	syncer := handlers.NewSyncer(database, apiKey, pushService)

	go func() {
		if apiKey != "" {
			syncer.Sync()
		}
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			if apiKey != "" {
				syncer.Sync()
			}
		}
	}()

	sub, err := fs.Sub(frontend.Static, "dist")
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()

	api := r.Group("/api")
	{
		api.GET("/users", handlers.GetUsers(database))
		api.GET("/matches", handlers.GetMatches(database))
		api.GET("/leaderboard", handlers.GetLeaderboard(database))
		api.GET("/push/vapid-public-key", handlers.GetVAPIDPublicKey(pushService))
		api.POST("/push/subscribe", handlers.SubscribePush(database))
		api.PUT("/push/preferences", handlers.UpdatePushPreferences(database))
		api.POST("/push/unsubscribe", handlers.UnsubscribePush(database))
		api.POST("/sync", func(c *gin.Context) {
			if apiKey == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "FOOTBALL_DATA_API_KEY not set"})
				return
			}
			go syncer.Sync()
			c.JSON(http.StatusOK, gin.H{"status": "sync triggered"})
		})
	}

	// Serve static assets directly; all other routes get index.html for SPA routing
	r.GET("/assets/*filepath", func(c *gin.Context) {
		c.FileFromFS("assets/"+c.Param("filepath"), http.FS(sub))
	})
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = path[1:]
		}
		if data, err := fs.ReadFile(sub, path); err == nil {
			ext := filepath.Ext(path)
			if mime := mime.TypeByExtension(ext); mime != "" {
				c.Data(http.StatusOK, mime, data)
			} else {
				c.Data(http.StatusOK, "application/octet-stream", data)
			}
			return
		}
		data, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
