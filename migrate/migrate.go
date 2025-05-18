package migrate

import (
	"database/sql"
	"fmt"
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
	fmt.Print("⏳ Creating pivot table... ")

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

func AlterTable(tx *sql.Tx, table string, addCols, updateCols map[string]string) error {
	if len(addCols) > 0 {
		fmt.Printf("⏳ Adding columns to %s table...\n", table)
	}
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
	if len(addCols) > 0 {
		fmt.Printf("✅  Added columns to %s table...\n", table)
	}

	if len(updateCols) > 0 {
		fmt.Printf("⏳ Updating columns to %s table...\n", table)
	}
	for col, typ := range updateCols {
		_, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s", table, col, typ))
		if err != nil {
			return fmt.Errorf("modify column %s failed: %w", col, err)
		}
	}

	if len(updateCols) > 0 {
		fmt.Printf("✅  Updated columns to %s table...\n", table)
	}
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

func MigrateData(tx *sql.Tx, insertQuery, updateSourceQuery, insertPivotQuery string) error {
	if _, err := tx.Exec(insertQuery); err != nil {
		return fmt.Errorf("failed to insert into target table: %w", err)
	}

	if _, err := tx.Exec(updateSourceQuery); err != nil {
		return fmt.Errorf("failed to update source table: %w", err)
	}

	if _, err := tx.Exec(insertPivotQuery); err != nil {
		return fmt.Errorf("failed to insert into pivot table: %w", err)
	}

	return nil
}
