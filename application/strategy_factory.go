package application

import (
	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal/strategies"
	"github.com/cloudfoundry-incubator/notifications/postal/utilities"
	"github.com/pivotal-cf/uaa-sso-golang/uaa"
)

type StrategyFactory struct {
	env             Environment
	uaaClient       uaa.UAA
	tokenLoader     utilities.TokenLoader
	mailer          strategies.Mailer
	templatesLoader utilities.TemplatesLoader
	receiptsRepo    models.ReceiptsRepo
	userLoader      utilities.UserLoader
	cloudController cf.CloudController
	findsUserGUIDs  utilities.FindsUserGUIDs
}

func (sf StrategyFactory) UserStrategy() strategies.UserStrategy {
	return strategies.NewUserStrategy(sf.tokenLoader, sf.userLoader, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) SpaceStrategy() strategies.SpaceStrategy {
	spaceLoader := utilities.NewSpaceLoader(sf.cloudController)
	organizationLoader := utilities.NewOrganizationLoader(sf.cloudController)
	return strategies.NewSpaceStrategy(sf.tokenLoader, sf.userLoader, spaceLoader, organizationLoader, sf.findsUserGUIDs, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) OrganizationStrategy() strategies.OrganizationStrategy {
	organizationLoader := utilities.NewOrganizationLoader(sf.cloudController)
	return strategies.NewOrganizationStrategy(sf.tokenLoader, sf.userLoader, organizationLoader, sf.findsUserGUIDs, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) EveryoneStrategy() strategies.EveryoneStrategy {
	allUsers := utilities.NewAllUsers(&sf.uaaClient)
	return strategies.NewEveryoneStrategy(sf.tokenLoader, allUsers, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) UAAScopeStrategy() strategies.UAAScopeStrategy {
	return strategies.NewUAAScopeStrategy(sf.tokenLoader, sf.userLoader, sf.findsUserGUIDs, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) EmailStrategy() strategies.EmailStrategy {
	return strategies.NewEmailStrategy(sf.mailer, sf.templatesLoader)
}
