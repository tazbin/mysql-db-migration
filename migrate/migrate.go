package migrate

import (
	"database/sql"
	"fmt"
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

func CreatePivotTable(db *sql.DB, pivot map[string]interface{}) error {
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

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("create pivot table failed: %w", err)
	}

	fmt.Println("✅ pivot table created")
	return nil
}

func AlterTable(db *sql.DB, table string, addCols, updateCols map[string]string) error {
	if len(addCols) > 0 {
		fmt.Printf("⏳ Adding new columns to table **%s**:\n", table)
		for col, typ := range addCols {
			fmt.Printf("   ➕ %s %s\n", col, typ)
		}
	}
	for col, typ := range addCols {
		exists, err := columnExists(db, table, col)
		if err != nil {
			return fmt.Errorf("checking column %s existence failed: %w", col, err)
		}
		if !exists {
			_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, typ))
			if err != nil {
				return fmt.Errorf("add column %s failed: %w", col, err)
			}
		}
	}
	if len(addCols) > 0 {
		fmt.Printf("✅ Successfully added new columns to table **%s**.\n\n", table)
	}

	if len(updateCols) > 0 {
		fmt.Printf("⏳ Modifying existing columns in table **%s**:\n", table)
		for col, typ := range updateCols {
			fmt.Printf("   ✏️  %s => %s\n", col, typ)
		}
	}
	for col, typ := range updateCols {
		_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s", table, col, typ))
		if err != nil {
			return fmt.Errorf("modify column %s failed: %w", col, err)
		}
	}
	if len(updateCols) > 0 {
		fmt.Printf("✅ Successfully updated columns in table **%s**.\n\n", table)
	}

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

func RollbackMigration(db *sql.DB, steps []RollbackStep) error {
	for _, step := range steps {
		if _, err := db.Exec(step.Query); err != nil {
			fmt.Printf("❌ Rollback step failed: %s %s\n", step.Description, step.Table)
			fmt.Println("🔎 Error:", err)
			fmt.Println("\n💥 Rollback was not fully completed.")
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
