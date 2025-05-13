package migrate

import (
	"db-migration/db"
	"fmt"
	"strings"
)

// MigrateData moves data with column mapping and adds `is_migrated = TRUE`. Uses transaction.
func MigrateData(mapping map[string]interface{}) error {
	sourceTable := mapping["source_table"].(string)
	targetTable := mapping["target_table"].(string)
	columns := mapping["columns"].(map[string]string)

	var sourceCols []string
	var targetCols []string
	for srcCol, tgtCol := range columns {
		sourceCols = append(sourceCols, srcCol)
		targetCols = append(targetCols, tgtCol)
	}

	// Start a transaction
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// 1️⃣ Check if 'is_migrated' column exists
	checkSQL := `
        SELECT COUNT(*)
        FROM INFORMATION_SCHEMA.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = ?
        AND COLUMN_NAME = 'is_migrated'
    `
	var count int
	err = tx.QueryRow(checkSQL, targetTable).Scan(&count)
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
func UndoMigration(mapping map[string]interface{}) error {
	targetTable := mapping["target_table"].(string)

	// Start a transaction
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

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
