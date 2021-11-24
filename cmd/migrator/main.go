package main

import (
	"context"
	"fmt"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/database/migration"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

func main() {
	ctx := context.Background()

	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	migrationSpecs, err := migration.ReadMigrationSpecs(dbconn.Frontend.FS)
	if err != nil {
		panic(err.Error())
	}

	db, err := dbconn.New(dbconn.Opts{DSN: "postgres://sourcegraph@localhost:5432/sourcegraph", DBName: "frontend", AppName: "migrator"})
	if err != nil {
		panic(err.Error())
	}

	store := migration.NewWithDB(db, migrationSpecs, "schema_migrations", observationContext)

	if err := store.EnsureSchemaTable(ctx); err != nil {
		panic(err.Error())
	}

	version, ok, err := store.Version(ctx)
	if err != nil {
		panic(err.Error())
	}
	if !ok {
		panic("no version")
	}

	upMigrations := migrationSpecs.UpFrom(version, 0)
	// downMigrations := migrationSpecs.DownFrom(version, 0)
	fmt.Printf("Current version: %v\n", version)
	// fmt.Printf("Up migrations: %v\n", upMigrations)
	// fmt.Printf("Down migrations: %v\n", downMigrations)

	for _, migration := range upMigrations {
		fmt.Printf("GOING UP -> %d\n", migration.ID)

		if err := store.Up(ctx, migration); err != nil {
			panic(err.Error())
		}
	}

	fmt.Printf("Migration complete\n")
}
