package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"PSF-LoginAPI/response"
	"PSF-LoginAPI/utils"
)

func GameToken(gc *gin.Context) {

	var (
		exists bool

		// token can only be a maximum of 31 characters (31 + \0)
		gameToken = utils.RandString(31)

		pClaims, _ = gc.Get("claims")
		claims     = pClaims.(jwt.MapClaims)

		account, _ = claims["account"].(json.Number).Int64()
		mode, _    = claims["mode"].(json.Number).Int64()
	)

	_, exists = claims["verified"]
	if !exists {

		fmt.Printf(
			"Account ID [%d] mode [%d] requested a gametoken before verification\n",
			account,
			mode,
		)

		gc.IndentedJSON(
			http.StatusOK,
			response.CreateErrorResponse(response.ResponseErrorLauncherGameTokenRequestNotVerified),
		)

		return
	}

	setTokenOnAccount(account, gameToken)

	gc.IndentedJSON(
		http.StatusOK,
		response.GameTokenResponse{
			DefaultResponse: response.DefaultResponse{
				Status: response.ResponseErrorSuccess,
			},
			GameToken: gameToken,
		},
	)

	return
}

func setTokenOnAccount(account int64, gameToken string) (statusCode int) {

	var (
		err error
	)

	_, err = utils.GetPostgrePool().Query(
		context.Background(),
		`UPDATE "account" SET "token" = $1 WHERE "id" = $2`,
		gameToken,
		account,
	)
	if err != nil {
		statusCode = response.ResponseErrorDatabase

		fmt.Printf("Error writing game token to account %d: %s\n", account, err.Error())

		return
	}

	return
}
