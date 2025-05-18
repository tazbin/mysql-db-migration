package migrate

import (
	"database/sql"
	"fmt"
	"strings"
)

// Helper to check if a column exists in a table
func columnExists(tx *sql.Tx, tableName, columnName string) (bool, error) {
	var count int
	err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?
	`, tableName, columnName).Scan(&count)
	return count > 0, err
}

func CreatePivotTable(tx *sql.Tx, pivot map[string]interface{}) error {
	tableName := pivot["table_name"].(string)
	columns := pivot["column_and_types"].(map[string]string)

	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP`, tableName)

	for col, typ := range columns {
		query += fmt.Sprintf(", %s %s", col, typ)
	}

	query += ");"

	_, err := tx.Exec(query)
	if err != nil {
		return fmt.Errorf("create pivot table failed: %w", err)
	}

	fmt.Println("✅ pivot table created")
	return nil
}

func AlterTargetTable(tx *sql.Tx, table string, addCols, updateCols map[string]string) error {
	for col, typ := range addCols {
		exists, err := columnExists(tx, table, col)
		if err != nil {
			return fmt.Errorf("checking column %s existence failed: %w", col, err)
		}
		if !exists {
			_, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, typ))
			if err != nil {
				return fmt.Errorf("add column %s failed: %w", col, err)
			}
		}
	}

	for col, typ := range updateCols {
		_, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s", table, col, typ))
		if err != nil {
			return fmt.Errorf("modify column %s failed: %w", col, err)
		}
	}

	fmt.Println("✅ target table columns altered.")
	return nil
}

func AddMigrationDoneColumnToTargetTable(tx *sql.Tx, table string) error {
	exists, err := columnExists(tx, table, "migration_done")
	if err != nil {
		return fmt.Errorf("checking 'migration_done' existence failed: %w", err)
	}
	if !exists {
		_, err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN migration_done TINYINT DEFAULT 0`, table))
		if err != nil {
			return fmt.Errorf("add column 'migration_done' failed: %w", err)
		}
	}

	fmt.Println("✅ 'migration_done' added to target table.")
	return nil
}

func MigrateData(tx *sql.Tx, sourceTable, targetTable, pivotTable string, columnMapping map[string]string) error {
	// 1. Insert into target table
	var sourceCols, targetCols []string
	for targetCol, sourceCol := range columnMapping {
		targetCols = append(targetCols, targetCol)
		sourceCols = append(sourceCols, sourceCol)
	}
	targetCols = append(targetCols, "sites_id", "is_migrated")
	sourceCols = append(sourceCols, "id", "1") // id -> sites_id, 1 -> is_migrated

	insertQuery := fmt.Sprintf(`
		INSERT INTO %s (%s)
		SELECT %s FROM %s
		WHERE migration_done = 0
	`, targetTable, strings.Join(targetCols, ", "), strings.Join(sourceCols, ", "), sourceTable)

	if _, err := tx.Exec(insertQuery); err != nil {
		return fmt.Errorf("failed to insert into target table: %w", err)
	}

	// 2. Update source table: set migration_done = 1
	updateSourceQuery := fmt.Sprintf(`
		UPDATE %s s
		JOIN %s t ON s.id = t.sites_id
		SET s.migration_done = 1
		WHERE s.migration_done = 0
	`, sourceTable, targetTable)

	if _, err := tx.Exec(updateSourceQuery); err != nil {
		return fmt.Errorf("failed to update source table: %w", err)
	}

	// 3. Insert into pivot table
	insertPivotQuery := fmt.Sprintf(`
		INSERT INTO %s (domain_id, site_id)
		SELECT t.id, t.sites_id
		FROM %s t
		WHERE t.is_migrated = 1
	`, pivotTable, targetTable)

	if _, err := tx.Exec(insertPivotQuery); err != nil {
		return fmt.Errorf("failed to insert into pivot table: %w", err)
	}

	return nil
}
