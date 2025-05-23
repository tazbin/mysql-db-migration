package migrate

import (
	"database/sql"
	"fmt"
)

func ValidateMigratedData(tx *sql.Tx, sourceTable, targetTable, pivotTable string) error {
	err := validateMigrationRowCount(tx, sourceTable, targetTable, pivotTable)
	if err != nil {
		return err
	}

	err = checkFieldLevelEquality(tx, sourceTable, targetTable)
	if err != nil {
		return err
	}

	return nil
}

func validateMigrationRowCount(tx *sql.Tx, sourceTable, targetTable, pivotTable string) error {
	var sourceCount, targetCount, pivotCount, validReferenceCount int

	err := tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE migration_done = 1", sourceTable)).Scan(&sourceCount)
	if err != nil {
		return fmt.Errorf("failed to count migrated rows in source: %w", err)
	}

	err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE is_migrated = 1", targetTable)).Scan(&targetCount)
	if err != nil {
		return fmt.Errorf("failed to count migrated rows in target: %w", err)
	}

	if sourceCount != targetCount {
		return fmt.Errorf("mismatch in migrated rows: source has %d, target has %d", sourceCount, targetCount)
	}

	fmt.Printf("✅ Migration validated: %d rows migrated successfully\n", sourceCount)

	err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", pivotTable)).Scan(&pivotCount)
	if err != nil {
		return fmt.Errorf("failed to count rows in pivot table: %w", err)
	}

	if pivotCount != targetCount {
		return fmt.Errorf("pivot table validation failed: expected %d rows, got %d", targetCount, pivotCount)
	}

	fmt.Printf("✅ Pivot table validated: %d mappings exist\n", pivotCount)

	query := fmt.Sprintf(`
				SELECT COUNT(*) 
				FROM %s p 
				JOIN %s t ON p.domain_id = t.domain_id 
				JOIN %s s ON p.site_id = s.id;
			`, pivotTable, targetTable, sourceTable)

	err = tx.QueryRow(query).Scan(&validReferenceCount)
	if err != nil {
		return fmt.Errorf("failed to count valid foreign key references in pivot table: %w", err)
	}

	if validReferenceCount != pivotCount {
		return fmt.Errorf("referential integrity check failed: expected %d valid mappings, but got %d", pivotCount, validReferenceCount)
	}

	fmt.Printf("✅ Referential integrity validated: %d valid foreign key mappings found in pivot table\n", validReferenceCount)

	return nil
}

func checkFieldLevelEquality(tx *sql.Tx, sourceTable, targetTable string) error {
	query := fmt.Sprintf(`
		SELECT s.id
		FROM %s s
		JOIN %s t ON s.id = t.sites_id
		WHERE s.migration_done = 1 AND t.is_migrated = 1 AND (
			-- NOT (s.status <=> t.domain_status) OR -- enum
			NOT (s.domain <=> t.domain_name) OR
			NOT (s.domain <=> t.domain_cname) OR
			NOT (s.domain <=> t.domain_alias) OR
			NOT (s.name <=> t.domain_sitename) OR
			NOT (DATE_FORMAT(s.created_at, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(t.domain_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s')) OR
			NOT (DATE_FORMAT(s.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(t.domain_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s')) OR
			-- NOT (s.internal <=> t.domain_billing_type) OR -- enum
			NOT (s.live <=> t.domain_live) OR
			NOT (s.postal_code <=> t.domain_postal) OR
			NOT (ROUND(s.lat, 5) <=> ROUND(t.lat, 5)) OR
			NOT (ROUND(s.lng, 5) <=> ROUND(t.lng, 5))
		)
		LIMIT 3;
	`, sourceTable, targetTable)

	rows, err := tx.Query(query)
	if err != nil {
		return fmt.Errorf("field-level validation failed: %w", err)
	}
	defer rows.Close()

	var mismatchedIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan mismatch row: %w", err)
		}
		mismatchedIDs = append(mismatchedIDs, id)
	}

	if len(mismatchedIDs) > 0 {
		fmt.Printf("❌ Field mismatch in %d rows. Example mismatched IDs: %v\n", len(mismatchedIDs), mismatchedIDs)
		return fmt.Errorf("field-level mismatch detected")
	}

	fmt.Println("✅ Field-level validation passed")
	return nil
}
