package endpoints

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"

	"PSF-LoginAPI/response"
	"PSF-LoginAPI/utils"
)

const fileHashQuery = `
SELECT %s
FROM filehash
WHERE
		"mode" = $1
	OR (
			"mode" = 0
		AND
			NOT EXISTS (
				SELECT 1
				FROM filehash AS selectedMode
				WHERE selectedMode.mode = $1
				AND selectedMode.file = filehash.file
			)
	)
ORDER BY "file";
`

type ValidateRequest struct {
	Launcher string `json:"launcher" binding:"required"`
	Files    string `json:"files" binding:"required"`
}

func ValidateGet(gc *gin.Context) {

	var (
		verifyFileNames []string

		validateResponse *response.ValidateResponse

		pClaims, _ = gc.Get("claims")
		claims     = pClaims.(jwt.MapClaims)

		mode, _ = (claims["mode"]).(json.Number).Int64()
	)

	verifyFileNames = getFileForMode(mode, `"file"`)

	validateResponse = &response.ValidateResponse{
		DefaultResponse: response.DefaultResponse{
			Status: response.ResponseErrorSuccess,
		},
		Files: verifyFileNames,
	}

	gc.IndentedJSON(
		http.StatusOK,
		validateResponse,
	)
}

func ValidatePost(gc *gin.Context) {

	var (
		err error

		token        string
		allFilesHash string

		verifyFileHashes []string

		hasher hash.Hash

		validationRequest ValidateRequest

		pClaims, _ = gc.Get("claims")
		claims     = pClaims.(jwt.MapClaims)

		mode, _ = (claims["mode"]).(json.Number).Int64()
	)

	// get client response body
	err = gc.BindJSON(&validationRequest)
	if err != nil {
		fmt.Println("Could not parse request body as POST in ValidatePost")

		return
	}

	// get file hashes for mode
	verifyFileHashes = getFileForMode(mode, `"hash"`)

	hasher = sha1.New()
	for _, fileHash := range verifyFileHashes {
		hasher.Write([]byte(fileHash))
	}

	allFilesHash = fmt.Sprintf("%x", hasher.Sum(nil))

	if strings.Compare(allFilesHash, validationRequest.Files) != 0 {

		fmt.Printf(
			"File verification failed for account ID [%s] and mode [%s]\n",
			claims["account"],
			claims["mode"],
		)

		gc.IndentedJSON(
			http.StatusOK,
			response.CreateErrorResponse(response.ResponseErrorCorruptFiles),
		)

		return
	}

	// generate token
	token, err = utils.GenerateToken(
		&jwt.MapClaims{
			"account":  claims["account"],
			"mode":     claims["mode"],
			"verified": true,
		},
	)
	if err != nil {

		fmt.Printf("Token singing failed: %s\n", err.Error())

		gc.IndentedJSON(
			http.StatusOK,
			response.CreateErrorResponse(response.ResponseErrorInternalTokenCreationFailed),
		)

		return
	}

	gc.IndentedJSON(
		http.StatusOK,
		response.TokenResponse{
			DefaultResponse: response.DefaultResponse{
				Status: response.ResponseErrorSuccess,
			},
			Token: token,
		},
	)

	return
}

func getFileForMode(mode int64, column string) (fileNames []string) {

	var (
		err error

		rows pgx.Rows
	)

	rows, err = utils.GetPostgrePool().Query(
		context.Background(),
		fmt.Sprintf(fileHashQuery, column),
		mode,
	)
	if err != nil {
		return
	}

	fileNames, err = pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return
	}

	return
}
