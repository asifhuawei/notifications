package application

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/notifications/web"
	"github.com/pivotal-cf/uaa-sso-golang/uaa"
	"github.com/ryanmoran/viron"
)

const WorkerCount = 10

type Application struct {
	env      Environment
	mother   *Mother
	migrator Migrator
}

func NewApplication() Application {
	mother := NewMother()
	env := NewEnvironment()

	return Application{
		env:      env,
		mother:   mother,
		migrator: NewMigrator(mother, env.VCAPApplication.InstanceIndex == 0),
	}
}

func (app Application) Boot() {
	app.PrintConfiguration()
	app.ConfigureSMTP()
	app.RetrieveUAAPublicKey()
	app.migrator.Migrate()
	app.EnableDBLogging()
	app.UnlockJobs()
	app.StartWorkers()
	app.StartMessageGC()
	app.StartServer()
}

func (app Application) PrintConfiguration() {
	logger := app.mother.Logger()
	logger.Println("Booting with configuration:")

	viron.Print(app.env, logger)
}

func (app Application) ConfigureSMTP() {
	if app.env.TestMode {
		return
	}

	mailClient := app.mother.MailClient()
	err := mailClient.Connect()
	if err != nil {
		panic(err)
	}

	err = mailClient.Hello()
	if err != nil {
		panic(err)
	}

	startTLSSupported, _ := mailClient.Extension("STARTTLS")

	mailClient.Quit()

	if !startTLSSupported && app.env.SMTPTLS {
		panic(errors.New(`SMTP TLS configuration mismatch: Configured to use TLS over SMTP, but the mail server does not support the "STARTTLS" extension.`))
	}

	if startTLSSupported && !app.env.SMTPTLS {
		panic(errors.New(`SMTP TLS configuration mismatch: Not configured to use TLS over SMTP, but the mail server does support the "STARTTLS" extension.`))
	}
}

func (app Application) RetrieveUAAPublicKey() {
	auth := app.mother.UAAClient()

	key, err := uaa.GetTokenKey(auth)
	if err != nil {
		panic(err)
	}

	UAAPublicKey = key
	log.Printf("UAA Public Key: %s", UAAPublicKey)
}

func (app Application) UnlockJobs() {
	app.mother.Queue().Unlock()
}

func (app Application) EnableDBLogging() {
	if app.env.DBLoggingEnabled {
		app.mother.Database().TraceOn("[DB]", app.mother.Logger())
	}
}

func (app Application) StartWorkers() {
	for i := 0; i < WorkerCount; i++ {
		worker := app.mother.DeliveryWorker(i + 1)
		worker.Work()
	}
}

func (app Application) StartMessageGC() {
	messageLifetime := 24 * time.Hour
	db := app.mother.Database()
	messagesRepo := app.mother.MessagesRepo()
	pollingInterval := 1 * time.Hour
	logger := app.mother.Logger()
	messageGC := postal.NewMessageGC(messageLifetime, db, messagesRepo, pollingInterval, logger)
	messageGC.Run()
}

func (app Application) StartServer() {
	config := app.mother.RouterConfig()
	router := web.NewRouter(config)
	log.Printf("Listening on localhost:%s\n", app.env.Port)

	http.ListenAndServe(":"+app.env.Port, router.Routes())
}

// This is a hack to get the logs output to the loggregator before the process exits
func (app Application) Crash() {
	err := recover()
	if err != nil {
		time.Sleep(5 * time.Second)
		panic(err)
	}
}
