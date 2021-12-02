package apis

import (
	"net/http"

	"code.cloudfoundry.org/cf-k8s-controllers/api/authorization"
	"github.com/go-http-utils/headers"
	"github.com/go-logr/logr"
)

type AuthenticationMiddleware struct {
	logger                   logr.Logger
	authInfoParser           AuthInfoParser
	identityProvider         IdentityProvider
	unauthenticatedEndpoints map[string]interface{}
}

//counterfeiter:generate -o fake -fake-name AuthInfoParser . AuthInfoParser

type AuthInfoParser interface {
	Parse(authHeader string) (authorization.Info, error)
}

func NewAuthenticationMiddleware(logger logr.Logger, authInfoParser AuthInfoParser, identityProvider IdentityProvider) *AuthenticationMiddleware {
	return &AuthenticationMiddleware{
		logger:           logger,
		authInfoParser:   authInfoParser,
		identityProvider: identityProvider,
		unauthenticatedEndpoints: map[string]interface{}{
			"/":   struct{}{},
			"/v3": struct{}{},
		},
	}
}

func (a *AuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, authNotRequired := a.unauthenticatedEndpoints[r.URL.Path]; authNotRequired {
			next.ServeHTTP(w, r)
			return
		}

		authInfo, err := a.authInfoParser.Parse(r.Header.Get(headers.Authorization))
		if err != nil {
			if authorization.IsNotAuthenticated(err) {
				writeNotAuthenticatedErrorResponse(w)
				return
			}

			if authorization.IsInvalidAuth(err) {
				writeInvalidAuthErrorResponse(w)
				return
			}

			a.logger.Error(err, "failed to parse auth info")
			writeUnknownErrorResponse(w)
			return
		}

		r = r.WithContext(authorization.NewContext(r.Context(), &authInfo))

		_, err = a.identityProvider.GetIdentity(r.Context(), authInfo)
		if err != nil {
			if authorization.IsInvalidAuth(err) {
				writeInvalidAuthErrorResponse(w)
				return
			}

			a.logger.Error(err, "failed to get identity")
			writeUnknownErrorResponse(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}