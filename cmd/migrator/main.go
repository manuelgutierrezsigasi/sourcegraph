package main

import (
	"context"
	"fmt"
	"os"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/database/migration"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

func main() {
	if err := mainErr(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func mainErr(ctx context.Context) error {
	n := 3               // TODO - receive from flag
	up := true           // TODO - receive from flag
	dbName := "frontend" // TODO - receive from flag

	// TODO - get from static db map
	migrationsTable := "schema_migrations"
	fs := dbconn.Frontend.FS
	opts := dbconn.Opts{DSN: "postgres://sourcegraph@localhost:5432/sourcegraph", DBName: dbName, AppName: "migrator"}

	//
	//

	store, err := initializeStore(ctx, opts, migrationsTable)
	if err != nil {
		return err
	}

	version, ok, err := store.Version(ctx)
	if err != nil {
		return err
	}
	if !ok {
		//
		// TODO - special case this depending on args
		//

		return err
	}

	log15.Info("Checked current version", "version", version)

	//
	//

	migrationSpecs, err := migration.ReadMigrationSpecs(fs)
	if err != nil {
		return err
	}

	if up {
		for _, migration := range migrationSpecs.UpFrom(version, n) {
			log15.Info("Running up migration", "migrationID", migration.ID)

			if store.Up(ctx, migration); err != nil {
				return err
			}
		}
	} else {
		for _, migration := range migrationSpecs.DownFrom(version, n) {
			log15.Info("Running down migration", "migrationID", migration.ID)

			if store.Down(ctx, migration); err != nil {
				return err
			}
		}
	}

	return nil
}

func initializeStore(ctx context.Context, opts dbconn.Opts, migrationsTable string) (*migration.Store, error) {
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	db, err := dbconn.New(opts)
	if err != nil {
		return nil, err
	}

	store := migration.NewWithDB(
		db,
		migrationsTable,
		observationContext,
	)

	if err := store.EnsureSchemaTable(ctx); err != nil {
		return nil, err
	}

	return store, nil
}
