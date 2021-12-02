package apis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/cf-k8s-controllers/api/authorization"
	"code.cloudfoundry.org/cf-k8s-controllers/api/presenter"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	WhoAmIEndpoint = "/whoami"
)

//counterfeiter:generate -o fake -fake-name IdentityProvider . IdentityProvider

type IdentityProvider interface {
	GetIdentity(context.Context, authorization.Info) (authorization.Identity, error)
}

type WhoAmIHandler struct {
	identityProvider IdentityProvider
	logger           logr.Logger
	apiBaseURL       url.URL
}

func NewWhoAmI(identityProvider IdentityProvider, apiBaseURL url.URL) *WhoAmIHandler {
	return &WhoAmIHandler{
		identityProvider: identityProvider,
		apiBaseURL:       apiBaseURL,
		logger:           controllerruntime.Log.WithName("Org Handler"),
	}
}

func (h *WhoAmIHandler) whoAmIHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	authInfo, ok := authorization.InfoFromContext(r.Context())
	if !ok {
		writeUnknownErrorResponse(w)
		return
	}

	identity, err := h.identityProvider.GetIdentity(ctx, authInfo)
	if err != nil {
		writeUnknownErrorResponse(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	identityResponse := presenter.ForWhoAmI(identity)

	err = json.NewEncoder(w).Encode(identityResponse)
	if err != nil {
		h.logger.Error(err, "Failed to write response")
	}
}

func (h *WhoAmIHandler) RegisterRoutes(router *mux.Router) {
	router.Path(WhoAmIEndpoint).Methods("GET").HandlerFunc(h.whoAmIHandler)
}