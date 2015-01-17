package application

import (
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal/strategies"
	"github.com/cloudfoundry-incubator/notifications/postal/utilities"
)

type StrategyFactory struct {
	templatesLoader    utilities.TemplatesLoader
	allUsers           utilities.AllUsers
	tokenLoader        utilities.TokenLoader
	userLoader         utilities.UserLoader
	findsUserGUIDs     utilities.FindsUserGUIDs
	spaceLoader        utilities.SpaceLoader
	organizationLoader utilities.OrganizationLoader
	mailer             strategies.Mailer
	receiptsRepo       models.ReceiptsRepo
}

func (sf StrategyFactory) UserStrategy() strategies.UserStrategy {
	return strategies.NewUserStrategy(sf.tokenLoader, sf.userLoader, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) SpaceStrategy() strategies.SpaceStrategy {
	return strategies.NewSpaceStrategy(sf.tokenLoader, sf.userLoader, sf.spaceLoader, sf.organizationLoader, sf.findsUserGUIDs, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) OrganizationStrategy() strategies.OrganizationStrategy {
	return strategies.NewOrganizationStrategy(sf.tokenLoader, sf.userLoader, sf.organizationLoader, sf.findsUserGUIDs, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) EveryoneStrategy() strategies.EveryoneStrategy {
	return strategies.NewEveryoneStrategy(sf.tokenLoader, sf.allUsers, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) UAAScopeStrategy() strategies.UAAScopeStrategy {
	return strategies.NewUAAScopeStrategy(sf.tokenLoader, sf.userLoader, sf.findsUserGUIDs, sf.templatesLoader, sf.mailer, sf.receiptsRepo)
}

func (sf StrategyFactory) EmailStrategy() strategies.EmailStrategy {
	return strategies.NewEmailStrategy(sf.mailer, sf.templatesLoader)
}
