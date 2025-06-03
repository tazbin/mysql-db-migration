package set10

import (
	"db-migration/sets"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		TargetTableName: "suppressions_2",
		SourceTableName: "email_suppressions",
		PivotTableName:  "mapping_suppressions_email_suppressions",

		PivotTableColumns: map[string]string{
			"suppression_id":       "BIGINT UNSIGNED NOT NULL",
			"email_suppression_id": "BIGINT UNSIGNED NOT NULL",
		},

		/* target table modification starts */
		NewColumnsForTargetTable: map[string]string{
			"email_suppression_id": "BIGINT UNSIGNED",
			"date_added_ts":        "TIMESTAMP",
			"is_migrated":          "TINYINT(1) DEFAULT 0",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"note": "TEXT",
		},
		/* target table modification ends */

		/* source table modification starts */
		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: `
			INSERT INTO suppressions_2 (
				email_suppression_id, 
				email, 
				` + "`type`" + `, 
				note, 
				date_added_ts, 
				is_migrated
				)
			SELECT
				id,
				email,
				` + "`type`" + `,
				message,
				created_at,
				1
			FROM
				email_suppressions;
		`,

		UpdateSourceQuery: `
			UPDATE
				email_suppressions
				JOIN suppressions_2 ON suppressions_2.email_suppression_id = email_suppressions.id
			SET
				email_suppressions.migration_done = 1;
		`,

		InsertToPivotQuery: `
			INSERT INTO mapping_suppressions_email_suppressions (
				suppression_id, 
				email_suppression_id
				)
			SELECT
				suppressions_2.id,
				suppressions_2.email_suppression_id
			FROM
				suppressions_2
			WHERE
				suppressions_2.is_migrated = 1;
		`,

		PivotTableMappingValidationQuery: `
			SELECT
				COUNT(*)
			FROM
				mapping_suppressions_email_suppressions p
				JOIN suppressions_2 t ON p.suppression_id = t.id
				JOIN email_suppressions s ON p.email_suppression_id = s.id;
		`,

		FieldLevelValidationQuery: `
			SELECT
				email_suppressions.id,
				NOT (BINARY suppressions_2.email <=> BINARY email_suppressions.email) AS email_mismatch,
				NOT (BINARY suppressions_2.` + "`type`" + ` <=> BINARY email_suppressions.` + "`type`" + `) as type_mismatch,
				NOT (BINARY suppressions_2.note <=> BINARY email_suppressions.message) as note_mismatch,	
				NOT(DATE_FORMAT(suppressions_2.date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(email_suppressions.created_at, '%%Y-%%m-%%d %%H:%%i:%%s')) as date_added_mismatch
			FROM
				email_suppressions
				JOIN suppressions_2 ON suppressions_2.email_suppression_id = email_suppressions.id			
				WHERE
				email_suppressions.migration_done = 1
				AND suppressions_2.is_migrated = 1
				AND(
				NOT (BINARY suppressions_2.email <=> BINARY email_suppressions.email) 
				OR	NOT (BINARY suppressions_2.` + "`type`" + ` <=> BINARY email_suppressions.` + "`type`" + `) -- enum	
				OR	NOT (BINARY suppressions_2.note <=> BINARY email_suppressions.message) 			
				OR NOT(DATE_FORMAT(suppressions_2.date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(email_suppressions.created_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
			);
		`,

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM suppressions_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "suppressions_2",
			},
			{
				Query:       "UPDATE email_suppressions SET migration_done = 0 WHERE migration_done = 1",
				Description: "🗑️  Deleted migration_done column from",
				Table:       "email_suppressions",
			},
			{
				Query:       "DELETE FROM mapping_suppressions_email_suppressions",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_suppressions_email_suppressions",
			},
		},
	}
}
