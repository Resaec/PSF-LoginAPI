package utils

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ConstantTime = 1 * time.Second

var versionRegex *regexp.Regexp
var jwtSigningKey []byte

var pgConnectionURI string

func getPostgresConnectionURI() string {

	const PGConnectionURI = "postgres://%s:%s@%s:%s/%s"

	if pgConnectionURI == "" {
		pgConnectionURI = fmt.Sprintf(
			PGConnectionURI,
			os.Getenv("PGUSER"),
			os.Getenv("PGPASS"),
			os.Getenv("PGHOST"),
			os.Getenv("PGPORT"),
			os.Getenv("PGDB"),
		)
	}

	return pgConnectionURI
}

var pgxPool *pgxpool.Pool

func GetPostgrePool() *pgxpool.Pool {

	var (
		err error
	)

	// create pool if it does not exist yet
	if pgxPool == nil {

		var (
			connConfig *pgxpool.Config
		)

		connConfig, err = pgxpool.ParseConfig(getPostgresConnectionURI())
		// connConfig.MinConns = 1
		// connConfig.MaxConns = 4
		// connConfig.ConnConfig.ConnectTimeout = 10 * time.Second
		if err != nil {
			log.Fatalf("Failed to parse pgxpool config: %s", err.Error())
		}

		pgxPool, err = pgxpool.NewWithConfig(
			context.Background(),
			connConfig,
		)
		if err != nil {
			log.Fatalf("Could not connect to database: %s", err.Error())
		}
	}

	return pgxPool
}

func getJwtSigningKey() []byte {

	if len(jwtSigningKey) == 0 {
		jwtSigningKey = []byte(os.Getenv("JWT_KEY"))
	}

	return jwtSigningKey

}

func getLauncherVersionRegex() *regexp.Regexp {

	if versionRegex == nil {
		versionRegex = regexp.MustCompile(`^PSF Launcher v((\d+\.){3}\d+)$`)
	}

	return versionRegex
}

func ExtractLauncherVersion(request *http.Request) []string {

	return getLauncherVersionRegex().FindStringSubmatch(request.UserAgent())
}

// https://stackoverflow.com/a/71235876
// Creates wrapper function and sets it to the passed pointer to function
func ConstantTimeCall[T any](constantTime time.Duration, function T) T {

	// call function
	v := reflect.MakeFunc(
		reflect.TypeOf(function), func(in []reflect.Value) (values []reflect.Value) {

			// get the time to enforce constant time
			var (
				remainingTime time.Duration

				timestamp = time.Now()
			)

			// reconstruct and call original function
			f := reflect.ValueOf(function)
			values = f.Call(in)

			// calc remaining constant time
			remainingTime = constantTime - time.Now().Sub(timestamp)

			// sleep to achieve constant time
			time.Sleep(remainingTime)

			return values
		},
	)

	return v.Interface().(T)
}

type CustomClaims struct {
	jwt.RegisteredClaims
	UserID int64 `json:"userid"`
	Mode   int64 `json:"mode"`
}

func GenerateToken(additionalClaims *jwt.MapClaims) (string, error) {

	var (
		timeNow = time.Now()

		claims = jwt.MapClaims{
			"iss": "Launcher Auth API",
			"iat": timeNow.Unix(),
			"nbf": timeNow.Unix(),
			"exp": timeNow.Add(10 * time.Minute).Unix(),
		}
	)

	if additionalClaims != nil {
		for k, v := range *additionalClaims {
			claims[k] = v
		}
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(getJwtSigningKey())
}

func ParseToken(token string) (decodedToken *jwt.Token, claims *jwt.MapClaims, err error) {

	claims = &jwt.MapClaims{}
	decodedToken, err = jwt.ParseWithClaims(
		token,
		claims,
		func(token *jwt.Token) (interface{}, error) {

			// check claimed signing method
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("wrong token signing method: %s", token.Method.Alg())
			}

			// return token key
			return getJwtSigningKey(), nil
		},
		jwt.WithJSONNumber(),
	)

	return
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
