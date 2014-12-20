package application

import (
    "errors"
    "log"
    "reflect"
    "time"

    "github.com/cloudfoundry-incubator/notifications/config"
    "github.com/cloudfoundry-incubator/notifications/gobble"
    "github.com/cloudfoundry-incubator/notifications/postal"
    "github.com/cloudfoundry-incubator/notifications/web"
    "github.com/pivotal-cf/uaa-sso-golang/uaa"
)

const WorkerCount = 10

type Application struct {
    env    config.Environment
    mother *Mother
}

func NewApplication() Application {
    return Application{
        env:    config.NewEnvironment(),
        mother: NewMother(),
    }
}

func (app Application) PrintConfiguration() {
    logger := app.mother.Logger()
    logger.Println("Booting with configuration:")

    t := reflect.TypeOf(app.env)
    v := reflect.ValueOf(app.env)
    for i := 0; i < v.NumField(); i++ {
        fieldType := t.Field(i)
        fieldValue := v.Field(i)

        if fieldType.Name == "EncryptionKey" {
            fieldValue = reflect.ValueOf("****************")
        }

        logger.Printf("  %-18s -> %+v", fieldType.Name, fieldValue.Interface())
    }
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
    auth := uaa.NewUAA("", app.env.UAAHost, app.env.UAAClientID, app.env.UAAClientSecret, "")
    auth.VerifySSL = app.env.VerifySSL

    key, err := uaa.GetTokenKey(auth)
    if err != nil {
        panic(err)
    }

    config.UAAPublicKey = key
    log.Printf("UAA Public Key: %s", config.UAAPublicKey)
}

func (app Application) Migrate() {
    if app.env.VCAPApplication.InstanceIndex == 0 {
		app.mother.Database().Migrate()
		gobble.Database().Migrate()
	}
}

func (app Application) EnableDBLogging() {
    if app.env.DBLoggingEnabled {
        app.mother.Database().TraceOn("[DB]", app.mother.Logger())
    }
}

func (app Application) UnlockJobs() {
    if app.env.VCAPApplication.InstanceIndex == 0 {
        gobble.NewQueue().Unlock()
    }
}

func (app Application) StartWorkers() {
    for i := 0; i < WorkerCount; i++ {
        worker := postal.NewDeliveryWorker(i+1, app.mother.Logger(), app.mother.MailClient(), app.mother.Queue(), app.mother.GlobalUnsubscribesRepo(), app.mother.UnsubscribesRepo(), app.mother.KindsRepo(), app.mother.Database(), app.env.Sender, app.env.EncryptionKey)
        worker.Work()
    }
}

func (app Application) StartServer() {
    web.NewServer().Run(app.env.Port, app.mother)
}

// This is a hack to get the logs output to the loggregator before the process exits
func (app Application) Crash() {
    err := recover()
    if err != nil {
        time.Sleep(5 * time.Second)
        panic(err)
    }
}
