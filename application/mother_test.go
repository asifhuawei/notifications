package application_test

import (
	"github.com/cloudfoundry-incubator/notifications/application"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mother", func() {
	Describe("UAAClient", func() {
		It("Returns a UAA client configured from the environment", func() {
			env := application.Environment{
				UAAHost:         "uaa.example.com",
				UAAClientID:     "notifications-admin",
				UAAClientSecret: "password",
				VerifySSL:       false,
			}
			m := application.NewMother(env)
			uaaClient := m.UAAClient()

			Expect(uaaClient.ClientID).To(Equal(env.UAAClientID))
			Expect(uaaClient.ClientSecret).To(Equal(env.UAAClientSecret))
			Expect(uaaClient.VerifySSL).To(Equal(env.VerifySSL))
		})
	})
})
