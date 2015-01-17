package web

import (
	"strings"

	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal/strategies"
	"github.com/cloudfoundry-incubator/notifications/web/handlers"
	"github.com/cloudfoundry-incubator/notifications/web/middleware"
	"github.com/cloudfoundry-incubator/notifications/web/services"
	"github.com/gorilla/mux"
	"github.com/ryanmoran/stack"
)

type strategyFactory interface {
	EmailStrategy() strategies.EmailStrategy
	UserStrategy() strategies.UserStrategy
	SpaceStrategy() strategies.SpaceStrategy
	OrganizationStrategy() strategies.OrganizationStrategy
	EveryoneStrategy() strategies.EveryoneStrategy
	UAAScopeStrategy() strategies.UAAScopeStrategy
}

type MotherInterface interface {
	Registrar() services.Registrar
	NotificationsFinder() services.NotificationsFinder
	NotificationsUpdater() services.NotificationsUpdater
	PreferencesFinder() *services.PreferencesFinder
	PreferenceUpdater() services.PreferenceUpdater
	MessageFinder() services.MessageFinder
	TemplateServiceObjects() (services.TemplateCreator, services.TemplateFinder, services.TemplateUpdater, services.TemplateDeleter, services.TemplateLister, services.TemplateAssigner, services.TemplateAssociationLister)
	Database() models.DatabaseInterface
	Logging() stack.Middleware
	CORS() middleware.CORS
}

type Router struct {
	stacks map[string]stack.Stack
}

func NewRouter(mother MotherInterface, strategies strategyFactory, authenticator func(...string) middleware.Authenticator) Router {
	registrar := mother.Registrar()
	notificationsFinder := mother.NotificationsFinder()

	notify := handlers.NewNotify(notificationsFinder, registrar)

	preferencesFinder := mother.PreferencesFinder()
	preferenceUpdater := mother.PreferenceUpdater()
	templateCreator, templateFinder, templateUpdater, templateDeleter, templateLister, templateAssigner, templateAssociationLister := mother.TemplateServiceObjects()
	notificationsUpdater := mother.NotificationsUpdater()
	messageFinder := mother.MessageFinder()

	logging := mother.Logging()
	errorWriter := handlers.NewErrorWriter()
	database := mother.Database()
	cors := mother.CORS()

	notificationsWriteAuthenticator := authenticator("notifications.write")
	notificationsManageAuthenticator := authenticator("notifications.manage")
	notificationPreferencesReadAuthenticator := authenticator("notification_preferences.read")
	notificationPreferencesWriteAuthenticator := authenticator("notification_preferences.write")
	notificationPreferencesAdminAuthenticator := authenticator("notification_preferences.admin")
	emailsWriteAuthenticator := authenticator("emails.write")
	notificationsTemplateWriteAuthenticator := authenticator("notification_templates.write")
	notificationsTemplateReadAuthenticator := authenticator("notification_templates.read")
	notificationsWriteOrEmailsWriteAuthenticator := authenticator("notifications.write", "emails.write")

	return Router{
		stacks: map[string]stack.Stack{
			"GET /info":                                                       stack.NewStack(handlers.NewGetInfo()).Use(logging),
			"POST /users/{guid}":                                              stack.NewStack(handlers.NewNotifyUser(notify, errorWriter, strategies.UserStrategy(), database)).Use(logging, notificationsWriteAuthenticator),
			"POST /spaces/{guid}":                                             stack.NewStack(handlers.NewNotifySpace(notify, errorWriter, strategies.SpaceStrategy(), database)).Use(logging, notificationsWriteAuthenticator),
			"POST /organizations/{guid}":                                      stack.NewStack(handlers.NewNotifyOrganization(notify, errorWriter, strategies.OrganizationStrategy(), database)).Use(logging, notificationsWriteAuthenticator),
			"POST /everyone":                                                  stack.NewStack(handlers.NewNotifyEveryone(notify, errorWriter, strategies.EveryoneStrategy(), database)).Use(logging, notificationsWriteAuthenticator),
			"POST /uaa_scopes/{scope}":                                        stack.NewStack(handlers.NewNotifyUAAScope(notify, errorWriter, strategies.UAAScopeStrategy(), database)).Use(logging, notificationsWriteAuthenticator),
			"POST /emails":                                                    stack.NewStack(handlers.NewNotifyEmail(notify, errorWriter, strategies.EmailStrategy(), database)).Use(logging, emailsWriteAuthenticator),
			"PUT /registration":                                               stack.NewStack(handlers.NewRegisterNotifications(registrar, errorWriter, database)).Use(logging, notificationsWriteAuthenticator),
			"PUT /notifications":                                              stack.NewStack(handlers.NewRegisterClientWithNotifications(registrar, errorWriter, database)).Use(logging, notificationsWriteAuthenticator),
			"PUT /clients/{clientID}/notifications/{notificationID}":          stack.NewStack(handlers.NewUpdateNotifications(notificationsUpdater, errorWriter)).Use(logging, notificationsManageAuthenticator),
			"GET /notifications":                                              stack.NewStack(handlers.NewGetAllNotifications(notificationsFinder, errorWriter)).Use(logging, notificationsManageAuthenticator),
			"OPTIONS /user_preferences":                                       stack.NewStack(handlers.NewOptionsPreferences()).Use(logging, cors),
			"OPTIONS /user_preferences/{guid}":                                stack.NewStack(handlers.NewOptionsPreferences()).Use(logging, cors),
			"GET /user_preferences":                                           stack.NewStack(handlers.NewGetPreferences(preferencesFinder, errorWriter)).Use(logging, cors, notificationPreferencesReadAuthenticator),
			"GET /user_preferences/{guid}":                                    stack.NewStack(handlers.NewGetPreferencesForUser(preferencesFinder, errorWriter)).Use(logging, cors, notificationPreferencesAdminAuthenticator),
			"PATCH /user_preferences":                                         stack.NewStack(handlers.NewUpdatePreferences(preferenceUpdater, errorWriter, database)).Use(logging, cors, notificationPreferencesWriteAuthenticator),
			"PATCH /user_preferences/{guid}":                                  stack.NewStack(handlers.NewUpdateSpecificUserPreferences(preferenceUpdater, errorWriter, database)).Use(logging, cors, notificationPreferencesAdminAuthenticator),
			"POST /templates":                                                 stack.NewStack(handlers.NewCreateTemplate(templateCreator, errorWriter)).Use(logging, notificationsTemplateWriteAuthenticator),
			"GET /default_template":                                           stack.NewStack(handlers.NewGetDefaultTemplate(templateFinder, errorWriter)).Use(logging, notificationsTemplateReadAuthenticator),
			"PUT /default_template":                                           stack.NewStack(handlers.NewUpdateDefaultTemplate(templateUpdater, errorWriter)).Use(logging, notificationsTemplateWriteAuthenticator),
			"GET /templates/{templateID}":                                     stack.NewStack(handlers.NewGetTemplates(templateFinder, errorWriter)).Use(logging, notificationsTemplateReadAuthenticator),
			"PUT /templates/{templateID}":                                     stack.NewStack(handlers.NewUpdateTemplates(templateUpdater, errorWriter)).Use(logging, notificationsTemplateWriteAuthenticator),
			"DELETE /templates/{templateID}":                                  stack.NewStack(handlers.NewDeleteTemplates(templateDeleter, errorWriter)).Use(logging, notificationsTemplateWriteAuthenticator),
			"GET /templates":                                                  stack.NewStack(handlers.NewListTemplates(templateLister, errorWriter)).Use(logging, notificationsTemplateReadAuthenticator),
			"PUT /clients/{clientID}/template":                                stack.NewStack(handlers.NewAssignClientTemplate(templateAssigner, errorWriter)).Use(logging, notificationsManageAuthenticator),
			"PUT /clients/{clientID}/notifications/{notificationID}/template": stack.NewStack(handlers.NewAssignNotificationTemplate(templateAssigner, errorWriter)).Use(logging, notificationsManageAuthenticator),
			"GET /templates/{templateID}/associations":                        stack.NewStack(handlers.NewListTemplateAssociations(templateAssociationLister, errorWriter)).Use(logging, notificationsManageAuthenticator),
			"GET /messages/{messageID}":                                       stack.NewStack(handlers.NewGetMessages(messageFinder, errorWriter)).Use(logging, notificationsWriteOrEmailsWriteAuthenticator),
		},
	}
}

func (router Router) Routes() *mux.Router {
	r := mux.NewRouter()
	for methodPath, stack := range router.stacks {
		var name = methodPath
		parts := strings.SplitN(methodPath, " ", 2)
		r.Handle(parts[1], stack).Methods(parts[0]).Name(name)
	}
	return r
}
