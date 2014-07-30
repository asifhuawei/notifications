package postal

import "github.com/cloudfoundry-incubator/notifications/cf"

type SpaceLoader struct {
    cloudController cf.CloudControllerInterface
}

func NewSpaceLoader(cloudController cf.CloudControllerInterface) SpaceLoader {
    return SpaceLoader{
        cloudController: cloudController,
    }
}

func (loader SpaceLoader) Load(guid, token string) (string, string, error) {
    space, err := loader.cloudController.LoadSpace(guid, token)
    if err != nil {
        return "", "", err
    }

    org, err := loader.cloudController.LoadOrganization(space.OrganizationGuid, token)
    if err != nil {
        return "", "", err
    }

    return space.Name, org.Name, nil
}
