package migrate

import (
	"database/sql"
	"fmt"
	"strings"
)

// AddColumnsIfNotExist checks for and adds missing columns to the table
func AddColumnsIfNotExist(tx *sql.Tx, tableName string, columns map[string]string) error {
	for colName, colType := range columns {
		// Check if the column exists
		var count int
		query := `
			SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
			AND COLUMN_NAME = ?
		`
		err := tx.QueryRow(query, tableName, colName).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check column %s: %v", colName, err)
		}

		if count == 0 {
			// Column does not exist, add it
			alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, colName, colType)
			fmt.Printf("➕ Adding column: %s %s\n", colName, colType)
			_, err := tx.Exec(alterSQL)
			if err != nil {
				return fmt.Errorf("failed to add column %s: %v", colName, err)
			}
			fmt.Printf("✅ Column %s added successfully.\n", colName)
		} else {
			fmt.Printf("✅ Column %s already exists.\n", colName)
		}
	}
	return nil
}

// Updates column type
func UpdateColumnTypes(tx *sql.Tx, tableName string, mapping map[string]string) error {
	for col, newType := range mapping {
		// Get current type
		var columnType string
		query := `
			SELECT COLUMN_TYPE
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
			AND COLUMN_NAME = ?
		`

		err := tx.QueryRow(query, tableName, col).Scan(&columnType)
		if err != nil {
			return fmt.Errorf("failed to get current type for column %s: %v", col, err)
		}

		fmt.Printf("🔍 Column: %s | Current Type: %s | New Type: %s\n", col, columnType, newType)

		// Alter table
		alterSQL := fmt.Sprintf("ALTER TABLE %s MODIFY %s %s", tableName, col, newType)
		fmt.Println("Executing:", alterSQL)
		_, err = tx.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to alter column %s: %v", col, err)
		}

		fmt.Printf("✅ Column %s updated to %s successfully.\n", col, newType)
	}
	return nil
}

// MigrateData moves data with column mapping and adds `is_migrated = TRUE`.
func MigrateData(tx *sql.Tx, mapping map[string]interface{}) error {
	sourceTable := mapping["source_table"].(string)
	targetTable := mapping["target_table"].(string)
	columns := mapping["columns"].(map[string]string)

	var sourceCols []string
	var targetCols []string
	for srcCol, tgtCol := range columns {
		sourceCols = append(sourceCols, srcCol)
		targetCols = append(targetCols, tgtCol)
	}

	// 1️⃣ Check if 'is_migrated' column exists
	checkSQL := `
        SELECT COUNT(*)
        FROM INFORMATION_SCHEMA.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = ?
        AND COLUMN_NAME = 'is_migrated'
    `
	var count int
	err := tx.QueryRow(checkSQL, targetTable).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check if 'is_migrated' column exists: %v", err)
	}

	if count == 0 {
		alterSQL := fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN is_migrated BOOLEAN NOT NULL DEFAULT FALSE",
			targetTable,
		)
		fmt.Println("Adding is_migrated column:", alterSQL)
		_, err = tx.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to add 'is_migrated' column: %v", err)
		}
		fmt.Println("✅ 'is_migrated' column added successfully.")
	} else {
		fmt.Println("✅ 'is_migrated' column already exists.")
	}

	// 2️⃣ Build insert query with is_migrated = TRUE
	selectCols := fmt.Sprintf("%s, TRUE", joinCols(sourceCols))
	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s, is_migrated) SELECT %s FROM %s",
		targetTable,
		joinCols(targetCols),
		selectCols,
		sourceTable,
	)

	fmt.Println("Executing:", insertSQL)
	res, err := tx.Exec(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to migrate data: %v", err)
	}

	// ✅ Log how many rows were inserted
	rows, _ := res.RowsAffected()
	fmt.Printf("✅ Data migration completed successfully. Migrated %d rows.\n", rows)

	return nil
}

// joinCols wraps each column in backticks and joins them with commas
func joinCols(cols []string) string {
	for i := range cols {
		cols[i] = "`" + cols[i] + "`"
	}
	return strings.Join(cols, ", ")
}

// UndoMigration removes rows with `is_migrated = TRUE` from the target table.
func UndoMigration(tx *sql.Tx, mapping map[string]interface{}) error {
	targetTable := mapping["target_table"].(string)

	// 1️⃣ Delete rows with is_migrated = TRUE
	deleteSQL := fmt.Sprintf(
		"DELETE FROM %s WHERE is_migrated = TRUE",
		targetTable,
	)

	fmt.Println("Executing:", deleteSQL)
	res, err := tx.Exec(deleteSQL)
	if err != nil {
		return fmt.Errorf("failed to delete migrated rows: %v", err)
	}

	// ✅ Log how many rows were deleted
	rows, _ := res.RowsAffected()
	fmt.Printf("✅ Undo migration completed. Deleted %d rows.\n", rows)

	return nil
}
