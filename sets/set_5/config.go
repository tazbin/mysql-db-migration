package set5

import (
	"db-migration/sets"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		TargetTableName: "lk_user_groups_users_2",
		SourceTableName: "group_member",
		PivotTableName:  "mapping_lk_user_groups_users_group_member",

		PivotTableColumns: map[string]string{
			"user_groups_users_id": "BIGINT UNSIGNED NOT NULL",
			"group_member_id":      "BIGINT UNSIGNED NOT NULL",
		},

		/* target table modification starts */
		NewColumnsForTargetTable: map[string]string{
			"group_member_id":     "BIGINT UNSIGNED",
			"ugu_date_added_ts":   "TIMESTAMP",
			"ugu_date_updated_ts": "TIMESTAMP",
			"is_migrated":         "TINYINT(1) DEFAULT 0",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"ugu_domain_id": "BIGINT UNSIGNED",
			"ugu_user_id":   "BIGINT UNSIGNED",
			"ugu_ug_id":     "BIGINT UNSIGNED",
			"ugu_leader":    "BIGINT UNSIGNED",
		},
		/* target table modification ends */

		/* source table modification starts */
		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: `
			INSERT INTO lk_user_groups_users_2 (
				group_member_id,
				ugu_domain_id,
				ugu_user_id,
				ugu_ug_id,
				ugu_leader,
				ugu_date_added,
				ugu_date_updated,
				is_migrated
				)
			SELECT
				group_member.id,
				mapping_lk_domains_sites.domain_id,
				mapping_lk_users_members.user_id,
				mapping_lk_user_groups_groups.user_group_id,
				group_member.created_by_member_id,
				group_member.created_at,
				group_member.updated_at,
				1
			FROM
				group_member
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = group_member.site_id
				JOIN mapping_lk_users_members ON mapping_lk_users_members.member_id = group_member.member_id
				JOIN mapping_lk_user_groups_groups ON mapping_lk_user_groups_groups.group_id = group_member.group_id;
		`,

		UpdateSourceQuery: `
			UPDATE
				group_member
				JOIN lk_user_groups_users_2 ON lk_user_groups_users_2.group_member_id = group_member.id
			SET
				group_member.migration_done = 1;
		`,

		InsertToPivotQuery: `
			INSERT INTO mapping_lk_user_groups_users_group_member (
				user_groups_users_id, 
				group_member_id
				)
			SELECT
				lk_user_groups_users_2.ugu_id,
				lk_user_groups_users_2.group_member_id
			FROM
				lk_user_groups_users_2
			WHERE
				lk_user_groups_users_2.is_migrated = 1;
		`,

		PivotTableMappingValidationQuery: `
			SELECT
				COUNT(*)
			FROM
				mapping_lk_user_groups_users_group_member p
				JOIN lk_user_groups_users_2 t ON p.user_groups_users_id = t.ugu_id
				JOIN group_member s ON p.group_member_id = s.id;
		`,

		FieldLevelValidationQuery: `
			SELECT
				group_member.id
			FROM
				group_member
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = group_member.site_id
				JOIN mapping_lk_users_members ON mapping_lk_users_members.member_id = group_member.member_id
				JOIN mapping_lk_user_groups_groups ON mapping_lk_user_groups_groups.group_id = group_member.group_id
				JOIN lk_user_groups_users_2 ON lk_user_groups_users_2.group_member_id = group_member.id
			WHERE
				group_member.migration_done = 1
				AND lk_user_groups_users_2.is_migrated = 1
				AND(NOT(lk_user_groups_users_2.ugu_domain_id <=> mapping_lk_domains_sites.domain_id)
					OR NOT(lk_user_groups_users_2.ugu_user_id <=> mapping_lk_users_members.user_id)
					OR NOT(lk_user_groups_users_2.ugu_ug_id <=> mapping_lk_user_groups_groups.user_group_id)
					OR NOT(lk_user_groups_users_2.ugu_leader <=> group_member.created_by_member_id)
					OR NOT(DATE_FORMAT(lk_user_groups_users_2.ugu_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(group_member.created_at, '%%Y-%%m-%%d %%H:%%i:%%s'))
					OR NOT(DATE_FORMAT(lk_user_groups_users_2.ugu_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(group_member.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s')))
			LIMIT 3;
		`,

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM lk_user_groups_users_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "lk_user_groups_users_2",
			},
			{
				Query:       "UPDATE group_member SET migration_done = 0 WHERE migration_done = 1",
				Description: "🗑️  Deleted migration_done column from",
				Table:       "group_member",
			},
			{
				Query:       "DELETE FROM mapping_lk_user_groups_users_group_member",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_lk_user_groups_users_group_member",
			},
		},
	}
}
