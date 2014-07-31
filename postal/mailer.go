package postal

import (
    "log"

    "github.com/cloudfoundry-incubator/notifications/config"
    "github.com/cloudfoundry-incubator/notifications/mail"
    "github.com/pivotal-cf/uaa-sso-golang/uaa"
)

type Mailer struct {
    guidGenerator GUIDGenerationFunc
    templates     Templates
    logger        *log.Logger
    mailClient    mail.ClientInterface
}

func NewMailer(templates Templates, guidGenerator GUIDGenerationFunc, logger *log.Logger, mailClient mail.ClientInterface) Mailer {
    return Mailer{
        guidGenerator: guidGenerator,
        templates:     templates,
        logger:        logger,
        mailClient:    mailClient,
    }
}

func (mailer Mailer) Deliver(users map[string]uaa.User, options Options, space, organization, clientID string) NotifyResponse {
    env := config.NewEnvironment()
    messages := NotifyResponse{}
    for userGUID, uaaUser := range users {
        if len(uaaUser.Emails) > 0 {
            context := NewMessageContext(uaaUser, options, env, space, organization,
                clientID, mailer.guidGenerator, mailer.templates.Text, mailer.templates.HTML, mailer.templates.Subject)

            emailStatus := mailer.SendMailToUser(context, mailer.logger, mailer.mailClient)
            mailer.logger.Println(emailStatus)

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
    return messages
}

func (mailer Mailer) SendMailToUser(context MessageContext, logger *log.Logger, mailClient mail.ClientInterface) string {
    logger.Printf("Sending email to %s", context.To)
    status, message, err := SendMail(mailClient, context)
    if err != nil {
        panic(err)
    }

    logger.Print(message.Data())
    return status
}
