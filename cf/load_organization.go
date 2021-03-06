package cf

import (
	"time"

	"github.com/cloudfoundry-incubator/notifications/metrics"
)

func (cc CloudController) LoadOrganization(guid, token string) (CloudControllerOrganization, error) {
	then := time.Now()

	org, err := cc.client.Organizations.Get(guid, token)
	if err != nil {
		return CloudControllerOrganization{}, NewFailure(0, err.Error())
	}

	duration := time.Now().Sub(then)

	metrics.NewMetric("histogram", map[string]interface{}{
		"name":  "notifications.external-requests.cc.organization",
		"value": duration.Seconds(),
	}).Log()

	return CloudControllerOrganization{
		GUID: org.GUID,
		Name: org.Name,
	}, nil
}
