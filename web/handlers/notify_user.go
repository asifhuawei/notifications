package handlers

import (
    "net/http"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/postal"
)

type NotifyUser struct {
    courier postal.Courier
}

func NewNotifyUser(courier postal.Courier) NotifyUser {
    return NotifyUser{
        courier: courier,
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

    handler.courier.Dispatch(w, req, userGUID, postal.IsUser, params.ToOptions())
}
