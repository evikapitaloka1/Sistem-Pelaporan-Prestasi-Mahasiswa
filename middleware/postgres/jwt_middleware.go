package routes

import (
	"net/http"
	"strings"

	"uas/app/model/postgres"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret = []byte("SECRET_KEY")

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenStr := strings.Replace(authHeader, "Bearer ", "", 1)
		token, err := jwt.ParseWithClaims(tokenStr, &model.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token tidak valid"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*model.UserClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "claims token tidak valid"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Next()
	}
}
