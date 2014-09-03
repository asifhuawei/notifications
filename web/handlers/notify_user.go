package handlers

import (
    "net/http"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/metrics"
    "github.com/cloudfoundry-incubator/notifications/models"
    "github.com/cloudfoundry-incubator/notifications/postal"
)

type NotifyUser struct {
    errorWriter ErrorWriterInterface
    notify      Notify
}

func NewNotifyUser(notify Notify, errorWriter ErrorWriterInterface) NotifyUser {
    return NotifyUser{
        errorWriter: errorWriter,
        notify:      notify,
    }
}

func (handler NotifyUser) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    metrics.NewMetric("counter", map[string]interface{}{
        "name": "notifications.web.users",
    }).Log()

    transaction := models.NewTransaction()
    err := handler.Execute(w, req, transaction)
    if err != nil {
        handler.errorWriter.Write(w, err)
        return
    }
}

func (handler NotifyUser) Execute(w http.ResponseWriter, req *http.Request, transaction models.TransactionInterface) error {
    spaceGUID := postal.UserGUID(strings.TrimPrefix(req.URL.Path, "/users/"))

    output, err := handler.notify.Execute(transaction, req, spaceGUID)
    if err != nil {
        return err
    }

    w.WriteHeader(http.StatusOK)
    w.Write(output)

    return nil
}
