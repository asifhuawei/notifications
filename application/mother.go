package application

import (
	"log"
	"os"

	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/config"
	"github.com/cloudfoundry-incubator/notifications/gobble"
	"github.com/cloudfoundry-incubator/notifications/mail"
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal"
	"github.com/cloudfoundry-incubator/notifications/web/handlers"
	"github.com/cloudfoundry-incubator/notifications/web/middleware"
	"github.com/cloudfoundry-incubator/notifications/web/services"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-cf/uaa-sso-golang/uaa"
	"github.com/ryanmoran/stack"
)

type Mother struct {
	logger *log.Logger
	queue  *gobble.Queue
}

func NewMother() *Mother {
	return &Mother{}
}

func (mother *Mother) Logger() *log.Logger {
	if mother.logger == nil {
		mother.logger = log.New(os.Stdout, "[WEB] ", log.LstdFlags)
	}
	return mother.logger
}

func (mother *Mother) Queue() *gobble.Queue {
	if mother.queue == nil {
		mother.queue = gobble.NewQueue()
	}
	return mother.queue
}

func (mother Mother) NewUAARecipe() postal.UAARecipe {
	env := config.NewEnvironment()
	uaaClient := uaa.NewUAA("", env.UAAHost, env.UAAClientID, env.UAAClientSecret, "")
	uaaClient.VerifySSL = env.VerifySSL
	cloudController := cf.NewCloudController(env.CCHost, !env.VerifySSL)

	tokenLoader := postal.NewTokenLoader(&uaaClient)
	userLoader := postal.NewUserLoader(&uaaClient, mother.Logger(), cloudController)
	spaceLoader := postal.NewSpaceLoader(cloudController)
	templateLoader := postal.NewTemplateLoader(postal.NewFileSystem(), env.RootPath)
	mailer := mother.Mailer()
	receiptsRepo := models.NewReceiptsRepo()

	return postal.NewUAARecipe(tokenLoader, userLoader, spaceLoader, templateLoader, mailer, receiptsRepo)
}

func (mother Mother) EmailRecipe() postal.MailRecipeInterface {
	env := config.NewEnvironment()
	return postal.NewEmailRecipe(mother.Mailer(), postal.NewTemplateLoader(postal.NewFileSystem(), env.RootPath))
}

func (mother Mother) NotificationFinder() services.NotificationFinder {
	clientsRepo, kindsRepo := mother.Repos()
	return services.NewNotificationFinder(clientsRepo, kindsRepo, mother.Database())
}

func (mother Mother) Mailer() postal.Mailer {
	return postal.NewMailer(mother.Queue(), uuid.NewV4)
}

func (mother Mother) MailClient() *mail.Client {
	env := config.NewEnvironment()
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
	case config.SMTPAuthNone:
		mailConfig.AuthMechanism = mail.AuthNone
	case config.SMTPAuthPlain:
		mailConfig.AuthMechanism = mail.AuthPlain
	case config.SMTPAuthCRAMMD5:
		mailConfig.AuthMechanism = mail.AuthCRAMMD5
	}

	client, err := mail.NewClient(mailConfig, mother.Logger())
	if err != nil {
		panic(err)
	}

	return client
}

func (mother Mother) Repos() (models.ClientsRepo, models.KindsRepo) {
	return models.NewClientsRepo(), mother.KindsRepo()
}

func (mother Mother) Logging() stack.Middleware {
	return stack.NewLogging(mother.Logger())
}

func (mother Mother) ErrorWriter() handlers.ErrorWriter {
	return handlers.NewErrorWriter()
}

func (mother Mother) Authenticator(scopes ...string) middleware.Authenticator {
	return middleware.NewAuthenticator(config.UAAPublicKey, scopes...)
}

func (mother Mother) Registrar() services.Registrar {
	clientsRepo, kindsRepo := mother.Repos()
	return services.NewRegistrar(clientsRepo, kindsRepo)
}

func (mother Mother) Database() models.DatabaseInterface {
	env := config.NewEnvironment()
	return models.NewDatabase(env.DatabaseURL)
}

func (mother Mother) PreferencesFinder() *services.PreferencesFinder {
	return services.NewPreferencesFinder(models.NewPreferencesRepo(), mother.GlobalUnsubscribesRepo(), mother.Database())
}

func (mother Mother) PreferenceUpdater() services.PreferenceUpdater {
	return services.NewPreferenceUpdater(mother.GlobalUnsubscribesRepo(), mother.UnsubscribesRepo(), mother.KindsRepo())
}

func (mother Mother) KindsRepo() models.KindsRepo {
	return models.NewKindsRepo()
}

func (mother Mother) UnsubscribesRepo() models.UnsubscribesRepo {
	return models.NewUnsubscribesRepo()
}

func (mother Mother) GlobalUnsubscribesRepo() models.GlobalUnsubscribesRepo {
	return models.NewGlobalUnsubscribesRepo()
}

func (mother Mother) CORS() middleware.CORS {
	env := config.NewEnvironment()
	return middleware.NewCORS(env.CORSOrigin)
}
