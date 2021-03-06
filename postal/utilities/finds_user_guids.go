package utilities

import "github.com/cloudfoundry-incubator/notifications/cf"

type FindsUserGUIDs struct {
	cloudController cf.CloudControllerInterface
	uaa             UAAInterface
}

type FindsUserGUIDsInterface interface {
	UserGUIDsBelongingToSpace(string, string) ([]string, error)
	UserGUIDsBelongingToOrganization(string, string, string) ([]string, error)
	UserGUIDsBelongingToScope(string) ([]string, error)
}

func NewFindsUserGUIDs(cloudController cf.CloudControllerInterface, uaa UAAInterface) FindsUserGUIDs {
	return FindsUserGUIDs{
		cloudController: cloudController,
		uaa:             uaa,
	}
}

func (finder FindsUserGUIDs) UserGUIDsBelongingToSpace(spaceGUID, token string) ([]string, error) {
	var userGUIDs []string

	users, err := finder.cloudController.GetUsersBySpaceGuid(spaceGUID, token)
	if err != nil {
		return userGUIDs, err
	}

	for _, user := range users {
		userGUIDs = append(userGUIDs, user.GUID)
	}

	return userGUIDs, nil
}

func (finder FindsUserGUIDs) UserGUIDsBelongingToOrganization(orgGUID, role, token string) ([]string, error) {
	var userGUIDs []string
	var users []cf.CloudControllerUser
	var err error

	switch role {
	case "OrgManager":
		users, err = finder.cloudController.GetManagersByOrgGuid(orgGUID, token)
	case "OrgAuditor":
		users, err = finder.cloudController.GetAuditorsByOrgGuid(orgGUID, token)
	case "BillingManager":
		users, err = finder.cloudController.GetBillingManagersByOrgGuid(orgGUID, token)
	default:
		users, err = finder.cloudController.GetUsersByOrgGuid(orgGUID, token)
	}

	if err != nil {
		return userGUIDs, err
	}

	for _, user := range users {
		userGUIDs = append(userGUIDs, user.GUID)
	}

	return userGUIDs, nil
}

func (finder FindsUserGUIDs) UserGUIDsBelongingToScope(scope string) ([]string, error) {
	var userGUIDs []string

	userGUIDs, err := finder.uaa.UsersGUIDsByScope(scope)
	if err != nil {
		return userGUIDs, err
	}

	return userGUIDs, nil
}
