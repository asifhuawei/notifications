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
    "github.com/cloudfoundry-incubator/notifications/config"
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
    var userLoader postal.UserLoader
    var spaceLoader postal.SpaceLoader
    var templateLoader postal.TemplateLoader
    var mailer postal.Mailer
    var fs FakeFileSystem
    var env config.Environment

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
        env = config.NewEnvironment()
        fs = NewFakeFileSystem(env)

        userLoader = postal.NewUserLoader(&fakeUAA, logger, fakeCC)
        spaceLoader = postal.NewSpaceLoader(fakeCC)
        templateLoader = postal.NewTemplateLoader(&fs)
        mailer = postal.NewMailer(FakeGuidGenerator, logger, &mailClient)

        courier = postal.NewCourier(&fakeUAA, userLoader, spaceLoader, templateLoader, mailer)
    })

    Describe("Dispatch", func() {
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

                Context("when a template cannot be loaded", func() {
                    It("returns a TemplateLoadError", func() {
                        delete(fs.Files, env.RootPath+"/templates/user_body.text")

                        err := courier.Dispatch(writer, token, "user-123", postal.IsUser, options)

                        Expect(err).To(BeAssignableToTypeOf(postal.TemplateLoadError("")))
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
                    courier = postal.NewCourier(&fakeUAA, userLoader, spaceLoader, templateLoader, mailer)
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
})
