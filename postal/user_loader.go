package postal

import (
    "log"
    "net/url"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/cf"
    "github.com/pivotal-cf/uaa-sso-golang/uaa"
)

type UserLoader struct {
    uaaClient UAAInterface
    logger    *log.Logger
}

func NewUserLoader(uaaClient UAAInterface, logger *log.Logger) UserLoader {
    return UserLoader{
        uaaClient: uaaClient,
        logger:    logger,
    }
}

func (loader UserLoader) Load(ccUsers []cf.CloudControllerUser) (map[string]uaa.User, error) {
    users := make(map[string]uaa.User)

    var guids []string
    for _, ccUser := range ccUsers {
        loader.logger.Println("CloudController user guid: " + ccUser.Guid)
        guids = append(guids, ccUser.Guid)
    }

    usersByIDs, err := loader.uaaClient.UsersByIDs(guids...)
    if err != nil {
        return loader.errorFor(err)
    }

    for _, user := range usersByIDs {
        users[user.ID] = user
    }

    for _, guid := range guids {
        if _, ok := users[guid]; !ok {
            users[guid] = uaa.User{}
        }
    }

    return users, nil
}

func (loader UserLoader) errorFor(err error) (map[string]uaa.User, error) {
    users := make(map[string]uaa.User)

    switch err.(type) {
    case *url.Error:
        return users, UAADownError{
            message: "UAA is unavailable",
        }
    case uaa.Failure:
        uaaFailure := err.(uaa.Failure)
        loader.logger.Printf("error:  %v", err)

        if uaaFailure.Code() == 404 {
            if strings.Contains(uaaFailure.Message(), "Requested route") {
                return users, UAADownError{
                    message: "UAA is unavailable",
                }
            } else {
                return users, UAAGenericError{
                    message: "UAA Unknown 404 error message: " + uaaFailure.Message(),
                }
            }
        }

        return users, UAADownError{
            message: "UAA is unavailable",
        }
    default:
        return users, UAAGenericError{
            message: "UAA Unknown Error: " + err.Error(),
        }
    }
}
