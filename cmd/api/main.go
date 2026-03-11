package main

import (
	"i9pay/internal/database"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	db, err := database.InitDB()

	if err != nil {
		log.Fatalf("Erro ao conectar ao banco: %v", err)
	}
	defer db.Close()

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "live",
		})
	})

	r.Run(":8080")
}
