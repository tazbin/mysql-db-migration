package set7

import (
	"db-migration/sets"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		TargetTableName: "lk_module_uw_needs_schedule_2",
		SourceTableName: "shifts",
		PivotTableName:  "mapping_lk_module_uw_needs_schedule_shift",

		PivotTableColumns: map[string]string{
			"needs_schedule_id": "BIGINT UNSIGNED NOT NULL",
			"shift_id":          "BIGINT UNSIGNED NOT NULL",
		},

		/* target table modification starts */
		NewColumnsForTargetTable: map[string]string{
			"shift_id":            "BIGINT UNSIGNED",
			"sch_date_start_ts":   "DATE",
			"sch_time_start_ts":   "TIME",
			"sch_date_end_ts":     "DATE",
			"sch_time_end_ts":     "TIME",
			"sch_date_added_ts":   "TIMESTAMP",
			"sch_date_updated_ts": "TIMESTAMP",
			"is_migrated":         "TINYINT(1) DEFAULT 0",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"sch_domain_id": "BIGINT UNSIGNED",
			"sch_need_id":   "BIGINT UNSIGNED",
		},
		/* target table modification ends */

		/* source table modification starts */
		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: `
			INSERT INTO lk_module_uw_needs_schedule_2 (
				shift_id,
				sch_domain_id,
				sch_need_id,
				sch_slots,
				sch_date_start_ts,
				sch_time_start_ts,
				sch_date_end_ts,
				sch_time_end_ts,
				sch_date_added_ts,
				sch_date_updated_ts,
				is_migrated
			)
			SELECT
				shifts.id,
				mapping_lk_domains_sites.domain_id,
				mapping_lk_module_uw_needs_event.need_id,
				shifts.capacity,
				DATE(CONVERT_TZ(shifts.starts_at, 'UTC', 'Asia/Dhaka')) AS starts_date_at,
				TIME(CONVERT_TZ(shifts.starts_at, 'UTC', 'Asia/Dhaka')) AS starts_time_at,
				DATE(CONVERT_TZ(shifts.ends_at, 'UTC', 'Asia/Dhaka')) AS ends_date_at,
				TIME(CONVERT_TZ(shifts.ends_at, 'UTC', 'Asia/Dhaka')) AS ends_time_at,
				shifts.created_at,
				shifts.updated_at,
				1
			FROM
				shifts
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = shifts.site_id
				JOIN mapping_lk_module_uw_needs_event ON mapping_lk_module_uw_needs_event.event_id = shifts.event_id;
		`,

		UpdateSourceQuery: `
			UPDATE
				shifts
				JOIN lk_module_uw_needs_schedule_2 ON lk_module_uw_needs_schedule_2.shift_id = shifts.id
			SET
				shifts.migration_done = 1;
		`,

		InsertToPivotQuery: `
			INSERT INTO mapping_lk_module_uw_needs_schedule_shift (
				needs_schedule_id, 
				shift_id
				)
			SELECT
				lk_module_uw_needs_schedule_2.sch_id,
				lk_module_uw_needs_schedule_2.shift_id
			FROM
				lk_module_uw_needs_schedule_2
			WHERE
				lk_module_uw_needs_schedule_2.is_migrated = 1;
		`,

		PivotTableMappingValidationQuery: `
			SELECT
				COUNT(*)
			FROM
				mapping_lk_module_uw_needs_schedule_shift p
				JOIN lk_module_uw_needs_schedule_2 t ON p.needs_schedule_id = t.sch_id
				JOIN shifts s ON p.shift_id = s.id;
		`,

		FieldLevelValidationQuery: `
			SELECT
				shifts.id,
				NOT(lk_module_uw_needs_schedule_2.sch_domain_id <=> mapping_lk_domains_sites.domain_id) AS domain_mismatch,
				NOT(lk_module_uw_needs_schedule_2.sch_need_id <=> mapping_lk_module_uw_needs_event.need_id) AS need_id_mismatch,
				NOT(lk_module_uw_needs_schedule_2.sch_slots <=> shifts.capacity) AS slots_mismatch,
				NOT(lk_module_uw_needs_schedule_2.sch_date_start_ts <=> DATE(CONVERT_TZ(shifts.starts_at, 'UTC', 'Asia/Dhaka'))) AS date_start_mismatch,
				NOT(lk_module_uw_needs_schedule_2.sch_time_start_ts <=> TIME(CONVERT_TZ(shifts.starts_at, 'UTC', 'Asia/Dhaka'))) AS time_start_mismatch,
				NOT(lk_module_uw_needs_schedule_2.sch_date_end_ts <=> DATE(CONVERT_TZ(shifts.ends_at, 'UTC', 'Asia/Dhaka'))) AS date_end_mismatch,
				NOT(lk_module_uw_needs_schedule_2.sch_time_end_ts <=> TIME(CONVERT_TZ(shifts.ends_at, 'UTC', 'Asia/Dhaka'))) AS time_end_mismatch,
				NOT(DATE_FORMAT(lk_module_uw_needs_schedule_2.sch_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(shifts.created_at, '%%Y-%%m-%%d %%H:%%i:%%s')) AS date_added_mismatch,
				NOT(DATE_FORMAT(lk_module_uw_needs_schedule_2.sch_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(shifts.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s')) AS date_updated_mismatch
			FROM
				shifts
				JOIN lk_module_uw_needs_schedule_2 ON lk_module_uw_needs_schedule_2.shift_id = shifts.id
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = shifts.site_id
				JOIN mapping_lk_module_uw_needs_event ON mapping_lk_module_uw_needs_event.event_id = shifts.event_id
			WHERE
				shifts.migration_done = 1
				AND lk_module_uw_needs_schedule_2.is_migrated = 1
				AND(NOT(lk_module_uw_needs_schedule_2.sch_domain_id <=> mapping_lk_domains_sites.domain_id)
					OR NOT(lk_module_uw_needs_schedule_2.sch_need_id <=> mapping_lk_module_uw_needs_event.need_id)
					OR NOT(lk_module_uw_needs_schedule_2.sch_slots <=> shifts.capacity)

					OR NOT(lk_module_uw_needs_schedule_2.sch_date_start_ts <=> DATE(CONVERT_TZ(shifts.starts_at, 'UTC', 'Asia/Dhaka')))
					OR NOT(lk_module_uw_needs_schedule_2.sch_time_start_ts <=> TIME(CONVERT_TZ(shifts.starts_at, 'UTC', 'Asia/Dhaka')))

					OR NOT(lk_module_uw_needs_schedule_2.sch_date_end_ts <=> DATE(CONVERT_TZ(shifts.ends_at, 'UTC', 'Asia/Dhaka')))
					OR NOT(lk_module_uw_needs_schedule_2.sch_time_end_ts <=> TIME(CONVERT_TZ(shifts.ends_at, 'UTC', 'Asia/Dhaka')))

					OR NOT(DATE_FORMAT(lk_module_uw_needs_schedule_2.sch_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(shifts.created_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
					OR NOT(DATE_FORMAT(lk_module_uw_needs_schedule_2.sch_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(shifts.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
			);
		`,

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM lk_module_uw_needs_schedule_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "lk_module_uw_needs_schedule_2",
			},
			{
				Query:       "UPDATE shifts SET migration_done = 0 WHERE migration_done = 1",
				Description: "🗑️  Deleted migration_done column from",
				Table:       "shifts",
			},
			{
				Query:       "DELETE FROM mapping_lk_module_uw_needs_schedule_shift",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_lk_module_uw_needs_schedule_shift",
			},
		},
	}
}
