package set9

import (
	"db-migration/sets"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		TargetTableName: "lk_module_uw_timetrack_2",
		SourceTableName: "time_entries",
		PivotTableName:  "mapping_lk_module_uw_timetrack_time_entries",

		PivotTableColumns: map[string]string{
			"timetrack_id":    "BIGINT UNSIGNED NOT NULL",
			"time_entries_id": "BIGINT UNSIGNED NOT NULL",
		},

		/* target table modification starts */
		NewColumnsForTargetTable: map[string]string{
			"time_entries_id":      "BIGINT UNSIGNED",
			"hour_date_added_ts":   "TIMESTAMP",
			"hour_date_updated_ts": "TIMESTAMP",
			"is_migrated":          "TINYINT(1) DEFAULT 0",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"hour_domain_id":   "BIGINT UNSIGNED",
			"hour_user_id":     "BIGINT UNSIGNED",
			"hour_item_id":     "BIGINT UNSIGNED",
			"hour_response_id": "BIGINT UNSIGNED",
		},
		/* target table modification ends */

		/* source table modification starts */
		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: `
			INSERT INTO lk_module_uw_timetrack_2 (
				lk_module_uw_timetrack_2.time_entries_id,
				lk_module_uw_timetrack_2.hour_domain_id,
				lk_module_uw_timetrack_2.hour_user_id,
				lk_module_uw_timetrack_2.hour_item_id,
				lk_module_uw_timetrack_2.hour_response_id,
				lk_module_uw_timetrack_2.hour_date_added_ts,
				lk_module_uw_timetrack_2.hour_date_updated_ts,
				lk_module_uw_timetrack_2.is_migrated
				)
			SELECT
				time_entries.id,
				mapping_lk_domains_sites.domain_id,
				mapping_lk_users_members.user_id,
				mapping_lk_module_uw_needs_schedule_shift.needs_schedule_id,
				mapping_lk_module_uw_needs_responses_registrations.needs_response_id,
				time_entries.created_at,
				time_entries.updated_at,
				1
			FROM
				time_entries
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = time_entries.site_id
				JOIN mapping_lk_users_members ON mapping_lk_users_members.member_id = time_entries.member_id
				JOIN mapping_lk_module_uw_needs_schedule_shift ON mapping_lk_module_uw_needs_schedule_shift.shift_id = time_entries.shift_id
				JOIN mapping_lk_module_uw_needs_responses_registrations on mapping_lk_module_uw_needs_responses_registrations.registration_id = time_entries.registration_id;
		`,

		UpdateSourceQuery: `
			UPDATE
				time_entries
				JOIN lk_module_uw_timetrack_2 ON lk_module_uw_timetrack_2.time_entries_id = time_entries.id
			SET
				time_entries.migration_done = 1;
		`,

		InsertToPivotQuery: `
			INSERT INTO mapping_lk_module_uw_timetrack_time_entries (
				timetrack_id, 
				time_entries_id
				)
			SELECT
				lk_module_uw_timetrack_2.hour_id,
				lk_module_uw_timetrack_2.time_entries_id
			FROM
				lk_module_uw_timetrack_2
			WHERE
				lk_module_uw_timetrack_2.is_migrated = 1;
		`,

		PivotTableMappingValidationQuery: `
			SELECT
				COUNT(*)
			FROM
				mapping_lk_module_uw_timetrack_time_entries p
				JOIN lk_module_uw_timetrack_2 t ON p.timetrack_id = t.hour_id
				JOIN time_entries s ON p.time_entries_id = s.id;
		`,

		FieldLevelValidationQuery: `
			SELECT
				time_entries.id,
				NOT(DATE_FORMAT(lk_module_uw_timetrack_2.hour_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(time_entries.created_at, '%%Y-%%m-%%d %%H:%%i:%%s')) AS hour_date_added_mismatch,
				NOT(DATE_FORMAT(lk_module_uw_timetrack_2.hour_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(time_entries.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s')) AS hour_date_updated_mismatch
			FROM
				time_entries
				JOIN lk_module_uw_timetrack_2 ON lk_module_uw_timetrack_2.time_entries_id = time_entries.id
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = time_entries.site_id
				JOIN mapping_lk_users_members ON mapping_lk_users_members.member_id = time_entries.member_id
				JOIN mapping_lk_module_uw_needs_schedule_shift ON mapping_lk_module_uw_needs_schedule_shift.shift_id = time_entries.shift_id
				JOIN mapping_lk_module_uw_needs_responses_registrations on mapping_lk_module_uw_needs_responses_registrations.registration_id = time_entries.registration_id
			WHERE
				time_entries.migration_done = 1
				AND lk_module_uw_timetrack_2.is_migrated = 1
				AND(
					NOT(DATE_FORMAT(lk_module_uw_timetrack_2.hour_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(time_entries.created_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
					OR NOT(DATE_FORMAT(lk_module_uw_timetrack_2.hour_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(time_entries.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
			);
		`,

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM lk_module_uw_timetrack_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "lk_module_uw_timetrack_2",
			},
			{
				Query:       "UPDATE time_entries SET migration_done = 0 WHERE migration_done = 1",
				Description: "🗑️  Deleted migration_done column from",
				Table:       "time_entries",
			},
			{
				Query:       "DELETE FROM mapping_lk_module_uw_timetrack_time_entries",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_lk_module_uw_timetrack_time_entries",
			},
		},
	}
}
