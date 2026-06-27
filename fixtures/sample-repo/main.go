package main

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
		c.JSON(200, gin.H{"hash": string(hash)})
	})
	_ = r.Run(":8080")
}
