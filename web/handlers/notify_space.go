package handlers

import (
    "encoding/json"
    "log"
    "net/http"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/cf"
    "github.com/cloudfoundry-incubator/notifications/mail"
    "github.com/cloudfoundry-incubator/notifications/notifier"
)

type NotifySpace struct {
    logger          *log.Logger
    cloudController cf.CloudControllerInterface
    uaaClient       notifier.UAAInterface
    mailClient      mail.ClientInterface
    guidGenerator   notifier.GUIDGenerationFunc
    helper          notifier.NotifyHelper
}

func NewNotifySpace(logger *log.Logger, cloudController cf.CloudControllerInterface,
    uaaClient notifier.UAAInterface, mailClient mail.ClientInterface, guidGenerator notifier.GUIDGenerationFunc) NotifySpace {
    return NotifySpace{
        logger:          logger,
        cloudController: cloudController,
        uaaClient:       uaaClient,
        mailClient:      mailClient,
        guidGenerator:   guidGenerator,
        helper:          notifier.NewNotifyHelper(cloudController, logger, uaaClient, guidGenerator, mailClient),
    }
}

func Error(w http.ResponseWriter, code int, errors []string) {
    response, err := json.Marshal(notifier.NotifyFailureResponse{
        "errors": errors,
    })
    if err != nil {
        panic(err)
    }

    w.WriteHeader(code)
    w.Write(response)
}

func (handler NotifySpace) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    params, err := NewNotifyParams(req.Body)
    if err != nil {
        Error(w, 422, []string{"Request body could not be parsed"})
        return
    }

    if !params.Validate() {
        Error(w, 422, params.Errors)
        return
    }

    spaceGUID := strings.TrimPrefix(req.URL.Path, "/spaces/")

    loadUsers := func(spaceGuid, accessToken string) ([]cf.CloudControllerUser, error) {
        return handler.cloudController.GetUsersBySpaceGuid(spaceGuid, accessToken)
    }

    isSpace := true
    handler.helper.NotifyServeHTTP(w, req, spaceGUID, loadUsers, isSpace, params.ToOptions())
}
