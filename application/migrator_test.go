package application_test

import (
	"github.com/cloudfoundry-incubator/notifications/application"
	"github.com/cloudfoundry-incubator/notifications/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Migrator", func() {
	Describe("Migrate", func() {
		var migrator application.Migrator
		var provider *fakes.PersistenceProvider
		var database *fakes.Database

		BeforeEach(func() {
			database = fakes.NewDatabase()
			provider = fakes.NewPersistenceProvider(database)
		})

		Context("when configured to run migrations", func() {
			BeforeEach(func() {
				migrator = application.NewMigrator(provider, true)
			})

			It("calls the Database function on the persistence provider", func() {
				migrator.Migrate()

				Expect(provider.DatabaseWasCalled).To(BeTrue())
			})

			It("calls the Queue function on the persistence provider", func() {
				migrator.Migrate()

				Expect(provider.QueueWasCalled).To(BeTrue())
			})

			It("seeds the database", func() {
				migrator.Migrate()

				Expect(database.SeedWasCalled).To(BeTrue())
			})
		})

		Context("when configured to skip migrations", func() {
			BeforeEach(func() {
				migrator = application.NewMigrator(provider, false)
			})

			It("skips the Database function on the persistence provider", func() {
				migrator.Migrate()

				Expect(provider.DatabaseWasCalled).To(BeFalse())
			})

			It("skips the Queue function on the persistence provider", func() {
				migrator.Migrate()

				Expect(provider.QueueWasCalled).To(BeFalse())
			})

			It("does not seed the database", func() {
				migrator.Migrate()

				Expect(database.SeedWasCalled).To(BeFalse())
			})
		})
	})
})
