package set4

import (
	"db-migration/sets"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		TargetTableName: "lk_user_groups_2",
		SourceTableName: "`groups`",
		PivotTableName:  "mapping_lk_user_groups_groups",

		PivotTableColumns: map[string]string{
			"user_group_id": "BIGINT UNSIGNED NOT NULL",
			"group_id":      "BIGINT UNSIGNED NOT NULL",
		},

		/* target table modification starts */
		NewColumnsForTargetTable: map[string]string{
			"group_id":           "BIGINT UNSIGNED",
			"ug_date_added_ts":   "TIMESTAMP",
			"ug_date_updated_ts": "TIMESTAMP",
			"is_migrated":        "TINYINT(1) DEFAULT 0",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"ug_domain_id": "BIGINT UNSIGNED",
		},
		/* target table modification ends */

		/* source table modification starts */
		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: `
			INSERT INTO lk_user_groups_2 (
				group_id,
				ug_domain_id,
				ug_title,
				ug_date_added_ts,
				ug_date_updated_ts,
				is_migrated
				)
			SELECT
				` + "`groups`" + `.id,
				mapping_lk_domains_sites.domain_id,
				` + "`groups`" + `.name,
				` + "`groups`" + `.created_at,
				` + "`groups`" + `.updated_at,
				1
			FROM
				` + "`groups`" + `
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = ` + "`groups`" + `.site_id;
		`,

		UpdateSourceQuery: `
			UPDATE
				` + "`groups`" + `
				JOIN lk_user_groups_2 ON lk_user_groups_2.group_id = ` + "`groups`" + `.id
			SET
				` + "`groups`" + `.migration_done = 1;
		`,

		InsertToPivotQuery: `
			INSERT INTO mapping_lk_user_groups_groups (
				user_group_id, 
				group_id
				)
			SELECT
				lk_user_groups_2.ug_id,
				lk_user_groups_2.group_id
			FROM
				lk_user_groups_2
			WHERE
				lk_user_groups_2.is_migrated = 1;
		`,

		PivotTableMappingValidationQuery: `
			SELECT
				COUNT(*)
			FROM
				mapping_lk_user_groups_groups p
				JOIN lk_user_groups_2 t ON p.user_group_id = t.ug_id
				JOIN ` + "`groups`" + ` s ON p.group_id = s.id;
		`,

		FieldLevelValidationQuery: `
			SELECT
				` + "`groups`" + `.id
			FROM
				` + "`groups`" + `
				JOIN lk_user_groups_2 ON lk_user_groups_2.group_id = ` + "`groups`" + `.id
				JOIN mapping_lk_domains_sites ON ` + "`groups`" + `.site_id = mapping_lk_domains_sites.site_id
			WHERE
				` + "`groups`" + `.migration_done = 1
				AND lk_user_groups_2.is_migrated = 1
				AND(NOT(BINARY lk_user_groups_2.ug_title <=> BINARY ` + "`groups`" + `.name)
					OR NOT(lk_user_groups_2.ug_domain_id <=> mapping_lk_domains_sites.domain_id)
					OR NOT(DATE_FORMAT(lk_user_groups_2.ug_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(` + "`groups`" + `.created_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
					OR NOT(DATE_FORMAT(lk_user_groups_2.ug_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(` + "`groups`" + `.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s')))
			LIMIT 3;
		`,

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM lk_user_groups_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "lk_user_groups_2",
			},
			{
				// Query:       "UPDATE " + "`groups`" + " SET migration_done = 0 WHERE migration_done = 1",
				Query:       "ALTER TABLE " + "`groups`" + " DROP COLUMN migration_done",
				Description: "🗑️  Deleted migration_done column from",
				Table:       "groups",
			},
			{
				Query:       "DELETE FROM mapping_lk_user_groups_groups",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_lk_user_groups_groups",
			},
		},
	}
}
