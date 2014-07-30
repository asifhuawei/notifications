package handlers

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/cf"
    "github.com/cloudfoundry-incubator/notifications/postal"
)

type NotifySpace struct {
    cloudController cf.CloudControllerInterface
    courier         postal.Courier
}

func NewNotifySpace(cloudController cf.CloudControllerInterface, courier postal.Courier) NotifySpace {
    return NotifySpace{
        cloudController: cloudController,
        courier:         courier,
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
    rawToken := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")

    err = handler.courier.Dispatch(w, rawToken, spaceGUID, postal.IsSpace, params.ToOptions())
    if err != nil {
        switch err.(type) {
        case postal.CCDownError:
            Error(w, http.StatusBadGateway, []string{"Cloud Controller is unavailable"})
        }
    }
}
