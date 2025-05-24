package migrate

import (
	"database/sql"
	"db-migration/sets"
	"fmt"
	"strings"
)

type RollbackStep struct {
	Query       string
	Description string
	Table       string
}

// Helper to check if a column exists in a table
func columnExists(db *sql.DB, tableName, columnName string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?
	`, tableName, columnName).Scan(&count)
	return count > 0, err
}

func CreatePivotTable(db *sql.DB, tableName string, columns map[string]string) error {
	fmt.Printf("⏳ Creating pivot table %s...\n", tableName)

	columnDefinition := ""
	columnNames := []string{}
	checkConstraints := []string{}

	for col, typ := range columns {
		columnDefinition += fmt.Sprintf("%s %s,", col, typ)
		columnNames = append(columnNames, col)
		checkConstraints = append(checkConstraints, fmt.Sprintf("CHECK (%s > 0)", col))
	}

	uniqueKeyName := "unique_"
	for i, col := range columnNames {
		if i > 0 {
			uniqueKeyName += "_"
		}
		uniqueKeyName += col
	}

	uniqueKey := fmt.Sprintf("UNIQUE KEY %s (%s)", uniqueKeyName, strings.Join(columnNames, ", "))
	checks := strings.Join(checkConstraints, ",\n")

	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		%s
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		%s,
		%s
	);`, tableName, columnDefinition, uniqueKey, checks)

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("create pivot table failed: %w", err)
	}

	fmt.Printf("🛡️ ✨ UNIQUE constraint on (%s), key name: %s\n", strings.Join(columnNames, ", "), uniqueKeyName)
	fmt.Printf("✅ Pivot table **%s** created\n\n", tableName)
	return nil
}

func AlterTable(db *sql.DB, table string, addCols, updateCols map[string]string) error {
	fmt.Printf("⚙️  Altering table: **%s**:\n", table)

	var alterClauses []string

	// Check & prepare ADD COLUMN clauses
	if len(addCols) > 0 {
		fmt.Printf("⏳ Adding new columns to table **%s**:\n", table)
		for col, typ := range addCols {
			fmt.Printf("   ➕ %s %s\n", col, typ)

			exists, err := columnExists(db, table, col)
			if err != nil {
				return fmt.Errorf("checking column %s existence failed: %w", col, err)
			}
			if !exists {
				alterClauses = append(alterClauses, fmt.Sprintf("ADD COLUMN %s %s", col, typ))
			} else {
				fmt.Printf("	✅ column %s already exists\n", col)
			}
		}
	}

	// Prepare MODIFY COLUMN clauses
	if len(updateCols) > 0 {
		fmt.Printf("⏳ Modifying existing columns in table **%s**:\n", table)
		for col, typ := range updateCols {
			fmt.Printf("   ✏️  %s => %s\n", col, typ)
			alterClauses = append(alterClauses, fmt.Sprintf("MODIFY COLUMN %s %s", col, typ))
		}
	}

	// Execute combined ALTER TABLE statement if needed
	if len(alterClauses) > 0 {
		query := fmt.Sprintf("ALTER TABLE %s %s", table, strings.Join(alterClauses, ", "))
		fmt.Println(query)
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("executing ALTER TABLE failed: %w", err)
		}
		fmt.Printf("✅ Successfully altered table **%s**.\n\n", table)
	} else {
		fmt.Printf("ℹ️  No changes detected for table **%s**.\n\n", table)
	}

	return nil
}

func MigrateData(tx *sql.Tx, insertQuery, updateSourceQuery, insertPivotQuery string) error {
	fmt.Println("📝 Inserting into target table...")
	if _, err := tx.Exec(insertQuery); err != nil {
		return fmt.Errorf("failed to insert into target table: %w", err)
	}
	fmt.Printf("✅ Target table insert done\n\n")

	fmt.Println("✏️  Updating source table...")
	if _, err := tx.Exec(updateSourceQuery); err != nil {
		return fmt.Errorf("failed to update source table: %w", err)
	}
	fmt.Printf("✅ Source table update done\n\n")

	fmt.Println("🔗 Linking data into pivot table...")
	if _, err := tx.Exec(insertPivotQuery); err != nil {
		return fmt.Errorf("failed to insert into pivot table: %w", err)
	}
	fmt.Printf("✅ Pivot table insert done\n\n")

	return nil
}

func RollbackMigration(db *sql.DB, steps []sets.SingleRollbackStep) error {
	for _, step := range steps {
		if _, err := db.Exec(step.Query); err != nil {
			fmt.Printf("❌ Rollback step failed: %s %s\n", step.Description, step.Table)
			fmt.Println("🔎 Error:", err)
			fmt.Println("\n🚨 ⚠️  ❌ Rollback was not fully completed.")
			fmt.Println("📝 Please manually execute the following queries to finish rollback:")

			for _, s := range steps {
				fmt.Println()
				fmt.Printf("---- %s %s ----\n", s.Description, s.Table)
				fmt.Println(s.Query)
				fmt.Println()
			}

			return fmt.Errorf("rollback failed at step [%s %s]: %w", step.Description, step.Table, err)
		}
		fmt.Printf("%s %s\n", step.Description, step.Table)
	}

	return nil
}
