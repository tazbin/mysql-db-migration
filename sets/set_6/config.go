package set6

import (
	"db-migration/sets"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		TargetTableName: "lk_module_uw_needs_2",
		SourceTableName: "events",
		PivotTableName:  "mapping_lk_module_uw_needs_event",

		PivotTableColumns: map[string]string{
			"need_id":  "BIGINT UNSIGNED NOT NULL",
			"event_id": "BIGINT UNSIGNED NOT NULL",
		},

		/* target table modification starts */
		NewColumnsForTargetTable: map[string]string{
			"event_id":             "BIGINT UNSIGNED",
			"need_date_added_ts":   "TIMESTAMP",
			"need_date_updated_ts": "TIMESTAMP",
			"is_migrated":          "TINYINT(1) DEFAULT 0",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"need_domain_id": "BIGINT UNSIGNED",
			"need_city":      "VARCHAR(255)",
			"need_state":     "VARCHAR(255)",
			"need_postal":    "VARCHAR(255)",
			"need_country":   "VARCHAR(255)",
			"need_body":      "TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		},
		/* target table modification ends */

		/* source table modification starts */
		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: `
			INSERT INTO lk_module_uw_needs_2 (
				event_id,
				need_domain_id,
				need_address,
				need_city,
				need_state,
				need_postal,
				need_country,
				need_title,
				need_body,
				need_public,
				need_date_added_ts,
				need_date_updated_ts,
				need_status,
				is_migrated
				)
			SELECT
				events.id,
				mapping_lk_domains_sites.domain_id,
				events.address,
				events.city,
				events.` + "`state`" + `,
				events.postal_code,
				events.country,
				events.` + "`name`" + `,
				events.description,
				events.private,
				events.created_at,
				events.updated_at,
				events.status,
				1
			FROM
				events
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = events.site_id;
		`,

		UpdateSourceQuery: `
			UPDATE
				lk_module_uw_needs_2
				JOIN events ON lk_module_uw_needs_2.event_id = events.id
			SET
				events.migration_done = 1;
		`,

		InsertToPivotQuery: `
			INSERT INTO mapping_lk_module_uw_needs_event (
				need_id, 
				event_id
				)
			SELECT
				lk_module_uw_needs_2.need_id,
				lk_module_uw_needs_2.event_id
			FROM
				lk_module_uw_needs_2
			WHERE
				lk_module_uw_needs_2.is_migrated = 1;
		`,

		PivotTableMappingValidationQuery: `
			SELECT
				COUNT(*)
			FROM
				mapping_lk_module_uw_needs_event p
				JOIN lk_module_uw_needs_2 t ON p.need_id = t.need_id
				JOIN events s ON p.event_id = s.id;
		`,

		FieldLevelValidationQuery: `
			SELECT
				events.id
			FROM
				events
				JOIN lk_module_uw_needs_2 ON lk_module_uw_needs_2.event_id = events.id
			WHERE
				events.migration_done = 1
				AND lk_module_uw_needs_2.is_migrated = 1
				AND (
					NOT (BINARY lk_module_uw_needs_2.need_address <=> BINARY events.address)
					OR NOT (BINARY lk_module_uw_needs_2.need_city <=> BINARY events.city)
					OR NOT (BINARY lk_module_uw_needs_2.need_state <=> BINARY events.` + "`state`" + `)
					OR NOT (BINARY lk_module_uw_needs_2.need_postal <=> BINARY events.postal_code)
					OR NOT (BINARY lk_module_uw_needs_2.need_country <=> BINARY events.country)
					OR NOT (BINARY lk_module_uw_needs_2.need_title <=> BINARY events.` + "`name`" + `)
					OR NOT (BINARY lk_module_uw_needs_2.need_body <=> BINARY events.description)
					OR NOT (lk_module_uw_needs_2.need_public <=> events.private)
					OR NOT (DATE_FORMAT(lk_module_uw_needs_2.need_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(events.created_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
					OR NOT (DATE_FORMAT(lk_module_uw_needs_2.need_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(events.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
			-- 		OR NOT (lk_module_uw_needs_2.need_status <=> events.status)
				)
			LIMIT 3;
		`,

		/*
			SHOW FULL COLUMNS FROM lk_module_uw_needs_2 LIKE 'need_body'; -- utf8mb3_general_ci
			SHOW FULL COLUMNS FROM events LIKE 'description'; -- utf8mb4_unicode_ci
		*/

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM lk_module_uw_needs_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "lk_module_uw_needs_2",
			},
			{
				Query:       "UPDATE events SET migration_done = 0 WHERE migration_done = 1",
				Description: "🗑️  Deleted migration_done column from",
				Table:       "events",
			},
			{
				Query:       "DELETE FROM mapping_lk_module_uw_needs_event",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_lk_module_uw_needs_event",
			},
		},
	}
}
