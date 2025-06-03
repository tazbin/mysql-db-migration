package set3

import (
	"db-migration/sets"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		TargetTableName: "lk_users_2",
		SourceTableName: "members",
		PivotTableName:  "mapping_lk_users_members",

		PivotTableColumns: map[string]string{
			"user_id":   "BIGINT UNSIGNED NOT NULL",
			"member_id": "BIGINT UNSIGNED NOT NULL",
		},

		/* target table modification starts */
		NewColumnsForTargetTable: map[string]string{
			"member_id":            "BIGINT UNSIGNED",
			"user_date_added_ts":   "TIMESTAMP",
			"user_date_updated_ts": "TIMESTAMP",
			"is_migrated":          "TINYINT(1) DEFAULT 0",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"user_domain": "BIGINT UNSIGNED",
			// "user_fname":      "VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
			// "user_lname":      "VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
			// "user_email":      "VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
			"user_fname":      "VARCHAR(255)",
			"user_lname":      "VARCHAR(255)",
			"user_email":      "VARCHAR(255)",
			"user_phone_cell": "VARCHAR(255)",
		},

		/*
			SELECT first_name, last_name, email, mobile_phone, user_joined_at, user_updated_at FROM members WHERE id in (2429);
			SELECT user_fname, user_lname, user_email, user_phone_cell, user_date_updated_ts, user_date_updated_ts FROM lk_users_2 WHERE member_id in (2429);
			SHOW FULL COLUMNS FROM members LIKE 'email'; -- utf8mb4_unicode_ci
			SHOW FULL COLUMNS FROM lk_users_2 LIKE 'user_lname'; -- utf8mb3_general_ci, latin1_swedish_ci
		*/

		/* target table modification ends */

		/* source table modification starts */
		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: `
			INSERT INTO lk_users_2 (
				member_id,
				user_status,
				user_domain,
				user_fname,
				user_lname,
				user_email,
				user_phone_cell,
				user_date_added_ts,
				user_date_updated_ts,
				is_migrated
			)
			SELECT
				members.id,
				members.status,
				mapping_lk_domains_sites.domain_id,
				members.first_name,
				members.last_name,
				members.email,
				members.mobile_phone,
				members.user_joined_at,
				members.user_updated_at,
				1
			FROM
				members
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = members.site_id;
		`,

		UpdateSourceQuery: `
			UPDATE
				members
				JOIN lk_users_2 ON lk_users_2.member_id = members.id
			SET
				members.migration_done = 1;
		`,

		InsertToPivotQuery: `
			INSERT INTO mapping_lk_users_members (
				user_id,
				member_id
				)
			SELECT
				lk_users_2.user_id,
				lk_users_2.member_id
			FROM
				lk_users_2
			WHERE
				lk_users_2.is_migrated = 1;
		`,

		PivotTableMappingValidationQuery: `
			SELECT
				COUNT(*)
			FROM
				mapping_lk_users_members p
				JOIN lk_users_2 t ON p.user_id = t.user_id
				JOIN members s ON p.member_id = s.id;
		`,

		FieldLevelValidationQuery: `
			SELECT
				members.id,
				NOT(lk_users_2.user_domain <=> mapping_lk_domains_sites.domain_id) AS domain_mismatch,
				NOT(BINARY lk_users_2.user_fname <=> BINARY members.first_name) AS fname_mismatch,
				NOT(BINARY lk_users_2.user_status <=> BINARY members.status) AS status_mismatch,
				NOT(BINARY lk_users_2.user_lname <=> BINARY members.last_name) AS lname_mismatch,
				NOT(BINARY lk_users_2.user_email <=> BINARY members.email) AS email_mismatch,
				NOT(BINARY lk_users_2.user_phone_cell <=> BINARY members.mobile_phone) AS phone_mismatch,
				NOT(DATE_FORMAT(lk_users_2.user_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(members.user_joined_at, '%%Y-%%m-%%d %%H:%%i:%%s')) AS date_added_mismatch,
				NOT(DATE_FORMAT(lk_users_2.user_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(members.user_updated_at, '%%Y-%%m-%%d %%H:%%i:%%s')) AS date_updated_mismatch
			FROM
				members
				JOIN lk_users_2 ON lk_users_2.member_id = members.id
				JOIN mapping_lk_domains_sites ON members.site_id = mapping_lk_domains_sites.site_id
			WHERE
				members.migration_done = 1
				AND lk_users_2.is_migrated = 1
				AND(NOT(lk_users_2.user_domain <=> mapping_lk_domains_sites.domain_id)
					OR NOT(BINARY lk_users_2.user_fname <=> BINARY members.first_name)
			 		OR NOT(BINARY lk_users_2.user_status <=> BINARY members.status) -- enum
					OR NOT(BINARY lk_users_2.user_lname <=> BINARY members.last_name)
					OR NOT(BINARY lk_users_2.user_email <=> BINARY members.email)
					OR NOT(BINARY lk_users_2.user_phone_cell <=> BINARY members.mobile_phone)
					OR NOT(DATE_FORMAT(lk_users_2.user_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(members.user_joined_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
					OR NOT(DATE_FORMAT(lk_users_2.user_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(members.user_updated_at, '%%Y-%%m-%%d %%H:%%i:%%s')));
		`,

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM lk_users_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "lk_users_2",
			},
			{
				Query:       "UPDATE members SET migration_done = 0 WHERE migration_done = 1",
				Description: "♻️  Reset migration_done = 0 in",
				Table:       "members",
			},
			{
				Query:       "DELETE FROM mapping_lk_users_members",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_lk_users_members",
			},
		},
	}
}
