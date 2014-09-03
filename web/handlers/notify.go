package handlers

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/config"
    "github.com/cloudfoundry-incubator/notifications/models"
    "github.com/cloudfoundry-incubator/notifications/postal"
    "github.com/cloudfoundry-incubator/notifications/web/params"
    "github.com/cloudfoundry-incubator/notifications/web/services"
    "github.com/dgrijalva/jwt-go"
)

type Notify struct {
    courier   postal.CourierInterface
    finder    services.NotificationFinderInterface
    registrar services.RegistrarInterface
}

func NewNotify(courier postal.CourierInterface, finder services.NotificationFinderInterface, registrar services.RegistrarInterface) Notify {
    return Notify{
        courier:   courier,
        finder:    finder,
        registrar: registrar,
    }
}

func (handler Notify) Execute(transaction models.TransactionInterface, req *http.Request, guid postal.TypedGUID) ([]byte, error) {
    parameters, err := params.NewNotify(req.Body)
    if err != nil {
        return []byte{}, err
    }

    if !parameters.Validate() {
        return []byte{}, params.ValidationError(parameters.Errors)
    }

    clientID := handler.ParseClientID(strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer "))
    client, kind, err := handler.finder.ClientAndKind(clientID, parameters.KindID)
    if err != nil {
        return []byte{}, err
    }

    transaction.Begin()
    err = handler.registrar.Register(transaction, client, []models.Kind{kind})
    if err != nil {
        transaction.Rollback()
        return []byte{}, err
    }
    transaction.Commit()

    responses, err := handler.courier.Dispatch(clientID, guid, parameters.ToOptions(client, kind), transaction)
    if err != nil {
        return []byte{}, err
    }

    output, err := json.Marshal(responses)
    if err != nil {
        panic(err)
    }

    return output, nil
}

func (handler Notify) ParseClientID(rawToken string) string {
    token, _ := jwt.Parse(rawToken, func(token *jwt.Token) ([]byte, error) {
        return []byte(config.UAAPublicKey), nil
    })
    return token.Claims["client_id"].(string)
}
