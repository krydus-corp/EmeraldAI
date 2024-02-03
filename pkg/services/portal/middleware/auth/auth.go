package auth

import (
	"fmt"
	"net/http"

	jwtGo "github.com/golang-jwt/jwt"
	echo "github.com/labstack/echo/v4"
	echoMW "github.com/labstack/echo/v4/middleware"

	log "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	jwt "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/jwt"
)

// TokenService represents JWT token service interface
type TokenService interface {
	ValidateToken(*jwt.CustomClaims, bool) (models.User, error)
	ExpireAuth(string) (bool, error)
}

// Default middleware to check the token.
func JWT(secret string, apiKeySvc ApiKeyService) echo.MiddlewareFunc {
	config := echoMW.JWTConfig{
		Claims:     &jwt.CustomClaims{},
		SigningKey: []byte(secret),
		ErrorHandler: func(err error) error {
			return &echo.HTTPError{
				Code:    http.StatusUnauthorized,
				Message: "Not authorized",
			}
		},
		// Bypass JWT for API Key routes or if 'Authorization' key in query param
		Skipper: func(c echo.Context) bool {
			if apiKeySvc.IsWhitelistedRoute(c.Request().URL.EscapedPath()) && c.Request().Header.Get("apikey") != "" {
				c.Logger().Debug("Skipping JWT for whitelisted endpoint")
				return true
			}
			if tokenQueryParam := c.QueryParams().Get("Authorization"); tokenQueryParam != "" {
				c.Logger().Debug("Found 'Authorization' key in query param; skipping JWT header auth")
				return true
			}

			return false
		},
	}

	return echoMW.JWTWithConfig(config) //lint:ignore SA1019 - TODO: upgrade eventually
}

// Middleware for additional steps:
// 1. Check the user exists in DB
// 2. Check the token info exists in Redis
// 3. Add the user DB data to Context
// 4. Prolong the Redis TTL of the current token pair
func Middleware(secret string, tokenSvc TokenService, apiKeySvc ApiKeyService, enforcer *Enforcer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Bypass Authentication/Authorization for API Key routes
			if apiKeySvc.IsWhitelistedRoute(c.Request().URL.EscapedPath()) && c.Request().Header.Get("apikey") != "" {
				accessKey := c.Request().Header.Get("userid")
				secretKey := c.Request().Header.Get("apikey")
				if apiKeySvc.IsAuthorized(accessKey, secretKey) {
					return next(c)
				}
				return c.NoContent(http.StatusUnauthorized)
			}

			// ------------ Authentication ------------ //
			var token *jwtGo.Token
			var err error

			// Check if 'Authorization'key is passed as query param, else assume passed in header
			if tokenQueryParam := c.QueryParams().Get("Authorization"); tokenQueryParam != "" {
				token, err = jwtGo.ParseWithClaims(tokenQueryParam, &jwt.CustomClaims{}, func(t *jwtGo.Token) (interface{}, error) {
					return []byte(secret), nil
				})
				if err != nil {
					log.Debugf("error parsing query param jwt claims; err=%s", err.Error())
					return c.NoContent(http.StatusUnauthorized)
				}
			} else {
				token = c.Get("user").(*jwtGo.Token)
			}
			claims := token.Claims.(*jwt.CustomClaims)

			if err := claims.Valid(); err != nil {
				return c.NoContent(http.StatusUnauthorized)
			}

			user, err := tokenSvc.ValidateToken(claims, false)
			if err != nil {
				return c.NoContent(http.StatusUnauthorized)
			}

			c.Set("current_user", user)                     // Used to check current user
			c.Request().Header.Set("userid", user.ID.Hex()) // Used when proxying to other internal services

			go func() {
				tokenSvc.ExpireAuth(fmt.Sprintf("token-%s", claims.ID))
			}()

			// ------------ Authorization ------------ //
			method := c.Request().Method
			path := c.Request().URL.Path

			if enforcer != nil { // unit test workaround
				ok, err := enforcer.Enforcer.Enforce(user.Username, path, method)
				if err != nil || !ok {
					return c.NoContent(http.StatusUnauthorized)
				}
			}

			return next(c)
		}
	}
}
