package endpoints

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"PSF-LoginAPI/response"
	"PSF-LoginAPI/utils"
)

type LauncherInfo struct {
	Version    string    `db:"version"`
	ReleasedAt time.Time `db:"released_at"`
}

func Version(gc *gin.Context) {

	var (
		statusCode int

		matches []string

		launcherInfo *LauncherInfo

		request = gc.Request
	)

	matches = utils.ExtractLauncherVersion(request)
	if len(matches) == 0 {
		println("Not PSF Launcher:", request.UserAgent())

		gc.AbortWithStatus(http.StatusForbidden)
		return
	}

	statusCode, launcherInfo = getLastestLauncherVersion()
	if statusCode != response.ResponseErrorSuccess {

		gc.IndentedJSON(
			http.StatusOK,
			response.CreateErrorResponse(statusCode),
		)

		return
	}

	gc.IndentedJSON(
		http.StatusOK,
		response.VersionResponse{
			DefaultResponse: response.DefaultResponse{
				Status: response.ResponseErrorSuccess,
			},
			ReleaseDate:   launcherInfo.ReleasedAt.Unix(),
			VersionString: launcherInfo.Version,
		},
	)
}

func getLastestLauncherVersion() (statusCode int, launcherInfo *LauncherInfo) {

	var (
		err error

		rows pgx.Rows
	)

	rows, err = utils.GetPostgrePool().Query(
		context.Background(),
		`SELECT "version", "released_at" FROM "launcher" WHERE "active" = TRUE ORDER BY "released_at" DESC LIMIT 1`,
	)
	if err != nil {
		statusCode = response.ResponseErrorDatabase

		fmt.Printf("Error querying launcher information from DB: %s\n", err.Error())

		return
	}

	launcherInfo, err = pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[LauncherInfo])
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		statusCode = response.ResponseErrorDatabase

		fmt.Printf("Error parsing launcher information from DB: %s\n", err.Error())

		return
	}

	// if there are no active launchers, send a fake one
	if errors.Is(err, pgx.ErrNoRows) {
		statusCode = response.ResponseErrorSuccess

		launcherInfo = &LauncherInfo{
			Version:    "0.0.0.0",
			ReleasedAt: time.Time{},
		}
	}

	return
}
