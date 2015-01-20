package application

import (
	"log"
	"os"
	"path"
	"time"

	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/gobble"
	"github.com/cloudfoundry-incubator/notifications/mail"
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal"
	"github.com/cloudfoundry-incubator/notifications/postal/strategies"
	"github.com/cloudfoundry-incubator/notifications/web"
	"github.com/cloudfoundry-incubator/notifications/web/middleware"

	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-cf/uaa-sso-golang/uaa"
	"github.com/ryanmoran/stack"
)

type Mother struct {
	logger *log.Logger
	queue  *gobble.Queue
	env    Environment
}

func NewMother(env Environment) *Mother {
	return &Mother{
		env: env,
	}
}

func (mother *Mother) Logger() *log.Logger {
	if mother.logger == nil {
		mother.logger = log.New(os.Stdout, "[WEB] ", log.LstdFlags)
	}
	return mother.logger
}

func (mother *Mother) Queue() gobble.QueueInterface {
	if mother.queue == nil {
		mother.queue = gobble.NewQueue(gobble.Config{
			WaitMaxDuration: time.Duration(mother.env.GobbleWaitMaxDuration) * time.Millisecond,
		})
	}

	return mother.queue
}

func (mother Mother) UAAClient() uaa.UAA {
	env := mother.env
	uaaClient := uaa.NewUAA("", env.UAAHost, env.UAAClientID, env.UAAClientSecret, "")
	uaaClient.VerifySSL = env.VerifySSL

	return uaaClient
}

func (mother Mother) RouterConfig() web.RouterConfig {
	return web.RouterConfig{
		Database:      mother.Database(),
		Logging:       mother.Logging(),
		CORS:          mother.CORS(),
		Services:      mother.ServicesFactory(),
		Strategies:    mother.StrategyFactory(),
		Authenticator: mother.Authenticator,
	}
}

func (mother Mother) StrategyFactory() StrategyFactory {
	env := mother.env
	templatesLoader := mother.ServicesFactory().TemplatesLoader()
	mailer := mother.Mailer()
	logger := mother.Logger()

	cloudController := cf.NewCloudController(env.CCHost, !env.VerifySSL)
	uaaClient := mother.UAAClient()

	return NewStrategyFactory(uaaClient, cloudController, logger, mailer, templatesLoader)
}

func (mother Mother) ServicesFactory() ServicesFactory {
	return NewServicesFactory(mother.Database())
}

func (mother Mother) DeliveryWorker(id int) postal.DeliveryWorker {
	env := mother.env
	return postal.NewDeliveryWorker(id, mother.Logger(), mother.MailClient(), mother.Queue(),
		models.NewGlobalUnsubscribesRepo(), models.NewUnsubscribesRepo(), models.NewKindsRepo(), models.NewMessagesRepo(),
		mother.Database(), env.Sender, env.EncryptionKey)
}

func (mother Mother) Mailer() strategies.Mailer {
	return strategies.NewMailer(mother.Queue(), uuid.NewV4, models.NewMessagesRepo())
}

func (mother Mother) MailClient() *mail.Client {
	env := mother.env
	mailConfig := mail.Config{
		User:           env.SMTPUser,
		Pass:           env.SMTPPass,
		Host:           env.SMTPHost,
		Port:           env.SMTPPort,
		Secret:         env.SMTPCRAMMD5Secret,
		TestMode:       env.TestMode,
		SkipVerifySSL:  env.VerifySSL,
		DisableTLS:     !env.SMTPTLS,
		LoggingEnabled: env.SMTPLoggingEnabled,
	}

	switch env.SMTPAuthMechanism {
	case SMTPAuthNone:
		mailConfig.AuthMechanism = mail.AuthNone
	case SMTPAuthPlain:
		mailConfig.AuthMechanism = mail.AuthPlain
	case SMTPAuthCRAMMD5:
		mailConfig.AuthMechanism = mail.AuthCRAMMD5
	}

	client, err := mail.NewClient(mailConfig, mother.Logger())
	if err != nil {
		panic(err)
	}

	return client
}

func (mother Mother) Logging() stack.Middleware {
	return stack.NewLogging(mother.Logger())
}

func (mother Mother) Authenticator(scopes ...string) middleware.Authenticator {
	return middleware.NewAuthenticator(UAAPublicKey, scopes...)
}

func (mother Mother) Database() models.DatabaseInterface {
	env := mother.env
	return models.NewDatabase(models.Config{
		DatabaseURL:         env.DatabaseURL,
		MigrationsPath:      env.ModelMigrationsDir,
		DefaultTemplatePath: path.Join(env.RootPath, "templates", "default.json"),
	})
}

func (mother Mother) CORS() middleware.CORS {
	return middleware.NewCORS(mother.env.CORSOrigin)
}
