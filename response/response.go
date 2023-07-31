package response

// Status Codes
const (
	ResponseErrorGroupOK = iota * 100
	ResponseErrorGroupErrorLauncher
	ResponseErrorGroupErrorAccount
	ResponseErrorGroupErrorDB
	ResponseErrorGroupErrorInternal
)

//
// Error Codes
//

// No Error / Info
const (
	ResponseErrorSuccess = iota + ResponseErrorGroupOK
)

// Launcher Error
const (
	ResponseErrorUpdateLauncher = iota + ResponseErrorGroupErrorLauncher
	ResponseErrorCorruptLauncher
	ResponseErrorLauncherTokenExpired
	ResponseErrorCorruptFiles
	ResponseErrorLauncherNoLongerSupported
	ResponseErrorLauncherGameTokenRequestNotVerified
)

// Account Error
const (
	ResponseErrorUseStagingLoginToUpdatePassword = iota + ResponseErrorGroupErrorAccount
	ResponseErrorWrongUsernamePassword
	ResponseErrorAccountInactive
)

// DB Error
const (
	ResponseErrorDatabase = iota + ResponseErrorGroupErrorDB
)

// Internal Error
const (
	ResponseErrorInternalTokenCreationFailed = iota + ResponseErrorGroupErrorInternal
)

type DefaultResponse struct {
	Status int `json:"status"`
}

type ErrorResponse struct {
	DefaultResponse
	ErrorText string `json:"errorText"`
}

type TokenResponse struct {
	DefaultResponse
	Token string `json:"token"`
}

type VersionResponse struct {
	DefaultResponse
	ReleaseDate   int64  `json:"releaseDate"`
	VersionString string `json:"versionString"`
}

type ValidateResponse struct {
	DefaultResponse
	Files []string `json:"files"`
}

type GameTokenResponse struct {
	DefaultResponse
	GameToken string `json:"gameToken"`
}

func CreateErrorResponse(statusCode int) ErrorResponse {
	return CreateErrorResponseWithText(statusCode, "")
}

func CreateErrorResponseWithText(statusCode int, errorText string) (errorResponse ErrorResponse) {
	return ErrorResponse{
		DefaultResponse: DefaultResponse{
			Status: statusCode,
		},
		ErrorText: errorText,
	}
}
