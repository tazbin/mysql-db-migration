package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

func ValidateMigratedData(db *sql.DB, sourceTable, targetTable, pivotTable, PivotTableMappingValidationQuery, fieldLevelValidationQuery, setName string) error {
	err := validateMigrationRowCount(db, sourceTable, targetTable, pivotTable, PivotTableMappingValidationQuery)
	if err != nil {
		return err
	}

	err = checkFieldLevelEquality(db, fieldLevelValidationQuery, setName)
	if err != nil {
		return err
	}

	return nil
}

func validateMigrationRowCount(db *sql.DB, sourceTable, targetTable, pivotTable, PivotTableMappingValidationQuery string) error {
	var sourceCount, targetCount, pivotCount, validReferenceCount int

	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE migration_done = 1", sourceTable)).Scan(&sourceCount)
	if err != nil {
		return fmt.Errorf("failed to count migrated rows in source: %w", err)
	}

	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE is_migrated = 1", targetTable)).Scan(&targetCount)
	if err != nil {
		return fmt.Errorf("failed to count migrated rows in target: %w", err)
	}

	if sourceCount != targetCount {
		return fmt.Errorf("mismatch in migrated rows: source has %d, target has %d", sourceCount, targetCount)
	}

	fmt.Printf("✅ Migration validated: %d rows migrated successfully\n", sourceCount)

	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", pivotTable)).Scan(&pivotCount)
	if err != nil {
		return fmt.Errorf("failed to count rows in pivot table: %w", err)
	}

	if pivotCount != targetCount {
		return fmt.Errorf("pivot table validation failed: expected %d rows, got %d", targetCount, pivotCount)
	}

	fmt.Printf("✅ Pivot table validated: %d mappings exist\n", pivotCount)

	err = db.QueryRow(PivotTableMappingValidationQuery).Scan(&validReferenceCount)
	if err != nil {
		return fmt.Errorf("failed to count valid foreign key references in pivot table: %w", err)
	}

	if validReferenceCount != pivotCount {
		return fmt.Errorf("referential integrity check failed: expected %d valid mappings, but got %d", pivotCount, validReferenceCount)
	}

	fmt.Printf("✅ Referential integrity validated: %d valid foreign key mappings found in pivot table\n", validReferenceCount)

	return nil
}

func checkFieldLevelEquality(db *sql.DB, fieldLevelValidationQuery, setName string) error {
	rows, err := db.Query(fieldLevelValidationQuery)
	if err != nil {
		return fmt.Errorf("field-level validation failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get column names: %w", err)
	}

	var (
		file          *os.File
		mismatchCount int
		logFile       = fmt.Sprintf("mismatches/%s_mismatches.log", setName)
	)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Lazy create the file if mismatch found
		if mismatchCount == 0 {
			// Create log file only on first mismatch
			if err := os.MkdirAll("mismatches", os.ModePerm); err != nil {
				return fmt.Errorf("failed to create mismatches directory: %w", err)
			}

			file, err = os.Create(logFile)
			if err != nil {
				return fmt.Errorf("failed to create log file: %w", err)
			}

			// Write header
			header := strings.Join(columns, " | ")
			_, _ = file.WriteString(header + "\n")

			// Separator line
			var sepParts []string
			for _, col := range columns {
				sepParts = append(sepParts, strings.Repeat("-", len(col)))
			}
			_, _ = file.WriteString(strings.Join(sepParts, "-+-") + "\n")
		}

		var rowStrings []string
		for _, val := range values {
			switch v := val.(type) {
			case []byte:
				rowStrings = append(rowStrings, fmt.Sprintf("%-10s", string(v)))
			case nil:
				rowStrings = append(rowStrings, "NULL")
			default:
				rowStrings = append(rowStrings, fmt.Sprintf("%-10v", v))
			}
		}

		_, _ = file.WriteString(strings.Join(rowStrings, " | ") + "\n")
		mismatchCount++
	}

	if file != nil {
		defer file.Close()
	}

	if mismatchCount > 0 {
		fmt.Printf("❌ Field mismatch in %d rows. Details written to '%s'\n", mismatchCount, logFile)
		return fmt.Errorf("field-level mismatch detected")
	}

	fmt.Println("✅ Field-level validation passed")
	return nil
}
