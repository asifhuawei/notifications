package fakes

import (
	"github.com/cloudfoundry-incubator/notifications/web/services"
)

type ServicesFactory struct{}

func NewServicesFactory() ServicesFactory {
	return ServicesFactory{}
}

func (sf ServicesFactory) Registrar() services.Registrar {
	return services.Registrar{}
}

func (sf ServicesFactory) NotificationsFinder() services.NotificationsFinder {
	return services.NotificationsFinder{}
}

func (sf ServicesFactory) NotificationsUpdater() services.NotificationsUpdater {
	return services.NotificationsUpdater{}
}

func (sf ServicesFactory) PreferencesFinder() *services.PreferencesFinder {
	return &services.PreferencesFinder{}
}

func (sf ServicesFactory) PreferenceUpdater() services.PreferenceUpdater {
	return services.PreferenceUpdater{}
}

func (sf ServicesFactory) MessageFinder() services.MessageFinder {
	return services.MessageFinder{}
}

func (sf ServicesFactory) TemplateServiceObjects() (services.TemplateCreator, services.TemplateFinder,
	services.TemplateUpdater, services.TemplateDeleter, services.TemplateLister,
	services.TemplateAssigner, services.TemplateAssociationLister) {
	return services.TemplateCreator{}, services.TemplateFinder{}, services.TemplateUpdater{}, services.TemplateDeleter{}, services.TemplateLister{}, services.TemplateAssigner{}, services.TemplateAssociationLister{}
}
