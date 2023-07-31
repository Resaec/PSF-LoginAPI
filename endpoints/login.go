package endpoints

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"PSF-LoginAPI/response"
	"PSF-LoginAPI/utils"
)

type LoginRequest struct {
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password" binding:"required"`
	LauncherHash string `json:"launcher" binding:"required"`
	Mode         int64  `json:"mode"`
}

type LauncherState struct {
	Version string `db:"version"`
	Active  bool   `db:"active"`
}

type Account struct {
	ID           int64  `db:"id"`
	Username     string `db:"username"`
	Password     string `db:"password"`
	PasswordHash string `db:"passhash"`
	Inactive     bool   `db:"inactive"`
}

// getAccount function with constant time enforcement
var fConstantTimeGetAccount = utils.ConstantTimeCall(utils.ConstantTime, getAccount)

func Login(gc *gin.Context) {

	var (
		err error

		statusCode int

		launcherVersionFromHash string
		token                   string

		loginRequest LoginRequest
		account      *Account
	)

	// bind json
	err = gc.BindJSON(&loginRequest)
	if err != nil {
		fmt.Println("Could not parse request body as POST Login")

		return
	}

	// get account in constant time
	statusCode, account = fConstantTimeGetAccount(&loginRequest)
	if statusCode != response.ResponseErrorSuccess {

		gc.IndentedJSON(
			http.StatusOK,
			response.CreateErrorResponse(statusCode),
		)

		return
	}

	// check launcher hash
	statusCode, launcherVersionFromHash = getLauncherVersionFromHash(&loginRequest)
	if statusCode != response.ResponseErrorSuccess {

		gc.IndentedJSON(
			http.StatusOK,
			response.CreateErrorResponse(statusCode),
		)

		return
	}

	fmt.Printf(
		"User [%s] with ID %d is logging in for mode %d with launcher version %s (%s)\n",
		loginRequest.Username,
		account.ID,
		loginRequest.Mode,
		launcherVersionFromHash,
		loginRequest.LauncherHash,
	)

	// generate token
	token, err = utils.GenerateToken(
		&jwt.MapClaims{
			"account": account.ID,
			"mode":    loginRequest.Mode,
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

// returns true if there are active launchers
func getLaunchersActive() (hasActiveLaunchers bool) {

	var (
		err error

		rows pgx.Rows
	)

	rows, err = utils.GetPostgrePool().Query(
		context.Background(),
		`SELECT TRUE FROM "launcher" WHERE "active" = TRUE LIMIT 1`,
	)
	if err != nil {
		fmt.Printf("Error getting active launchers from DB: %s\n", err.Error())
		return
	}

	hasActiveLaunchers, err = pgx.CollectOneRow(rows, pgx.RowTo[bool])
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		fmt.Printf("Error parsing active launchers from DB: %s\n", err.Error())
		return
	}

	return
}

func getLauncherVersionFromHash(loginRequest *LoginRequest) (statusCode int, version string) {

	var (
		err error

		rows pgx.Rows

		launcherState *LauncherState
	)

	version = "UNK"

	// if there are no active launchers at all, just continue
	if getLaunchersActive() == false {
		return
	}

	rows, err = utils.GetPostgrePool().Query(
		context.Background(),
		`SELECT "version", "active" FROM "launcher" WHERE "hash" = $1`,
		loginRequest.LauncherHash,
	)
	if err != nil {
		statusCode = response.ResponseErrorDatabase

		fmt.Printf("Error querying launcher version from DB: %s\n", err.Error())

		return
	}

	launcherState, err = pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[LauncherState])
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		statusCode = response.ResponseErrorDatabase

		fmt.Printf("Error parsing launcher version from DB: %s\n", err.Error())

		return
	}

	// no launchers with that hash found
	if errors.Is(err, pgx.ErrNoRows) {
		statusCode = response.ResponseErrorCorruptLauncher

		fmt.Printf(
			"User [%s] uses launcher with unknown hash: %s\n",
			loginRequest.Username,
			loginRequest.LauncherHash,
		)

		return
	}

	// launcher found, check active
	if launcherState.Active == false {
		statusCode = response.ResponseErrorLauncherNoLongerSupported

		return
	}

	return
}

func getAccount(loginRequest *LoginRequest) (statusCode int, account *Account) {

	var (
		err error

		rows pgx.Rows
	)

	rows, err = utils.GetPostgrePool().Query(
		context.Background(),
		`SELECT "id", "username", "password", "passhash", "inactive" FROM "account" WHERE "username" = $1`,
		loginRequest.Username,
	)
	if err != nil {
		statusCode = response.ResponseErrorDatabase

		fmt.Printf("Error getting account from DB: %s\n", err.Error())

		return
	}

	account, err = pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[Account])
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		statusCode = response.ResponseErrorDatabase

		fmt.Printf("Error getting account from DB: %s\n", err.Error())

		return
	}

	// account not found
	if account == nil {
		statusCode = response.ResponseErrorWrongUsernamePassword

		fmt.Printf("Requested account not in DB: %s\n", loginRequest.Username)

		return
	}

	// this leaks usernames
	// if password field is empty the player has not yet logged in via StagingTest since the change
	if account.Password == "" {
		statusCode = response.ResponseErrorUseStagingLoginToUpdatePassword

		fmt.Printf(
			"User [%s] with ID %d needs to connect via StagingTest to update password\n",
			account.Username,
			account.ID,
		)
		return
	}

	// check password
	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(loginRequest.Password))
	if err != nil {
		statusCode = response.ResponseErrorWrongUsernamePassword

		fmt.Printf("Login as User [%s] with ID %d failed password check\n", account.Username, account.ID)
		return
	}

	// check account inactive
	if account.Inactive {
		statusCode = response.ResponseErrorAccountInactive

		fmt.Printf("User [%s] with ID %d tried to login to an inactive account\n", account.Username, account.ID)
		return
	}

	return
}
