package set2

import (
	"db-migration/sets"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		TargetTableName: "lk_domains_settings_2",
		SourceTableName: "site_settings",
		PivotTableName:  "mapping_lk_domains_settings_sites_settings",

		PivotTableColumns: map[string]string{
			"domain_settings_id": "BIGINT UNSIGNED NOT NULL",
			"site_settings_id":   "BIGINT UNSIGNED NOT NULL",
		},

		/* target table modification starts */
		NewColumnsForTargetTable: map[string]string{
			"site_settings_id": "BIGINT UNSIGNED",
			"is_migrated":      "TINYINT(1) DEFAULT 0",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"domain_id": "BIGINT UNSIGNED",
			"k":         "VARCHAR(255)",
		},
		/* target table modification ends */

		/* source table modification starts */
		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: `
			INSERT INTO lk_domains_settings_2 (
				site_settings_id,
				domain_id,
				k,
				v,
				is_migrated
			)
			SELECT
				site_settings.id,
				mapping_lk_domains_sites.domain_id,
				site_settings.key,
				site_settings.value,
				1
			FROM
				site_settings
				JOIN mapping_lk_domains_sites ON mapping_lk_domains_sites.site_id = site_settings.site_id;
		`,

		UpdateSourceQuery: `
			UPDATE
				site_settings
				JOIN lk_domains_settings_2 ON site_settings.id = lk_domains_settings_2.site_settings_id
			SET
				site_settings.migration_done = 1;
		`,

		InsertToPivotQuery: `
			INSERT INTO mapping_lk_domains_settings_sites_settings (
				domain_settings_id, 
				site_settings_id
				)
			SELECT
				lk_domains_settings_2.id,
				lk_domains_settings_2.site_settings_id
			FROM
				lk_domains_settings_2
			WHERE
				lk_domains_settings_2.is_migrated = 1;
		`,

		PivotTableMappingValidationQuery: `
			SELECT
				COUNT(*)
			FROM
				mapping_lk_domains_settings_sites_settings p
				JOIN lk_domains_settings_2 t ON p.domain_settings_id = t.id
				JOIN site_settings s ON p.site_settings_id = s.id;
		`,

		FieldLevelValidationQuery: `
			SELECT
				site_settings.id
			FROM
				site_settings
				JOIN lk_domains_settings_2 ON site_settings.id = lk_domains_settings_2.site_settings_id
				JOIN mapping_lk_domains_sites ON site_settings.site_id = mapping_lk_domains_sites.site_id
			WHERE
				site_settings.migration_done = 1
				AND lk_domains_settings_2.is_migrated = 1
				AND(
					NOT(lk_domains_settings_2.domain_id <=> mapping_lk_domains_sites.domain_id)
					OR NOT(BINARY lk_domains_settings_2.k <=> BINARY site_settings.key)
					OR NOT(BINARY lk_domains_settings_2.v <=> BINARY site_settings.value)
					)
			LIMIT 3;
		`,

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM lk_domains_settings_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "lk_domains_settings_2",
			},
			{
				Query:       "UPDATE site_settings SET migration_done = 0 WHERE migration_done = 1",
				Description: "♻️  Reset migration_done = 0 in",
				Table:       "site_settings",
			},
			{
				Query:       "DELETE FROM mapping_lk_domains_settings_sites_settings",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_lk_domains_settings_sites_settings",
			},
		},
	}
}
