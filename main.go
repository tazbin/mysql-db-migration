package main

import (
	"db-migration/db"
	"db-migration/migrate"
	"fmt"
	"log"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("❗ Please provide a command: \n   → do-migrate \n   → undo-migrate")
		return
	}

	command := os.Args[1]

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

	newColumns := map[string]string{
		"event_id":              "BIGINT(20)",
		"need_date_added_utc":   "DATETIME",
		"need_date_updated_utc": "DATETIME",
		"is_migrated":           "BOOLEAN DEFAULT FALSE",
	}

	// Define table and column mappings
	columnTypeMapping := map[string]string{
		"need_id":        "BIGINT(20) UNSIGNED",
		"need_domain_id": "BIGINT(20) UNSIGNED",
		"need_city":      "VARCHAR(255)",
		"need_state":     "VARCHAR(255)",
		"need_postal":    "VARCHAR(255)",
		"need_country":   "VARCHAR(255)",
		"need_latitude":  "DECIMAL(14,8)",
		"need_longitude": "DECIMAL(14,8)",
	}

	// Define migration column mapping
	migrationMapping := map[string]interface{}{
		"source_table": "events",
		"target_table": "lk_module_uw_needs_2",
		"columns": map[string]string{
			// sourvce      target
			"id":          "event_id",
			"site_id":     "need_domain_id",
			"address":     "need_address",
			"city":        "need_city",
			"state":       "need_state",
			"postal_code": "need_postal",
			"country":     "need_country",
			"name":        "need_title",
			"description": "need_body",
			"private":     "need_public",
			"lat":         "need_latitude",
			"lng":         "need_longitude",
			"created_at":  "need_date_added_utc",
			"updated_at":  "need_date_updated_utc",
			"status":      "need_status",
		},
	}

	sourceTable := migrationMapping["source_table"].(string)
	targetTable := migrationMapping["target_table"].(string)

	switch command {
	case "do-migrate":
		fmt.Printf("⚠️  You are about to migrate data:\n   → FROM: %s\n   → TO:   %s\n", sourceTable, targetTable)
		fmt.Print("Proceed with migration? (y/N): ")
		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("❌ Migration cancelled.")
			return
		}

		tx, err := db.DB.Begin()
		if err != nil {
			log.Fatalf("❌ Failed to start transaction: %v", err)
		}

		err = migrate.AddColumnsIfNotExist(tx, targetTable, newColumns)
		if err != nil {
			tx.Rollback()
			log.Fatalf("Failed to add required columns: %v", err)
		}

		err = migrate.UpdateColumnTypes(tx, targetTable, columnTypeMapping)
		if err != nil {
			tx.Rollback()
			log.Fatal(err)
		}

		err = migrate.MigrateData(tx, migrationMapping)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Migration failed: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			log.Fatalf("❌ Failed to commit transaction: %v", err)
		}

		fmt.Println("✅ Migration successful!")

	case "undo-migrate":
		fmt.Printf("⚠️  You are about to DELETE migrated data \n   → DELETE FROM: %s\n   → Condition: is_migrated = TRUE\n", targetTable)
		fmt.Print("Proceed with undo migration? (y/N): ")
		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("❌ Undo migration cancelled.")
			return
		}

		tx, err := db.DB.Begin()
		if err != nil {
			log.Fatalf("❌ Failed to start transaction: %v", err)
		}

		err = migrate.UndoMigration(tx, migrationMapping)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Undo migration failed: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			log.Fatalf("❌ Failed to commit transaction: %v", err)
		}

		fmt.Println("✅ Undo migration completed successfully!")

	default:
		fmt.Printf("❗ Unknown command: %s\n", command)
		fmt.Println("Available commands: do-migrate, undo-migrate")
	}

}
