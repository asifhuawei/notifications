package postal

import (
    "encoding/json"
    "log"
    "net/http"

    "github.com/cloudfoundry-incubator/notifications/cf"
    "github.com/cloudfoundry-incubator/notifications/config"
    "github.com/cloudfoundry-incubator/notifications/mail"
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
    cloudController cf.CloudControllerInterface
    logger          *log.Logger
    uaaClient       UAAInterface
    guidGenerator   GUIDGenerationFunc
    mailClient      mail.ClientInterface
}

func NewCourier(logger *log.Logger, cloudController cf.CloudControllerInterface,
    uaaClient UAAInterface, mailClient mail.ClientInterface,
    guidGenerator GUIDGenerationFunc) Courier {
    return Courier{
        cloudController: cloudController,
        logger:          logger,
        uaaClient:       uaaClient,
        guidGenerator:   guidGenerator,
        mailClient:      mailClient,
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

func (courier Courier) Dispatch(w http.ResponseWriter, rawToken,
    guid string, notificationType NotificationType, options Options) error {

    token, err := courier.uaaClient.GetClientToken()
    if err != nil {
        panic(err)
    }
    courier.uaaClient.SetToken(token.Access)

    userLoader := NewUserLoader(courier.uaaClient, courier.logger, courier.cloudController)
    users, err := userLoader.Load(notificationType, guid, token.Access)
    if err != nil {
        return err
    }

    spaceLoader := NewSpaceLoader(courier.cloudController)
    space, organization, err := spaceLoader.Load(guid, token.Access, notificationType)
    if err != nil {
        return CCDownError("Cloud Controller is unavailable")
    }

    clientToken, _ := jwt.Parse(rawToken, func(t *jwt.Token) ([]byte, error) {
        return []byte(config.UAAPublicKey), nil
    })
    clientID := clientToken.Claims["client_id"].(string)

    ////////////////////////////////////////////////////////////////////////////////////////////////////
    env := config.NewEnvironment()
    messages := NotifyResponse{}

    subjectTemplate, err := courier.LoadSubjectTemplate(options.Subject, NewTemplateManager())
    if err != nil {
        Error(w, http.StatusInternalServerError, []string{"An email template could not be loaded"})
        return nil
    }

    plainTextTemplate, htmlTemplate, err := courier.LoadBodyTemplates(notificationType, NewTemplateManager())
    if err != nil {
        Error(w, http.StatusInternalServerError, []string{"An email template could not be loaded"})
        return nil
    }

    for userGUID, uaaUser := range users {
        if len(uaaUser.Emails) > 0 {
            context := NewMessageContext(uaaUser, options, env, space, organization,
                clientID, courier.guidGenerator, plainTextTemplate, htmlTemplate, subjectTemplate)

            emailStatus := courier.SendMailToUser(context, courier.logger, courier.mailClient)
            courier.logger.Println(emailStatus)

            mailInfo := make(map[string]string)
            mailInfo["status"] = emailStatus
            mailInfo["recipient"] = uaaUser.ID
            mailInfo["notification_id"] = context.MessageID

            messages = append(messages, mailInfo)
        } else {
            var status string
            if uaaUser.ID == "" {
                status = StatusNotFound
            } else {
                status = StatusNoAddress
            }
            mailInfo := make(map[string]string)
            mailInfo["status"] = status
            mailInfo["recipient"] = userGUID
            mailInfo["notification_id"] = ""

            messages = append(messages, mailInfo)
        }
    }

    responseBytes, err := json.Marshal(messages)
    if err != nil {
        panic(err)
    }
    w.WriteHeader(http.StatusOK)
    w.Write(responseBytes)
    ////////////////////////////////////////////////////////////////////////////////////////////////////

    return nil
}

func (courier Courier) SendMailToUser(context MessageContext, logger *log.Logger,
    mailClient mail.ClientInterface) string {

    logger.Printf("Sending email to %s", context.To)
    status, message, err := SendMail(mailClient, context)
    if err != nil {
        panic(err)
    }

    logger.Print(message.Data())
    return status
}

func (courier Courier) LoadSubjectTemplate(subject string, templateManager EmailTemplateManager) (string, error) {
    var templateToLoad string
    if subject == "" {
        templateToLoad = "subject.missing"
    } else {
        templateToLoad = "subject.provided"
    }

    subjectTemplate, err := templateManager.LoadEmailTemplate(templateToLoad)
    if err != nil {
        return "", err
    }

    return subjectTemplate, nil
}

func (courier Courier) LoadBodyTemplates(notificationType NotificationType, templateManager EmailTemplateManager) (string, string, error) {
    var plainTextTemplate, htmlTemplate string
    var plainErr, htmlErr error

    if notificationType == IsSpace {
        plainTextTemplate, plainErr = templateManager.LoadEmailTemplate("space_body.text")
        htmlTemplate, htmlErr = templateManager.LoadEmailTemplate("space_body.html")
    } else {
        plainTextTemplate, plainErr = templateManager.LoadEmailTemplate("user_body.text")
        htmlTemplate, htmlErr = templateManager.LoadEmailTemplate("user_body.html")
    }

    if plainErr != nil {
        return "", "", plainErr
    }

    if htmlErr != nil {
        return "", "", htmlErr
    }

    return plainTextTemplate, htmlTemplate, nil
}
