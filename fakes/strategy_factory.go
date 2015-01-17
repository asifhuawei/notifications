package fakes

import (
	"github.com/cloudfoundry-incubator/notifications/postal/strategies"
)

type StrategyFactory struct{}

func NewStrategyFactory() StrategyFactory {
	return StrategyFactory{}
}

func (f StrategyFactory) EmailStrategy() strategies.EmailStrategy {
	return strategies.EmailStrategy{}
}

func (f StrategyFactory) UserStrategy() strategies.UserStrategy {
	return strategies.UserStrategy{}
}

func (f StrategyFactory) SpaceStrategy() strategies.SpaceStrategy {
	return strategies.SpaceStrategy{}
}

func (f StrategyFactory) OrganizationStrategy() strategies.OrganizationStrategy {
	return strategies.OrganizationStrategy{}
}

func (f StrategyFactory) EveryoneStrategy() strategies.EveryoneStrategy {
	return strategies.EveryoneStrategy{}
}

func (f StrategyFactory) UAAScopeStrategy() strategies.UAAScopeStrategy {
	return strategies.UAAScopeStrategy{}
}
