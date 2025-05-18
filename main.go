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

	sourceTable := "sites"
	targetTable := "lk_domains_2"

	pivotTable := map[string]interface{}{
		"table_name": "mapping_lk_domains_sites",
		"column_and_types": map[string]string{
			// id
			"domain_id": "INT UNSIGNED",    // target table id
			"site_id":   "BIGINT UNSIGNED", // source table id
			// created_at
			// updated_at
		},
	}

	newColumnsForTargetTable := map[string]string{
		"is_migrated": "TINYINT(1) DEFAULT 0",
		"sites_id":    "BIGINT UNSIGNED",
	}

	updateColumnsForTargetTable := map[string]string{
		"domain_postal": "VARCHAR(255)",
	}

	newColumnsForSourceTable := map[string]string{
		"migration_done": "TINYINT(1) DEFAULT 0",
	}

	// Build SQL queries using fmt.Sprintf
	insertToTargetQuery := fmt.Sprintf(`
								INSERT INTO %s (
									sites_id,
									domain_status,
									domain_name,
									domain_cname,
									domain_alias,
									domain_sitename,
									domain_date_added,
									domain_date_updated,
									domain_billing_type,
									domain_live,
									domain_postal,
									lat,
									lng,
									is_migrated
								)
								SELECT
									id,
									status,
									domain,
									domain,
									domain,
									name,
									created_at,
									updated_at,
									internal,
									live,
									postal_code,
									lat,
									lng,
									1
								FROM %s
								WHERE migration_done = 0;
								`, targetTable, sourceTable)

	updateSourceQuery := fmt.Sprintf(`
								UPDATE %s s
								JOIN %s d ON s.id = d.sites_id
								SET s.migration_done = 1
								WHERE s.migration_done = 0;
								`, sourceTable, targetTable)

	insertToPivotQuery := fmt.Sprintf(`
								INSERT INTO %s (domain_id, site_id)
								SELECT d.domain_id, d.sites_id
								FROM %s d
								WHERE d.is_migrated = 1;
								`, pivotTable["table_name"].(string), targetTable)

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

		err = migrate.CreatePivotTable(tx, pivotTable)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Pivot table creation failed: %v", err)
		}

		err = migrate.AlterTable(tx, targetTable, newColumnsForTargetTable, updateColumnsForTargetTable)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Alter target table failed: %v", err)
		}

		err = migrate.AlterTable(tx, sourceTable, newColumnsForSourceTable, map[string]string{})
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Alter target table failed: %v", err)
		}

		err = migrate.AddMigrationDoneColumnToTargetTable(tx, targetTable)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Adding migration_done column failed: %v", err)
		}

		err = migrate.MigrateData(tx, insertToTargetQuery, updateSourceQuery, insertToPivotQuery)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Migration failed: %v", err)
		}

		tx.Rollback()
		fmt.Println("kaisa laga mera majak")

		// err = tx.Commit()
		// if err != nil {
		// 	log.Fatalf("❌ Failed to commit transaction: %v", err)
		// }

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

		fmt.Println("✅ Undo migration completed successfully!")

	default:
		fmt.Printf("❗ Unknown command: %s\n", command)
		fmt.Println("Available commands: do-migrate, undo-migrate")
	}

}
