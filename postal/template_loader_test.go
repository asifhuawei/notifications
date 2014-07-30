package postal_test

import (
    "strings"

    "github.com/cloudfoundry-incubator/notifications/postal"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("TemplateLoader", func() {
    var loader postal.TemplateLoader

    BeforeEach(func() {
        loader = postal.NewTemplateLoader()
    })

    Describe("Load", func() {
        Context("when subject is not set in the params", func() {
            It("returns the subject.missing template", func() {
                loader.ReadFile = func(path string) (string, error) {
                    if strings.Contains(path, "missing") {
                        return "the missing subject", nil
                    }
                    return "incorrect", nil
                }

                loader.FileExists = func(path string) bool {
                    return false
                }

                subject := ""

                templates, err := loader.Load(subject, postal.IsSpace)
                if err != nil {
                    panic(err)
                }

                Expect(templates.Subject).To(Equal("the missing subject"))
            })
        })

        Context("when subject is set in the params", func() {
            It("returns the subject.provided template", func() {
                loader.ReadFile = func(path string) (string, error) {
                    if strings.Contains(path, "provided") {
                        return "the provided subject", nil
                    }
                    return "incorrect", nil
                }

                loader.FileExists = func(path string) bool {
                    return false
                }

                subject := "is provided"

                templates, err := loader.Load(subject, postal.IsSpace)
                if err != nil {
                    panic(err)
                }

                Expect(templates.Subject).To(Equal("the provided subject"))
            })
        })

        Context("notificationType is IsSpace", func() {
            It("returns the space templates", func() {
                loader.ReadFile = func(path string) (string, error) {
                    if strings.Contains(path, "space") && strings.Contains(path, "text") {
                        return "space plain text", nil
                    }
                    if strings.Contains(path, "space") && strings.Contains(path, "html") {
                        return "space html code", nil
                    }
                    return "incorrect", nil
                }

                loader.FileExists = func(path string) bool {
                    return false
                }

                templates, err := loader.Load("", postal.IsSpace)
                if err != nil {
                    panic(err)
                }

                Expect(templates.Text).To(Equal("space plain text"))
                Expect(templates.HTML).To(Equal("space html code"))
            })
        })

        Context("notificationType is IsSpace", func() {
            It("returns the user templates", func() {
                loader.ReadFile = func(path string) (string, error) {
                    if strings.Contains(path, "user") && strings.Contains(path, "text") {
                        return "user plain text", nil
                    }
                    if strings.Contains(path, "user") && strings.Contains(path, "html") {
                        return "user html code", nil
                    }
                    return "incorrect", nil
                }

                loader.FileExists = func(path string) bool {
                    return false
                }

                templates, err := loader.Load("", postal.IsUser)
                if err != nil {
                    panic(err)
                }

                Expect(templates.Text).To(Equal("user plain text"))
                Expect(templates.HTML).To(Equal("user html code"))
            })
        })
    })

    Describe("LoadTemplate", func() {
        BeforeEach(func() {
            loader.ReadFile = func(path string) (string, error) {
                switch {
                case strings.Contains(path, "overrides"):
                    return "override text", nil
                default:
                    return "default text", nil
                }
            }
        })

        Context("when there are no template overrides", func() {
            It("loads the templates from the default location", func() {
                loader.FileExists = func(path string) bool {
                    return false
                }

                text, err := loader.LoadTemplate("user_body.text")
                if err != nil {
                    panic(err)
                }
                Expect(text).To(Equal("default text"))
            })
        })

        Context("when a template has an override set", func() {
            It("replaces the default template with the user provided one", func() {
                loader.FileExists = func(path string) bool {
                    return true
                }

                text, err := loader.LoadTemplate("user_body.text")
                if err != nil {
                    panic(err)
                }

                Expect(text).To(Equal("override text"))
            })
        })
    })
})
