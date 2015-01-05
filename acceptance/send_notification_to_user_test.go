package acceptance

import (
	"net/http"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/notifications/acceptance/support"
	"github.com/cloudfoundry-incubator/notifications/application"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Send a notification to a user", func() {
	It("sends a single notification email to a user", func() {
		var templateID string
		var response support.NotifyResponse
		env := application.NewEnvironment()
		clientID := "notifications-sender"
		clientToken := GetClientTokenFor(clientID)
		client := support.NewClient(Servers.Notifications)
		userID := "user-123"

		By("registering a notification", func() {
			code, err := client.Notifications.Register(clientToken.Access, support.RegisterClient{
				SourceName: "Notifications Sender",
				Notifications: map[string]support.RegisterNotification{
					"acceptance-test": {
						Description: "Acceptance Test",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(http.StatusNoContent))
		})

		By("creating a new template", func() {
			var status int
			var err error
			status, templateID, err = client.Templates.Create(clientToken.Access, support.Template{
				Name:    "Star Wars",
				Subject: "Awesomeness {{.Subject}}",
				HTML:    "<p>Millenium Falcon</p>{{.HTML}}",
				Text:    "Millenium Falcon\n{{.Text}}",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusCreated))
			Expect(templateID).NotTo(BeNil())
		})

		By("assigning the template to a client", func() {
			status, err := client.Templates.AssignToClient(clientToken.Access, clientID, templateID)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusNoContent))
		})

		By("sending a notifications to a user", func() {
			status, responses, err := client.Notify.User(clientToken.Access, userID, support.Notify{
				KindID:  "acceptance-test",
				HTML:    "<p>this is an acceptance%40test</p>",
				Text:    "hello from the acceptance test",
				Subject: "my-special-subject",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusOK))

			Expect(responses).To(HaveLen(1))
			response = responses[0]
			Expect(response.Status).To(Equal("queued"))
			Expect(response.Recipient).To(Equal(userID))
			Expect(GUIDRegex.MatchString(response.NotificationID)).To(BeTrue())
		})

		By("verifying that the message was sent", func() {
			Eventually(func() int {
				return len(Servers.SMTP.Deliveries)
			}, 1*time.Second).Should(Equal(1))
			delivery := Servers.SMTP.Deliveries[0]

			Expect(delivery.Sender).To(Equal(env.Sender))
			Expect(delivery.Recipients).To(Equal([]string{"user-123@example.com"}))

			data := strings.Split(string(delivery.Data), "\n")
			Expect(data).To(ContainElement("X-CF-Client-ID: notifications-sender"))
			Expect(data).To(ContainElement("X-CF-Notification-ID: " + response.NotificationID))
			Expect(data).To(ContainElement("Subject: Awesomeness my-special-subject"))
			Expect(data).To(ContainElement(`        <p>Millenium Falcon</p><p>this is an acceptance%40test</p>`))
			Expect(data).To(ContainElement("hello from the acceptance test"))
		})
	})
})
