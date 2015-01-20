package application

import (
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal/utilities"
	"github.com/cloudfoundry-incubator/notifications/web/services"
)

type ServicesFactory struct {
	database models.DatabaseInterface

	clientsRepo            models.ClientsRepo
	kindsRepo              models.KindsRepo
	preferencesRepo        models.PreferencesRepo
	unsubscribesRepo       models.UnsubscribesRepo
	globalUnsubscribesRepo models.GlobalUnsubscribesRepo
	templatesRepo          models.TemplatesRepo
	messagesRepo           models.MessagesRepo
}

func (s ServicesFactory) NotificationsFinder() services.NotificationsFinder {
	return services.NewNotificationsFinder(s.clientsRepo, s.kindsRepo, s.database)
}

func (s ServicesFactory) NotificationsUpdater() services.NotificationsUpdater {
	return services.NewNotificationsUpdater(s.kindsRepo, s.database)
}

func (s ServicesFactory) PreferencesFinder() *services.PreferencesFinder {
	return services.NewPreferencesFinder(s.preferencesRepo, s.globalUnsubscribesRepo, s.database)
}

func (s ServicesFactory) PreferenceUpdater() services.PreferenceUpdater {
	return services.NewPreferenceUpdater(s.globalUnsubscribesRepo, s.unsubscribesRepo, s.kindsRepo)
}

func (s ServicesFactory) TemplateFinder() services.TemplateFinder {
	return services.NewTemplateFinder(s.templatesRepo, s.database)
}

func (s ServicesFactory) MessageFinder() services.MessageFinder {
	return services.NewMessageFinder(s.messagesRepo, s.database)
}

func (s ServicesFactory) TemplateServiceObjects() (services.TemplateCreator, services.TemplateFinder, services.TemplateUpdater,
	services.TemplateDeleter, services.TemplateLister, services.TemplateAssigner, services.TemplateAssociationLister) {

	return services.NewTemplateCreator(s.templatesRepo, s.database),
		services.NewTemplateFinder(s.templatesRepo, s.database),
		services.NewTemplateUpdater(s.templatesRepo, s.database),
		services.NewTemplateDeleter(s.templatesRepo, s.database),
		services.NewTemplateLister(s.templatesRepo, s.database),
		services.NewTemplateAssigner(s.clientsRepo, s.kindsRepo, s.templatesRepo, s.database),
		services.NewTemplateAssociationLister(s.clientsRepo, s.kindsRepo, s.templatesRepo, s.database)
}

func (s ServicesFactory) TemplatesLoader() utilities.TemplatesLoader {
	return utilities.NewTemplatesLoader(s.TemplateFinder(), s.database, s.clientsRepo, s.kindsRepo, s.templatesRepo)
}

func (s ServicesFactory) Registrar() services.Registrar {
	return services.NewRegistrar(s.clientsRepo, s.kindsRepo)
}
