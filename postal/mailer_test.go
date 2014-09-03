package postal_test

import (
    "bytes"
    "log"

    "github.com/cloudfoundry-incubator/notifications/models"
    "github.com/cloudfoundry-incubator/notifications/postal"
    "github.com/pivotal-cf/uaa-sso-golang/uaa"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Mailer", func() {
    var mailClient FakeMailClient
    var mailer postal.Mailer
    var logger *log.Logger
    var buffer *bytes.Buffer
    var queue *FakeQueue
    var unsubscribesRepo *FakeUnsubscribesRepo
    var conn *FakeDBConn

    BeforeEach(func() {
        buffer = bytes.NewBuffer([]byte{})
        logger = log.New(buffer, "", 0)
        mailClient = FakeMailClient{}
        queue = NewFakeQueue()
        unsubscribesRepo = NewFakeUnsubscribesRepo()
        conn = &FakeDBConn{}
        mailer = postal.NewMailer(queue, FakeGuidGenerator)
    })

    Describe("Deliver", func() {
        It("returns the correct types of responses for users", func() {
            users := map[string]uaa.User{
                "user-1": {ID: "user-1", Emails: []string{"user-1@example.com"}},
                "user-2": {},
                "user-3": {ID: "user-3"},
                "user-4": {ID: "user-4", Emails: []string{"user-4"}},
            }
            responses := mailer.Deliver(postal.Templates{}, users, postal.Options{}, "the-space", "the-org", "the-client")

            Expect(len(responses)).To(Equal(4))
            Expect(responses).To(ContainElement(postal.Response{
                Status:         "queued",
                Recipient:      "user-1",
                NotificationID: "deadbeef-aabb-ccdd-eeff-001122334455",
            }))

            Expect(responses).To(ContainElement(postal.Response{
                Status:         "queued",
                Recipient:      "user-2",
                NotificationID: "deadbeef-aabb-ccdd-eeff-001122334455",
            }))

            Expect(responses).To(ContainElement(postal.Response{
                Status:         "queued",
                Recipient:      "user-3",
                NotificationID: "deadbeef-aabb-ccdd-eeff-001122334455",
            }))

            Expect(responses).To(ContainElement(postal.Response{
                Status:         "queued",
                Recipient:      "user-4",
                NotificationID: "deadbeef-aabb-ccdd-eeff-001122334455",
            }))
        })

        FIt("enqueues jobs with the deliveries", func() {
            users := map[string]uaa.User{
                "user-1": {ID: "user-1", Emails: []string{"user-1@example.com"}},
                "user-2": {},
                "user-3": {ID: "user-3"},
                "user-4": {ID: "user-4", Emails: []string{"user-4"}},
            }
            mailer.Deliver(postal.Templates{}, users, postal.Options{}, "the-space", "the-org", "the-client")

            for userGUID, user := range users {
                job := <-queue.Reserve("me")
                var delivery postal.Delivery
                err := job.Unmarshal(&delivery)
                if err != nil {
                    panic(err)
                }
                Expect(delivery).To(Equal(postal.Delivery{
                    User: user,
                    Options: postal.Options{
                        ReplyTo: "",
                        Subject: "",
                        KindDescription: "",
                        SourceDescription: "",
                        Text: "",
                        HTML: "",
                        KindID: "",
                    },
                    UserGUID: userGUID,
                    Space: "the-space",
                    Organization: "the-org",
                    ClientID: "the-client",
                    Templates: postal.Templates{Subject: "", Text: "", HTML: ""},
                    MessageID: "deadbeef-aabb-ccdd-eeff-001122334455",
                    Subscribed: true,
                }))
            }
        })

        It("checks to see if the recipient is unsubscribed from the notification", func() {
            _, err := unsubscribesRepo.Create(conn, models.Unsubscribe{
                UserID:   "user-123",
                ClientID: "some-client",
                KindID:   "some-kind",
            })
            if err != nil {
                panic(err)
            }

            users := map[string]uaa.User{
                "user-123": {ID: "user-123", Emails: []string{"user-123@example.com"}},
            }
            mailer.Deliver(postal.Templates{}, users, postal.Options{}, "the-space", "the-org", "the-client")

            job := <-queue.Reserve("me")
            var delivery postal.Delivery
            err = job.Unmarshal(&delivery)
            if err != nil {
                panic(err)
            }
            Expect(delivery.Subscribed).To(BeFalse())
        })
    })
})
