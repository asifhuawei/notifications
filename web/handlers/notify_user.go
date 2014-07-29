package handlers

import (
    "log"
    "net/http"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/cf"
    "github.com/cloudfoundry-incubator/notifications/mail"
    "github.com/cloudfoundry-incubator/notifications/notifier"
)

type NotifyUser struct {
    logger        *log.Logger
    mailClient    mail.ClientInterface
    uaaClient     notifier.UAAInterface
    guidGenerator notifier.GUIDGenerationFunc
    helper        notifier.NotifyHelper
}

func NewNotifyUser(logger *log.Logger, mailClient mail.ClientInterface, uaaClient notifier.UAAInterface, guidGenerator notifier.GUIDGenerationFunc) NotifyUser {
    return NotifyUser{
        logger:        logger,
        mailClient:    mailClient,
        uaaClient:     uaaClient,
        guidGenerator: guidGenerator,
        helper:        notifier.NewNotifyHelper(cf.CloudController{}, logger, uaaClient, guidGenerator, mailClient),
    }
}

func (handler NotifyUser) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    userGUID := strings.TrimPrefix(req.URL.Path, "/users/")

    params, err := NewNotifyParams(req.Body)
    if err != nil {
        Error(w, 422, []string{"Request body could not be parsed"})
        return
    }

    if !params.Validate() {
        Error(w, 422, params.Errors)
        return
    }

    loadUsers := func(userGuid, accessToken string) ([]cf.CloudControllerUser, error) {
        return []cf.CloudControllerUser{{Guid: userGuid}}, nil
    }

    loadSpaceAndOrganization := false
    handler.helper.NotifyServeHTTP(w, req, userGUID, loadUsers, loadSpaceAndOrganization, params.ToOptions())
}
