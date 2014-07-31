package postal

import (
    "encoding/json"
    "net/http"

    "github.com/cloudfoundry-incubator/notifications/config"
    "github.com/dgrijalva/jwt-go"
    "github.com/nu7hatch/gouuid"
    "github.com/pivotal-cf/uaa-sso-golang/uaa"
)

type NotifyResponse []map[string]string
type NotifyFailureResponse map[string][]string
type NotificationType int

const (
    StatusNotFound  = "notfound"
    StatusNoAddress = "noaddress"

    IsSpace NotificationType = iota
    IsUser
)

type GUIDGenerationFunc func() (*uuid.UUID, error)

type UAAInterface interface {
    uaa.GetClientTokenInterface
    uaa.SetTokenInterface
    uaa.UsersByIDsInterface
}

type Options struct {
    ReplyTo           string
    Subject           string
    KindDescription   string
    SourceDescription string
    Text              string
    HTML              string
    Kind              string
}

type Courier struct {
    uaaClient UAAInterface

    userLoader     UserLoader
    spaceLoader    SpaceLoader
    templateLoader TemplateLoader
    mailer         Mailer
}

func NewCourier(uaaClient UAAInterface, userLoader UserLoader, spaceLoader SpaceLoader, templateLoader TemplateLoader, mailer Mailer) Courier {
    return Courier{
        uaaClient:      uaaClient,
        userLoader:     userLoader,
        spaceLoader:    spaceLoader,
        templateLoader: templateLoader,
        mailer:         mailer,
    }
}

func Error(w http.ResponseWriter, code int, errors []string) {
    response, err := json.Marshal(NotifyFailureResponse{
        "errors": errors,
    })
    if err != nil {
        panic(err)
    }

    w.WriteHeader(code)
    w.Write(response)
}

func (courier Courier) Dispatch(w http.ResponseWriter, rawToken, guid string, notificationType NotificationType, options Options) error {
    token, err := courier.uaaClient.GetClientToken()
    if err != nil {
        panic(err)
    }
    courier.uaaClient.SetToken(token.Access)

    users, err := courier.userLoader.Load(notificationType, guid, token.Access)
    if err != nil {
        return err
    }

    space, organization, err := courier.spaceLoader.Load(guid, token.Access, notificationType)
    if err != nil {
        return CCDownError("Cloud Controller is unavailable")
    }

    clientToken, _ := jwt.Parse(rawToken, func(t *jwt.Token) ([]byte, error) {
        return []byte(config.UAAPublicKey), nil
    })
    clientID := clientToken.Claims["client_id"].(string)

    ////////////////////////////////////////////////////////////////////////////////////////////////////
    templates, err := courier.templateLoader.Load(options.Subject, notificationType)
    if err != nil {
        return TemplateLoadError("An email template could not be loaded")
    }

    messages := courier.mailer.Deliver(templates, users, options, space, organization, clientID)

    responseBytes, err := json.Marshal(messages)
    if err != nil {
        panic(err)
    }
    w.WriteHeader(http.StatusOK)
    w.Write(responseBytes)
    ////////////////////////////////////////////////////////////////////////////////////////////////////

    return nil
}
