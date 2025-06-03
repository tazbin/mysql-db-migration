package set8

import (
	"db-migration/sets"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		TargetTableName: "lk_module_uw_needs_responses_2",
		SourceTableName: "registrations",
		PivotTableName:  "mapping_lk_module_uw_needs_responses_registrations",

		PivotTableColumns: map[string]string{
			"needs_response_id": "BIGINT UNSIGNED NOT NULL",
			"registration_id":   "BIGINT UNSIGNED NOT NULL",
		},

		/* target table modification starts */
		NewColumnsForTargetTable: map[string]string{
			"registration_id":          "BIGINT UNSIGNED",
			"response_date_added_ts":   "TIMESTAMP",
			"response_date_updated_ts": "TIMESTAMP",
			"is_migrated":              "TINYINT(1) DEFAULT 0",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"response_domain_id": "BIGINT UNSIGNED",
			"response_sch_id":    "BIGINT UNSIGNED",
			"response_user_id":   "BIGINT UNSIGNED",
			"response_ug_id":     "BIGINT UNSIGNED",
		},
		/* target table modification ends */

		/* source table modification starts */
		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: `
			INSERT INTO lk_module_uw_needs_responses_2 (
				registration_id,
				response_domain_id,
				response_sch_id,
				response_user_id,
				response_ug_id,
				response_date_added_ts,
				response_date_updated_ts,
				is_migrated
				)
			SELECT
				registrations.id,
				mapping_lk_domains_sites.domain_id,
				mapping_lk_module_uw_needs_schedule_shift.needs_schedule_id,
				mapping_lk_users_members.user_id,
				mapping_lk_user_groups_groups.user_group_id,
				registrations.created_at,
				registrations.updated_at,
				1
			FROM
				registrations
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = registrations.site_id
				JOIN mapping_lk_module_uw_needs_schedule_shift ON mapping_lk_module_uw_needs_schedule_shift.shift_id = registrations.shift_id
				JOIN mapping_lk_users_members ON mapping_lk_users_members.member_id = registrations.member_id
				JOIN mapping_lk_user_groups_groups on mapping_lk_user_groups_groups.group_id = registrations.group_id;
		`,

		UpdateSourceQuery: `
			UPDATE
				registrations
				JOIN lk_module_uw_needs_responses_2 ON lk_module_uw_needs_responses_2.registration_id = registrations.id
			SET
				registrations.migration_done = 1;
		`,

		InsertToPivotQuery: `
			INSERT INTO mapping_lk_module_uw_needs_responses_registrations (
				needs_response_id, 
				registration_id
				)
			SELECT
				lk_module_uw_needs_responses_2.response_id,
				lk_module_uw_needs_responses_2.registration_id
			FROM
				lk_module_uw_needs_responses_2
			WHERE
				lk_module_uw_needs_responses_2.is_migrated = 1;
		`,

		PivotTableMappingValidationQuery: `
			SELECT
				COUNT(*)
			FROM
				mapping_lk_module_uw_needs_responses_registrations p
				JOIN lk_module_uw_needs_responses_2 t ON p.needs_response_id = t.response_id
				JOIN registrations s ON p.registration_id = s.id;
		`,

		FieldLevelValidationQuery: `
			SELECT
				registrations.id
			FROM
				registrations
				JOIN lk_module_uw_needs_responses_2 ON lk_module_uw_needs_responses_2.registration_id = registrations.id
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = registrations.site_id
				JOIN mapping_lk_module_uw_needs_schedule_shift ON mapping_lk_module_uw_needs_schedule_shift.shift_id = registrations.shift_id
				JOIN mapping_lk_users_members ON mapping_lk_users_members.member_id = registrations.member_id
				JOIN mapping_lk_user_groups_groups on mapping_lk_user_groups_groups.group_id = registrations.group_id
			WHERE
				registrations.migration_done = 1
				AND lk_module_uw_needs_responses_2.is_migrated = 1
				AND(NOT(lk_module_uw_needs_responses_2.response_domain_id <=> mapping_lk_domains_sites.domain_id)
					OR NOT(lk_module_uw_needs_responses_2.response_sch_id <=> mapping_lk_module_uw_needs_schedule_shift.needs_schedule_id)
					OR NOT(lk_module_uw_needs_responses_2.response_user_id <=> mapping_lk_users_members.user_id)
					OR NOT(lk_module_uw_needs_responses_2.response_ug_id <=> mapping_lk_user_groups_groups.user_group_id)
					OR NOT(DATE_FORMAT(lk_module_uw_needs_responses_2.response_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(registrations.created_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
					OR NOT(DATE_FORMAT(lk_module_uw_needs_responses_2.response_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(registrations.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
			);
		`,

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM lk_module_uw_needs_responses_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "lk_module_uw_needs_responses_2",
			},
			{
				Query:       "UPDATE registrations SET migration_done = 0 WHERE migration_done = 1",
				Description: "🗑️  Deleted migration_done column from",
				Table:       "registrations",
			},
			{
				Query:       "DELETE FROM mapping_lk_module_uw_needs_responses_registrations",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_lk_module_uw_needs_responses_registrations",
			},
		},
	}
}
