package handlers

import (
    "encoding/json"
    "log"
    "net/http"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/cf"
    "github.com/cloudfoundry-incubator/notifications/mail"
    "github.com/cloudfoundry-incubator/notifications/postal"
)

type NotifySpace struct {
    cloudController cf.CloudControllerInterface
    helper          postal.NotifyHelper
}

func NewNotifySpace(logger *log.Logger, cloudController cf.CloudControllerInterface,
    uaaClient postal.UAAInterface, mailClient mail.ClientInterface, guidGenerator postal.GUIDGenerationFunc) NotifySpace {
    return NotifySpace{
        cloudController: cloudController,
        helper:          postal.NewNotifyHelper(cloudController, logger, uaaClient, guidGenerator, mailClient),
    }
}

func Error(w http.ResponseWriter, code int, errors []string) {
    response, err := json.Marshal(postal.NotifyFailureResponse{
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

    handler.helper.Execute(w, req, spaceGUID, loadUsers, postal.IsSpace, params.ToOptions())
}
