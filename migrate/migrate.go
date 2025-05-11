package migrate

import (
	"db-migration/db"
	"fmt"
)

// MigrateData moves data and adds an `is_migrated` flag.
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

	// 1️⃣ Check if 'is_migrated' column exists
	checkSQL := `
        SELECT COUNT(*)
        FROM INFORMATION_SCHEMA.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = ?
        AND COLUMN_NAME = 'is_migrated'
    `
	var count int
	err := db.DB.QueryRow(checkSQL, targetTable).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check if 'is_migrated' column exists: %v", err)
	}

	if count == 0 {
		alterSQL := fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN is_migrated BOOLEAN NOT NULL DEFAULT FALSE",
			targetTable,
		)
		fmt.Println("Adding is_migrated column:", alterSQL)
		_, err := db.DB.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to add 'is_migrated' column: %v", err)
		}
		fmt.Println("✅ 'is_migrated' column added successfully.")
	} else {
		fmt.Println("✅ 'is_migrated' column already exists.")
	}

	// 2️⃣ Handle primary key for auto-incrementing
	var selectCols string
	var targetInsertCols []string = append([]string{}, targetCols...) // copy

	primaryKey, hasPrimaryKey := mapping["primary_key"].(string)
	if hasPrimaryKey && primaryKey != "" {
		// 🔍 Get max current value of the primary key
		var maxID int64
		maxSQL := fmt.Sprintf("SELECT COALESCE(MAX(`%s`), 0) FROM %s", primaryKey, targetTable)
		err := db.DB.QueryRow(maxSQL).Scan(&maxID)
		if err != nil {
			return fmt.Errorf("failed to get max %s: %v", primaryKey, err)
		}
		fmt.Printf("✅ Current max %s: %d\n", primaryKey, maxID)

		// 🆙 Initialize @rownum
		setRownumSQL := fmt.Sprintf("SET @rownum = %d", maxID)
		fmt.Println("Initializing @rownum:", setRownumSQL)
		_, err = db.DB.Exec(setRownumSQL)
		if err != nil {
			return fmt.Errorf("failed to initialize @rownum: %v", err)
		}

		// 🛠️ Add primary key at the start of insert columns
		targetInsertCols = append([]string{primaryKey}, targetInsertCols...)
		selectCols = fmt.Sprintf("(@rownum := @rownum + 1), %s, TRUE", joinCols(sourceCols))
	} else {
		selectCols = fmt.Sprintf("%s, TRUE", joinCols(sourceCols))
	}

	// 3️⃣ Build and run the final insert
	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s, is_migrated) SELECT %s FROM %s",
		targetTable,
		joinCols(targetInsertCols),
		selectCols,
		sourceTable,
	)

	fmt.Println("Executing:", insertSQL)
	_, err = db.DB.Exec(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to migrate data: %v", err)
	}

	fmt.Println("✅ Data migration completed successfully.")
	return nil
}

func joinCols(cols []string) string {
	return "`" + join(cols, "`, `") + "`"
}

func join(a []string, sep string) string {
	out := ""
	for i, v := range a {
		if i > 0 {
			out += sep
		}
		out += v
	}
	return out
}
