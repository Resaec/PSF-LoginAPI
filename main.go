package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"PSF-LoginAPI/endpoints"
	"PSF-LoginAPI/response"
	"PSF-LoginAPI/utils"
)

func main() {

	var (
		err error

		pool *pgxpool.Pool
	)

	// connect to db, create pool
	pool = utils.GetPostgrePool()

	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatalf("Could not open database connection: %v", err.Error())
	}

	// create router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	_ = router.SetTrustedProxies(nil)

	// add live group
	unauthenticated := router.Group("/live")
	{
		// setup routes
		unauthenticated.GET("/version", endpoints.Version)
		unauthenticated.POST("/login", endpoints.Login)
	}

	authenticated := router.Group("/live")
	{
		authenticated.Use(GetAuthMiddleware())

		authenticated.GET("/validate", endpoints.ValidateGet)
		authenticated.POST("/validate", endpoints.ValidatePost)

		authenticated.GET("/gametoken", endpoints.GameToken)
	}

	router.Run("localhost:9001")
}

func GetAuthMiddleware() gin.HandlerFunc {

	return func(gc *gin.Context) {

		var (
			err error

			token string

			decodedToken *jwt.Token
			claims       *jwt.MapClaims

			authHeader  = gc.Request.Header.Get("Authorization")
			splitHeader = strings.Split(authHeader, "Bearer ")
		)

		if len(splitHeader) < 2 {
			token = ""
		} else {
			token = splitHeader[1]
		}

		// token is required from here
		if token == "" {

			fmt.Println("Authenticated API called without token")

			gc.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// decode token
		decodedToken, claims, err = utils.ParseToken(token)
		if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {

			fmt.Printf("Authenticated API called with invalid token: %v\n", err.Error())

			gc.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if errors.Is(err, jwt.ErrTokenExpired) {
			gc.IndentedJSON(
				http.StatusOK,
				response.CreateErrorResponseWithText(
					response.ResponseErrorLauncherTokenExpired,
					"login expired",
				),
			)

			return
		}

		// some other reason the token is invalid
		if !decodedToken.Valid {

			fmt.Println("Authenticated API called with otherwise invalid token")

			gc.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// add token to context
		gc.Set("token", decodedToken)
		gc.Set("claims", *claims)

		// continue chained execution
		gc.Next()
	}

}
