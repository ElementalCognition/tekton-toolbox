package githubtransport

import (
	"github.com/bradleyfalzon/ghinstallation"
	"net/http"
)

func NewTransport(appID, installationID int64, appKey string) (*ghinstallation.Transport, error) {
	atr, err := ghinstallation.NewAppsTransportKeyFromFile(http.DefaultTransport, appID, appKey)
	if err != nil {
		return nil, err
	}
	return ghinstallation.NewFromAppsTransport(atr, installationID), nil
}
