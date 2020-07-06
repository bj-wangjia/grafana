package migrations

import (
	. "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func AddExperimentMigrations(mg *Migrator) {
	experimentV1 := Table{
		Name: "experiment",
		Columns: []*Column{
			{Name: "id", Type: DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "name", Type: DB_Text, Nullable: false},
			{Name: "value", Type: DB_Text, Nullable: false},
			{Name: "status", Type: DB_BigInt, Nullable: false},
			{Name: "created", Type: DB_DateTime, Nullable: false},
			{Name: "updated", Type: DB_DateTime, Nullable: false},
			{Name: "created_by", Type: DB_Text, Nullable: false},
			{Name: "updated_by", Type: DB_Text, Nullable: false},
		},
		Indices: []*Index{
			{Cols: []string{"id"}, Type: IndexType},
			{Cols: []string{"status"}, Type: IndexType},
			{Cols: []string{"name"}, Type: UniqueIndex},
		},
	}

	// create table
	mg.AddMigration("drop experiment table v3", NewDropTableMigration("experiment"))
	mg.AddMigration("create experiment table v4", NewAddTableMigration(experimentV1))

	// create indices
	mg.AddMigration("add index experiment id ", NewAddIndexMigration(experimentV1, experimentV1.Indices[0]))
	mg.AddMigration("add index status name", NewAddIndexMigration(experimentV1, experimentV1.Indices[1]))
	mg.AddMigration("add index experiment name", NewAddIndexMigration(experimentV1, experimentV1.Indices[2]))

}
