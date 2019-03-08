package db

import (
	"github.com/go-pg/migrations"
	"github.com/go-seatbelt/seatbelt/internal/trace"
	"github.com/sirupsen/logrus"
)

// Migrate executes the intended migration action on the provided DB. It will
// also report any changes in the migration version, or if there are no
// changes.
func Migrate(action string, db migrations.DB) {
	if _, _, err := migrations.Run(db, "init"); err != nil {
		logrus.Info("Initial migration has already been run")
	}

	oldVersion, newVersion, err := migrations.Run(db, action)
	if err != nil {
		logrus.Fatalf("%s: Failed to migrate database: %+v", trace.Getfl(), err)
	}

	if newVersion != oldVersion {
		logrus.Infof("Migrated database from version %d to %d", oldVersion, newVersion)
	} else {
		logrus.Infof("No migrations to execute, current version is %d", oldVersion)
	}
}
