package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"worldcup-stake/models"
)

func GetUsers(database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := database.Query("SELECT id, name FROM users ORDER BY name")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var users []models.User
		for rows.Next() {
			var u models.User
			if err := rows.Scan(&u.ID, &u.Name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			users = append(users, u)
		}
		if users == nil {
			users = []models.User{}
		}

		c.JSON(http.StatusOK, users)
	}
}
