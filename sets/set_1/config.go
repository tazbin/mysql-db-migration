package set1

import (
	"db-migration/sets"
	"fmt"
)

func GetMigrationSet() sets.MigrationSet {
	return sets.MigrationSet{
		SourceTableName: "sites",
		TargetTableName: "lk_domains_2",
		PivotTableName:  "mapping_lk_domains_sites",

		PivotTableColumns: map[string]string{
			"domain_id": "INT UNSIGNED NOT NULL",
			"site_id":   "BIGINT UNSIGNED NOT NULL",
		},

		NewColumnsForTargetTable: map[string]string{
			"is_migrated":            "TINYINT(1) DEFAULT 0",
			"sites_id":               "BIGINT UNSIGNED",
			"domain_date_added_ts":   "TIMESTAMP",
			"domain_date_updated_ts": "TIMESTAMP",
		},

		UpdateColumnsForTargetTable: map[string]string{
			"domain_postal": "VARCHAR(255)",
		},

		NewColumnsForSourceTable: map[string]string{
			"migration_done": "TINYINT(1) DEFAULT 0",
		},

		InsertToTargetQuery: fmt.Sprintf(`
			INSERT INTO %s (
				sites_id,
				domain_status,
				domain_name,
				domain_cname,
				domain_alias,
				domain_sitename,
				domain_date_added_ts,
				domain_date_updated_ts,
				domain_billing_type,
				domain_live,
				domain_postal,
				lat,
				lng,
				is_migrated
			)
			SELECT
				id,
				status,
				domain,
				domain,
				domain,
				name,
				created_at,
				updated_at,
				internal,
				live,
				postal_code,
				lat,
				lng,
				1
			FROM %s
			WHERE migration_done = 0;
		`, "lk_domains_2", "sites"),

		UpdateSourceQuery: fmt.Sprintf(`
			UPDATE %s s
			JOIN %s t ON s.id = t.sites_id
			SET s.migration_done = 1;
		`, "sites", "lk_domains_2"),

		InsertToPivotQuery: fmt.Sprintf(`
			INSERT INTO %s (domain_id, site_id)
			SELECT t.domain_id, t.sites_id
			FROM %s t
			WHERE t.is_migrated = 1;
		`, "mapping_lk_domains_sites", "lk_domains_2"),

		FieldLevelValidationQuery: fmt.Sprintf(`
			SELECT s.id
			FROM %s s
			JOIN %s t ON s.id = t.sites_id
			WHERE s.migration_done = 1 AND t.is_migrated = 1 AND (
				NOT (t.domain_name <=> s.domain) OR
				NOT (t.domain_cname <=> s.domain) OR
				NOT (t.domain_alias <=> s.domain) OR
				NOT (t.domain_sitename <=> s.name) OR
				NOT (DATE_FORMAT(t.domain_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(s.created_at, '%%Y-%%m-%%d %%H:%%i:%%s')) OR
				NOT (DATE_FORMAT(t.domain_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(s.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s')) OR
				NOT (t.domain_live <=> s.live) OR
				NOT (t.domain_postal <=> s.postal_code) OR
				NOT (ROUND(t.lat, 5) <=> ROUND(s.lat, 5)) OR
				NOT (ROUND(t.lng, 5) <=> ROUND(s.lng, 5))
			)
			LIMIT 3;
		`, "sites", "lk_domains_2"),

		RollbackSteps: []sets.SingleRollbackStep{
			{
				Query:       "DELETE FROM lk_domains_2 WHERE is_migrated = 1",
				Description: "🗑️  Deleted migrated rows from",
				Table:       "lk_domains_2",
			},
			{
				Query:       "UPDATE sites SET migration_done = 0 WHERE migration_done = 1",
				Description: "♻️  Reset migration_done = 0 in",
				Table:       "sites",
			},
			{
				Query:       "DELETE FROM mapping_lk_domains_sites",
				Description: "🧹 Deleted rows from",
				Table:       "mapping_lk_domains_sites",
			},
		},
	}
}
