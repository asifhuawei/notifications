package postal_test

import (
    "bytes"
    "encoding/json"
    "errors"
    "log"
    "net/http"
    "net/http/httptest"
    "strings"

    "github.com/cloudfoundry-incubator/notifications/cf"
    "github.com/cloudfoundry-incubator/notifications/postal"
    "github.com/pivotal-cf/uaa-sso-golang/uaa"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Courier", func() {
    var courier postal.Courier
    var fakeCC *FakeCloudController
    var logger *log.Logger
    var fakeUAA FakeUAAClient
    var mailClient FakeMailClient
    var writer *httptest.ResponseRecorder
    var token string
    var buffer *bytes.Buffer
    var options postal.Options

    BeforeEach(func() {
        tokenHeader := map[string]interface{}{
            "alg": "FAST",
        }

        tokenClaims := map[string]interface{}{
            "client_id": "mister-client",
            "exp":       3404281214,
            "scope":     []string{"notifications.write"},
        }

        token = BuildToken(tokenHeader, tokenClaims)

        writer = httptest.NewRecorder()

        fakeCC = NewFakeCloudController()
        fakeCC.UsersBySpaceGuid["space-001"] = []cf.CloudControllerUser{
            cf.CloudControllerUser{Guid: "user-123"},
            cf.CloudControllerUser{Guid: "user-456"},
        }

        fakeCC.Spaces = map[string]cf.CloudControllerSpace{
            "space-001": cf.CloudControllerSpace{
                Name:             "production",
                Guid:             "space-001",
                OrganizationGuid: "org-001",
            },
        }

        fakeCC.Orgs = map[string]cf.CloudControllerOrganization{
            "org-001": cf.CloudControllerOrganization{
                Name: "pivotaltracker",
            },
        }

        fakeUAA = FakeUAAClient{
            ClientToken: uaa.Token{
                Access: token,
            },
            UsersByID: map[string]uaa.User{
                "user-123": uaa.User{
                    ID:     "user-123",
                    Emails: []string{"user-123@example.com"},
                },
                "user-456": uaa.User{
                    ID:     "user-456",
                    Emails: []string{"user-456@example.com"},
                },
            },
        }

        buffer = bytes.NewBuffer([]byte{})
        logger = log.New(buffer, "", 0)

        mailClient = FakeMailClient{}

        courier = postal.NewCourier(logger, fakeCC, &fakeUAA, &mailClient, FakeGuidGenerator)
    })

    Describe("NofifyServeHTTP", func() {
        Context("when the request is valid", func() {
            BeforeEach(func() {
                options = postal.Options{
                    Kind:              "forgot_password",
                    KindDescription:   "Password reminder",
                    SourceDescription: "Login system",
                    Text:              "Please reset your password by clicking on this link...",
                    HTML:              "<p>Please reset your password by clicking on this link...</p>",
                }
            })

            Context("failure cases", func() {
                Context("when Cloud Controller is unavailable to load space users", func() {
                    It("returns a CCDownError error", func() {
                        fakeCC.GetUsersBySpaceGuidError = errors.New("BOOM!")
                        err := courier.Dispatch(writer, token, "user-123", postal.IsSpace, options)

                        Expect(err).To(BeAssignableToTypeOf(postal.CCDownError("")))
                    })
                })

                Context("when Cloud Controller is unavailable to load a space", func() {
                    It("returns a CCDownError error", func() {
                        fakeCC.LoadSpaceError = errors.New("BOOM!")
                        err := courier.Dispatch(writer, token, "user-123", postal.IsSpace, options)

                        Expect(err).To(BeAssignableToTypeOf(postal.CCDownError("")))
                    })
                })

                Context("when UAA cannot be reached", func() {
                    It("returns a UAADownError", func() {
                        fakeUAA.ErrorForUserByID = uaa.NewFailure(404, []byte("Requested route ('uaa.10.244.0.34.xip.io') does not exist"))
                        err := courier.Dispatch(writer, token, "user-123", postal.IsUser, options)

                        Expect(err).To(BeAssignableToTypeOf(postal.UAADownError("")))
                    })
                })

                Context("when UAA fails for unknown reasons", func() {
                    It("returns a UAAGenericError", func() {
                        fakeUAA.ErrorForUserByID = errors.New("BOOM!")
                        err := courier.Dispatch(writer, token, "user-123", postal.IsUser, options)

                        Expect(err).To(BeAssignableToTypeOf(postal.UAAGenericError("")))
                    })
                })
            })

            Context("when the SMTP server fails to deliver the mail", func() {
                It("returns a status indicating that delivery failed", func() {
                    mailClient.errorOnSend = true
                    err := courier.Dispatch(writer, token, "user-123", postal.IsUser, options)
                    if err != nil {
                        panic(err)
                    }

                    Expect(writer.Code).To(Equal(http.StatusOK))
                    parsed := []map[string]string{}
                    err = json.Unmarshal(writer.Body.Bytes(), &parsed)
                    if err != nil {
                        panic(err)
                    }

                    Expect(parsed[0]["status"]).To(Equal("failed"))
                })
            })

            Context("when the SMTP server cannot be reached", func() {
                It("returns a status indicating that the server is unavailable", func() {
                    mailClient.errorOnConnect = true
                    err := courier.Dispatch(writer, token, "user-123", postal.IsUser, options)
                    if err != nil {
                        panic(err)
                    }

                    Expect(writer.Code).To(Equal(http.StatusOK))
                    parsed := []map[string]string{}
                    err = json.Unmarshal(writer.Body.Bytes(), &parsed)
                    if err != nil {
                        panic(err)
                    }

                    Expect(parsed[0]["status"]).To(Equal("unavailable"))
                })
            })

            Context("when UAA cannot find the user", func() {
                It("returns that the user in the response with status notfound", func() {
                    err := courier.Dispatch(writer, token, "user-789", postal.IsUser, options)
                    if err != nil {
                        panic(err)
                    }

                    Expect(writer.Code).To(Equal(http.StatusOK))

                    response := []map[string]string{}
                    err = json.Unmarshal(writer.Body.Bytes(), &response)
                    if err != nil {
                        panic(err)
                    }
                    Expect(response[0]["status"]).To(Equal(postal.StatusNotFound))
                    Expect(response[0]["recipient"]).To(Equal("user-789"))
                })
            })

            Context("when the UAA user has no email", func() {
                It("returns the user in the response with the status noaddress", func() {
                    fakeUAA.UsersByID["user-123"] = uaa.User{
                        ID:     "user-123",
                        Emails: []string{},
                    }

                    err := courier.Dispatch(writer, token, "user-123", postal.IsUser, options)
                    if err != nil {
                        panic(err)
                    }

                    response := []map[string]string{}
                    err = json.Unmarshal(writer.Body.Bytes(), &response)
                    if err != nil {
                        panic(err)
                    }

                    Expect(writer.Code).To(Equal(http.StatusOK))
                    Expect(response[0]["status"]).To(Equal(postal.StatusNoAddress))
                })
            })

            Context("When load Users returns multiple users", func() {
                It("logs the UUIDs of all recipients", func() {
                    err := courier.Dispatch(writer, token, "space-001", postal.IsSpace, options)
                    if err != nil {
                        panic(err)
                    }

                    lines := strings.Split(buffer.String(), "\n")

                    Expect(lines).To(ContainElement("CloudController user guid: user-123"))
                    Expect(lines).To(ContainElement("CloudController user guid: user-456"))
                })

                It("returns necessary info in the response for the sent mail", func() {
                    courier = postal.NewCourier(logger, fakeCC, &fakeUAA, &mailClient, FakeGuidGenerator)
                    err := courier.Dispatch(writer, token, "space-001", postal.IsSpace, options)
                    if err != nil {
                        panic(err)
                    }

                    Expect(writer.Code).To(Equal(http.StatusOK))
                    parsed := []map[string]string{}
                    err = json.Unmarshal(writer.Body.Bytes(), &parsed)
                    if err != nil {
                        panic(err)
                    }

                    Expect(parsed).To(ContainElement(map[string]string{
                        "recipient":       "user-123",
                        "status":          "delivered",
                        "notification_id": "deadbeef-aabb-ccdd-eeff-001122334455",
                    }))

                    Expect(parsed).To(ContainElement(map[string]string{
                        "recipient":       "user-456",
                        "status":          "delivered",
                        "notification_id": "deadbeef-aabb-ccdd-eeff-001122334455",
                    }))
                })
            })
        })
    })

    Describe("SendMailToUser", func() {
        It("logs the email address of the recipient and returns the status", func() {
            messageContext := postal.MessageContext{
                To: "fake-user@example.com",
            }

            mailClient = FakeMailClient{}

            status := courier.SendMailToUser(messageContext, logger, &mailClient)

            Expect(buffer.String()).To(ContainSubstring("Sending email to fake-user@example.com"))
            Expect(status).To(Equal("delivered"))
        })

        It("logs the message envelope", func() {
            messageContext := postal.MessageContext{
                To:                     "fake-user@example.com",
                From:                   "from@email.com",
                Subject:                "the subject",
                Text:                   "body content",
                KindDescription:        "the kind description",
                PlainTextEmailTemplate: "{{.Text}}",
                SubjectEmailTemplate:   "{{.Subject}}",
            }

            mailClient = FakeMailClient{}

            courier.SendMailToUser(messageContext, logger, &mailClient)

            data := []string{
                "From: from@email.com",
                "To: fake-user@example.com",
                "Subject: the subject",
                `body content`,
            }
            results := strings.Split(buffer.String(), "\n")
            for _, item := range data {
                Expect(results).To(ContainElement(item))
            }
        })
    })

    Describe("LoadSubjectTemplate", func() {
        var manager postal.EmailTemplateManager

        Context("when subject is not set in the params", func() {
            It("returns the subject.missing template", func() {
                manager.ReadFile = func(path string) (string, error) {
                    if strings.Contains(path, "missing") {
                        return "the missing subject", nil
                    }
                    return "incorrect", nil
                }

                manager.FileExists = func(path string) bool {
                    return false
                }

                subject := ""

                subjectTemplate, err := courier.LoadSubjectTemplate(subject, manager)
                if err != nil {
                    panic(err)
                }

                Expect(subjectTemplate).To(Equal("the missing subject"))
            })
        })

        Context("when subject is set in the params", func() {
            It("returns the subject.provided template", func() {
                manager.ReadFile = func(path string) (string, error) {
                    if strings.Contains(path, "provided") {
                        return "the provided subject", nil
                    }
                    return "incorrect", nil
                }

                manager.FileExists = func(path string) bool {
                    return false
                }

                subject := "is provided"

                subjectTemplate, err := courier.LoadSubjectTemplate(subject, manager)
                if err != nil {
                    panic(err)
                }

                Expect(subjectTemplate).To(Equal("the provided subject"))
            })
        })
    })

    Describe("LoadBodyTemplates", func() {
        var manager postal.EmailTemplateManager

        Context("loadSpace is true", func() {
            It("returns the space templates", func() {

                manager.ReadFile = func(path string) (string, error) {
                    if strings.Contains(path, "space") && strings.Contains(path, "text") {
                        return "space plain text", nil
                    }
                    if strings.Contains(path, "space") && strings.Contains(path, "html") {
                        return "space html code", nil
                    }
                    return "incorrect", nil
                }

                manager.FileExists = func(path string) bool {
                    return false
                }

                plain, html, err := courier.LoadBodyTemplates(postal.IsSpace, manager)
                if err != nil {
                    panic(err)
                }

                Expect(plain).To(Equal("space plain text"))
                Expect(html).To(Equal("space html code"))
            })
        })

        Context("loadSpace is false", func() {
            It("returns the user templates", func() {
                manager.ReadFile = func(path string) (string, error) {
                    if strings.Contains(path, "user") && strings.Contains(path, "text") {
                        return "user plain text", nil
                    }
                    if strings.Contains(path, "user") && strings.Contains(path, "html") {
                        return "user html code", nil
                    }
                    return "incorrect", nil
                }

                manager.FileExists = func(path string) bool {
                    return false
                }

                plain, html, err := courier.LoadBodyTemplates(postal.IsUser, manager)
                if err != nil {
                    panic(err)
                }

                Expect(plain).To(Equal("user plain text"))
                Expect(html).To(Equal("user html code"))
            })
        })
    })
})
