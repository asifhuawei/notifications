package postal_test

import (
    "os"

    "github.com/cloudfoundry-incubator/notifications/config"
    "github.com/cloudfoundry-incubator/notifications/mail"
    "github.com/cloudfoundry-incubator/notifications/postal"
    "github.com/pivotal-cf/uaa-sso-golang/uaa"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("MailSender", func() {

    var mailSender postal.MailSender
    var context postal.MessageContext
    var client mail.Client

    BeforeEach(func() {
        client = mail.Client{}
        context = postal.MessageContext{
            From:      "banana man",
            ReplyTo:   "awesomeness",
            To:        "endless monkeys",
            Subject:   "we will be eaten",
            ClientID:  "3&3",
            MessageID: "4'4",
            Text:      "User <supplied> \"banana\" text",
            HTML:      "<p>user supplied banana html</p>",
            PlainTextEmailTemplate: "Banana preamble {{.Text}} {{.ClientID}} {{.MessageID}}",
            HTMLEmailTemplate:      "Banana preamble {{.HTML}} {{.Text}} {{.ClientID}} {{.MessageID}}",
            SubjectEmailTemplate:   "The Subject: {{.Subject}}",
        }
        mailSender = postal.NewMailSender(&client, context)
    })

    Describe("CompileBody", func() {
        It("returns the compiled email containing both the plaintext and html portions, escaping variables for the html portion only", func() {
            body, err := mailSender.CompileBody()
            if err != nil {
                panic(err)
            }

            emailBody := `
This is a multi-part message in MIME format...

--our-content-boundary
Content-type: text/plain

Banana preamble User <supplied> "banana" text 3&3 4'4
--our-content-boundary
Content-Type: text/html
Content-Disposition: inline
Content-Transfer-Encoding: quoted-printable

<html>
    <body>
        Banana preamble <p>user supplied banana html</p> User &lt;supplied&gt; &#34;banana&#34; text 3&amp;3 4&#39;4
    </body>
</html>
--our-content-boundary--`

            Expect(body).To(Equal(emailBody))
        })

        Context("when no html is set", func() {
            It("only sends a plaintext of the email", func() {
                context.HTML = ""
                mailSender = postal.NewMailSender(&client, context)

                body, err := mailSender.CompileBody()
                if err != nil {
                    panic(err)
                }

                emailBody := `
This is a multi-part message in MIME format...

--our-content-boundary
Content-type: text/plain

Banana preamble User <supplied> "banana" text 3&3 4'4
--our-content-boundary--`
                Expect(body).To(Equal(emailBody))
            })
        })

        Context("when no text is set", func() {
            It("omits the plaintext portion of the email", func() {
                context.Text = ""
                mailSender = postal.NewMailSender(&client, context)

                body, err := mailSender.CompileBody()
                if err != nil {
                    panic(err)
                }

                emailBody := `
This is a multi-part message in MIME format...

--our-content-boundary
Content-Type: text/html
Content-Disposition: inline
Content-Transfer-Encoding: quoted-printable

<html>
    <body>
        Banana preamble <p>user supplied banana html</p>  3&amp;3 4&#39;4
    </body>
</html>
--our-content-boundary--`
                Expect(body).To(Equal(emailBody))
            })
        })
    })

    Describe("CompileMessage", func() {
        It("returns a mail message with all fields", func() {
            message, err := mailSender.CompileMessage("New Body")
            if err != nil {
                panic(err)
            }

            Expect(message.From).To(Equal("banana man"))
            Expect(message.ReplyTo).To(Equal("awesomeness"))
            Expect(message.To).To(Equal("endless monkeys"))
            Expect(message.Subject).To(Equal("The Subject: we will be eaten"))
            Expect(message.Body).To(Equal("New Body"))
            Expect(message.Headers).To(Equal([]string{"X-CF-Client-ID: 3&3", "X-CF-Notification-ID: 4'4"}))
        })
    })

    Describe("NewMessageContext", func() {
        var plainTextEmailTemplate, htmlEmailTemplate, subjectEmailTemplate string
        var user uaa.User
        var env config.Environment
        var options postal.Options

        BeforeEach(func() {
            user = uaa.User{
                ID:     "user-456",
                Emails: []string{"bounce@example.com"},
            }

            env = config.NewEnvironment()

            plainTextEmailTemplate = "the plainText email template"
            htmlEmailTemplate = "the html email template"
            subjectEmailTemplate = "the subject template"

            options = postal.Options{
                ReplyTo:           "awesomeness",
                Subject:           "the subject",
                KindDescription:   "the kind description",
                SourceDescription: "the source description",
                Text:              "user supplied email text",
                HTML:              "user supplied html",
                Kind:              "the-kind",
            }
        })

        It("returns the appropriate MessageContext when all options are specified", func() {
            messageContext := postal.NewMessageContext(user, options, env, "the-space", "the-org",
                "the-client-ID", FakeGuidGenerator, plainTextEmailTemplate, htmlEmailTemplate, subjectEmailTemplate)

            guid, err := FakeGuidGenerator()
            if err != nil {
                panic(err)
            }

            Expect(messageContext.From).To(Equal(os.Getenv("SENDER")))
            Expect(messageContext.ReplyTo).To(Equal(options.ReplyTo))
            Expect(messageContext.To).To(Equal(user.Emails[0]))
            Expect(messageContext.Subject).To(Equal(options.Subject))
            Expect(messageContext.Text).To(Equal(options.Text))
            Expect(messageContext.HTML).To(Equal(options.HTML))
            Expect(messageContext.PlainTextEmailTemplate).To(Equal(plainTextEmailTemplate))
            Expect(messageContext.HTMLEmailTemplate).To(Equal(htmlEmailTemplate))
            Expect(messageContext.SubjectEmailTemplate).To(Equal(subjectEmailTemplate))
            Expect(messageContext.KindDescription).To(Equal(options.KindDescription))
            Expect(messageContext.SourceDescription).To(Equal(options.SourceDescription))
            Expect(messageContext.ClientID).To(Equal("the-client-ID"))
            Expect(messageContext.MessageID).To(Equal(guid.String()))
            Expect(messageContext.Space).To(Equal("the-space"))
            Expect(messageContext.Organization).To(Equal("the-org"))
        })

        It("falls back to Kind if KindDescription is missing", func() {
            options.KindDescription = ""

            messageContext := postal.NewMessageContext(user, options, env, "the-space",
                "the-org", "the-client-ID", FakeGuidGenerator, plainTextEmailTemplate, htmlEmailTemplate, subjectEmailTemplate)

            Expect(messageContext.KindDescription).To(Equal("the-kind"))
        })

        It("falls back to clientID when SourceDescription is missing", func() {
            options.SourceDescription = ""

            messageContext := postal.NewMessageContext(user, options, env, "the-space",
                "the-org", "the-client-ID", FakeGuidGenerator, plainTextEmailTemplate, htmlEmailTemplate, subjectEmailTemplate)

            Expect(messageContext.SourceDescription).To(Equal("the-client-ID"))
        })
    })
})
