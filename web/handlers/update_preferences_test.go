package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/notifications/application"
	"github.com/cloudfoundry-incubator/notifications/fakes"
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/web/handlers"
	"github.com/cloudfoundry-incubator/notifications/web/params"
	"github.com/cloudfoundry-incubator/notifications/web/services"
	"github.com/dgrijalva/jwt-go"
	"github.com/ryanmoran/stack"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UpdatePreferences", func() {
	Describe("Execute", func() {
		var handler handlers.UpdatePreferences
		var writer *httptest.ResponseRecorder
		var request *http.Request
		var updater *fakes.PreferenceUpdater
		var errorWriter *fakes.ErrorWriter
		var conn *fakes.DBConn
		var context stack.Context

		BeforeEach(func() {
			conn = fakes.NewDBConn()
			builder := services.NewPreferencesBuilder()

			builder.Add(models.Preference{
				ClientID: "raptors",
				KindID:   "door-opening",
				Email:    false,
			})
			builder.Add(models.Preference{
				ClientID: "raptors",
				KindID:   "feeding-time",
				Email:    true,
			})
			builder.Add(models.Preference{
				ClientID: "dogs",
				KindID:   "barking",
				Email:    false,
			})
			builder.GlobalUnsubscribe = true

			body, err := json.Marshal(builder)
			if err != nil {
				panic(err)
			}

			request, err = http.NewRequest("PATCH", "/user_preferences", bytes.NewBuffer(body))
			if err != nil {
				panic(err)
			}

			tokenHeader := map[string]interface{}{
				"alg": "FAST",
			}
			tokenClaims := map[string]interface{}{
				"user_id": "correct-user",
				"exp":     int64(3404281214),
			}

			rawToken := fakes.BuildToken(tokenHeader, tokenClaims)
			request.Header.Set("Authorization", "Bearer "+rawToken)

			token, err := jwt.Parse(rawToken, func(*jwt.Token) (interface{}, error) {
				return []byte(application.UAAPublicKey), nil
			})

			context = stack.NewContext()
			context.Set("token", token)

			errorWriter = fakes.NewErrorWriter()
			updater = fakes.NewPreferenceUpdater()
			fakeDatabase := fakes.NewDatabase()
			handler = handlers.NewUpdatePreferences(updater, errorWriter, fakeDatabase)
			writer = httptest.NewRecorder()
		})

		It("Passes The Correct Arguments to PreferenceUpdater Execute", func() {
			handler.Execute(writer, request, conn, context)
			Expect(len(updater.ExecuteArguments)).To(Equal(3))

			preferencesArguments := updater.ExecuteArguments[0]

			Expect(preferencesArguments).To(ContainElement(models.Preference{
				ClientID: "raptors",
				KindID:   "door-opening",
				Email:    false,
			}))
			Expect(preferencesArguments).To(ContainElement(models.Preference{
				ClientID: "raptors",
				KindID:   "feeding-time",
				Email:    true,
			}))
			Expect(preferencesArguments).To(ContainElement(models.Preference{
				ClientID: "dogs",
				KindID:   "barking",
				Email:    false,
			}))

			Expect(updater.ExecuteArguments[1]).To(BeTrue())
			Expect(updater.ExecuteArguments[2]).To(Equal("correct-user"))
		})

		It("Returns a 204 status code when the Preference object does not error", func() {
			handler.Execute(writer, request, conn, context)

			Expect(writer.Code).To(Equal(http.StatusNoContent))
		})

		Context("Failure cases", func() {
			It("returns an error when the clients key is missing", func() {
				jsonBody := `{"raptor-client": {"containment-unit-breach": {"email": false}}}`

				request, err := http.NewRequest("PATCH", "/user_preferences", bytes.NewBuffer([]byte(jsonBody)))
				if err != nil {
					panic(err)
				}

				tokenHeader := map[string]interface{}{
					"alg": "FAST",
				}
				tokenClaims := map[string]interface{}{
					"user_id": "correct-user",
					"exp":     int64(3404281214),
				}

				rawToken := fakes.BuildToken(tokenHeader, tokenClaims)
				request.Header.Set("Authorization", "Bearer "+rawToken)

				handler.Execute(writer, request, conn, context)

				Expect(errorWriter.Error).ToNot(BeNil())
				Expect(errorWriter.Error).To(BeAssignableToTypeOf(params.ValidationError{}))
				Expect(conn.BeginWasCalled).To(BeFalse())
				Expect(conn.CommitWasCalled).To(BeFalse())
				Expect(conn.RollbackWasCalled).To(BeFalse())
			})

			Context("preferenceUpdater.Execute errors", func() {
				Context("when the user_id claim is not present in the token", func() {
					It("Writes a MissingUserTokenError to the error writer", func() {
						tokenHeader := map[string]interface{}{
							"alg": "FAST",
						}

						tokenClaims := map[string]interface{}{}

						request, err := http.NewRequest("PATCH", "/user_preferences", nil)
						if err != nil {
							panic(err)
						}

						token, err := jwt.Parse(fakes.BuildToken(tokenHeader, tokenClaims), func(token *jwt.Token) (interface{}, error) {
							return []byte(application.UAAPublicKey), nil
						})

						context = stack.NewContext()
						context.Set("token", token)

						handler.ServeHTTP(writer, request, context)
						Expect(errorWriter.Error).To(BeAssignableToTypeOf(handlers.MissingUserTokenError("")))
						Expect(conn.BeginWasCalled).To(BeFalse())
						Expect(conn.CommitWasCalled).To(BeFalse())
						Expect(conn.RollbackWasCalled).To(BeFalse())
					})
				})

				It("delegates MissingKindOrClientErrors as params.ValidationError to the ErrorWriter", func() {
					updater.ExecuteError = services.MissingKindOrClientError("BOOM!")

					handler.Execute(writer, request, conn, context)

					Expect(errorWriter.Error).To(Equal(params.ValidationError([]string{"BOOM!"})))

					Expect(conn.BeginWasCalled).To(BeTrue())
					Expect(conn.CommitWasCalled).To(BeFalse())
					Expect(conn.RollbackWasCalled).To(BeTrue())
				})

				It("delegates CriticalKindErrors as params.ValidationError to the ErrorWriter", func() {
					updater.ExecuteError = services.CriticalKindError("BOOM!")

					handler.Execute(writer, request, conn, context)

					Expect(errorWriter.Error).To(Equal(params.ValidationError([]string{"BOOM!"})))

					Expect(conn.BeginWasCalled).To(BeTrue())
					Expect(conn.CommitWasCalled).To(BeFalse())
					Expect(conn.RollbackWasCalled).To(BeTrue())
				})

				It("delegates other errors to the ErrorWriter", func() {
					updater.ExecuteError = errors.New("BOOM!")

					handler.Execute(writer, request, conn, context)

					Expect(errorWriter.Error).To(Equal(errors.New("BOOM!")))

					Expect(conn.BeginWasCalled).To(BeTrue())
					Expect(conn.CommitWasCalled).To(BeFalse())
					Expect(conn.RollbackWasCalled).To(BeTrue())
				})
			})

			It("delegates json validation errors to the ErrorWriter", func() {
				requestBody, err := json.Marshal(map[string]interface{}{
					"something": true,
				})
				if err != nil {
					panic(err)
				}

				request, err = http.NewRequest("PATCH", "/user_preferences", bytes.NewBuffer(requestBody))
				if err != nil {
					panic(err)
				}

				handler.Execute(writer, request, conn, context)

				Expect(errorWriter.Error).To(BeAssignableToTypeOf(params.ValidationError{}))
				Expect(conn.BeginWasCalled).To(BeFalse())
				Expect(conn.CommitWasCalled).To(BeFalse())
				Expect(conn.RollbackWasCalled).To(BeFalse())
			})

			It("delegates validation errors to the error writer", func() {
				requestBody, err := json.Marshal(map[string]map[string]map[string]map[string]interface{}{
					"clients": {
						"client-id": {
							"kind-id": {},
						},
					},
				})
				if err != nil {
					panic(err)
				}

				request, err = http.NewRequest("PATCH", "/user_preferences", bytes.NewBuffer(requestBody))
				if err != nil {
					panic(err)
				}

				handler.Execute(writer, request, conn, context)

				Expect(errorWriter.Error).To(BeAssignableToTypeOf(params.ValidationError{}))
				Expect(conn.BeginWasCalled).To(BeFalse())
				Expect(conn.CommitWasCalled).To(BeFalse())
				Expect(conn.RollbackWasCalled).To(BeFalse())
			})

			It("delegates transaction errors to the error writer", func() {
				conn.CommitError = "transaction error, oh no"
				handler.Execute(writer, request, conn, context)

				Expect(errorWriter.Error).To(BeAssignableToTypeOf(models.NewTransactionCommitError("transaction error, oh no")))
				Expect(conn.BeginWasCalled).To(BeTrue())
				Expect(conn.CommitWasCalled).To(BeTrue())
				Expect(conn.RollbackWasCalled).To(BeFalse())
			})
		})
	})
})
