package main

import (
	"db-migration/db"
	"db-migration/migrate"
	"fmt"
	"log"
)

func updateColumnTypes(tableName string, mapping map[string]string) error {
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

		err := db.DB.QueryRow(query, tableName, col).Scan(&columnType)
		if err != nil {
			return fmt.Errorf("failed to get current type for column %s: %v", col, err)
		}

		fmt.Printf("🔍 Column: %s | Current Type: %s | New Type: %s\n", col, columnType, newType)

		// Alter table
		alterSQL := fmt.Sprintf("ALTER TABLE %s MODIFY %s %s", tableName, col, newType)
		fmt.Println("Executing:", alterSQL)
		_, err = db.DB.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to alter column %s: %v", col, err)
		}

		fmt.Printf("✅ Column %s updated to %s successfully.\n", col, newType)
	}
	return nil
}

func main() {

	// Config setup
	cfg := db.Config{
		SSHUser:    "ec2-user",
		SSHHost:    "bastion.dev.galaxydigital.com",
		SSHPort:    22,
		SSHKeyPath: "sabbir-ssh-key",
		DBUser:     "galaxy_g2",
		DBPassword: "yourpassword",
		DBHost:     "mysql.dev.galaxydigital.com",
		DBPort:     3306,
		DBName:     "galaxy_g2",
	}

	// Connect to DB
	db.Connect(cfg)

	// Define table and column mappings
	tableName := "lk_module_uw_needs_2"
	columnMapping := map[string]string{
		"need_id":        "BIGINT(20) UNSIGNED",
		"need_domain_id": "BIGINT(20) UNSIGNED",
		"need_city":      "VARCHAR(255)",
		"need_state":     "VARCHAR(255)",
		"need_postal":    "VARCHAR(255)",
		"need_country":   "VARCHAR(255)",
	}

	err := updateColumnTypes(tableName, columnMapping)
	if err != nil {
		log.Fatal(err)
	}

	mapping := map[string]interface{}{
		"source_table": "events",
		"target_table": "lk_module_uw_needs_2",
		"primary_key":  "need_id",
		"columns": map[string]string{
			// "id":          "need_id",
			"site_id":     "need_domain_id",
			"address":     "need_address",
			"city":        "need_city",
			"state":       "need_state",
			"postal_code": "need_postal",
			"country":     "need_country",
			"name":        "need_title",
			"description": "need_body",
		},
	}
	// 		sourvce      target

	err = migrate.MigrateData(mapping)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration successful!")
}
