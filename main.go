package main

import (
	"db-migration/db"
	"db-migration/logger"
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

	sourceTableName := "sites"
	targetTableName := "lk_domains_2"
	pivotTableName := "mapping_lk_domains_sites"

	pivotTableColumns := map[string]string{
		// id
		"domain_id": "INT UNSIGNED NOT NULL",    // target table id
		"site_id":   "BIGINT UNSIGNED NOT NULL", // source table id
		// created_at
		// updated_at
	}

	newColumnsForTargetTable := map[string]string{
		"is_migrated":            "TINYINT(1) DEFAULT 0",
		"sites_id":               "BIGINT UNSIGNED",
		"domain_date_added_ts":   "TIMESTAMP",
		"domain_date_updated_ts": "TIMESTAMP",
	}

	updateColumnsForTargetTable := map[string]string{
		"domain_postal": "VARCHAR(255)",
	}

	newColumnsForSourceTable := map[string]string{
		"migration_done": "TINYINT(1) DEFAULT 0",
	}

	insertToTargetQuery := fmt.Sprintf(`
								INSERT INTO %s (
									sites_id,
									domain_status,
									domain_name,
									domain_cname,
									domain_alias,
									domain_sitename,
									domain_date_added_ts,
									domain_date_updated_ts,
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
								`, targetTableName, sourceTableName)

	updateSourceQuery := fmt.Sprintf(`
								UPDATE %s s
								JOIN %s t ON s.id = t.sites_id
								SET s.migration_done = 1;
								`, sourceTableName, targetTableName)

	insertToPivotQuery := fmt.Sprintf(`
								INSERT INTO %s (domain_id, site_id)
								SELECT t.domain_id, t.sites_id
								FROM %s t
								WHERE t.is_migrated = 1;
								`, pivotTableName, targetTableName)

	fieldLevelValidationQuery := fmt.Sprintf(`
		SELECT s.id
		FROM %s s
		JOIN %s t ON s.id = t.sites_id
		WHERE s.migration_done = 1 AND t.is_migrated = 1 AND (
			-- NOT (s.status <=> t.domain_status) OR -- enum
			NOT (s.domain <=> t.domain_name) OR
			NOT (s.domain <=> t.domain_cname) OR
			NOT (s.domain <=> t.domain_alias) OR
			NOT (s.name <=> t.domain_sitename) OR
			NOT (DATE_FORMAT(s.created_at, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(t.domain_date_added_ts, '%%Y-%%m-%%d %%H:%%i:%%s')) OR
			NOT (DATE_FORMAT(s.updated_at, '%%Y-%%m-%%d %%H:%%i:%%s') <=> DATE_FORMAT(t.domain_date_updated_ts, '%%Y-%%m-%%d %%H:%%i:%%s')) OR
			-- NOT (s.internal <=> t.domain_billing_type) OR -- enum
			NOT (s.live <=> t.domain_live) OR
			NOT (s.postal_code <=> t.domain_postal) OR
			NOT (ROUND(s.lat, 5) <=> ROUND(t.lat, 5)) OR
			NOT (ROUND(s.lng, 5) <=> ROUND(t.lng, 5))
		)
		LIMIT 3;
	`, sourceTableName, targetTableName)

	rollbackSteps := []migrate.RollbackStep{
		{
			Query:       fmt.Sprintf(`DELETE FROM %s WHERE is_migrated = 1`, targetTableName),
			Description: "🗑️  Deleted migrated rows from",
			Table:       targetTableName,
		},
		{
			Query:       fmt.Sprintf(`UPDATE %s SET migration_done = 0 WHERE migration_done = 1`, sourceTableName),
			Description: "♻️  Reset migration_done = 0 in",
			Table:       sourceTableName,
		},
		{
			Query:       fmt.Sprintf(`DELETE FROM %s`, pivotTableName),
			Description: "🧹 Deleted rows from",
			Table:       pivotTableName,
		},
	}

	// Setup logging
	logFile := "migration.log"

	err := logger.Init(logFile)
	if err != nil {
		log.Fatalf("❌ Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	// Replace default log output to use our logger
	log.SetOutput(logger.Logger.Writer())

	switch command {
	case "do-migrate":
		fmt.Printf("⚠️  You are about to migrate data:\n")
		fmt.Printf("   → FROM: %s\n", sourceTableName)
		fmt.Printf("   → TO:   %s\n", targetTableName)
		if pivotTableName != "" {
			fmt.Printf("   → VIA:  %s\n", pivotTableName)
		}

		fmt.Print("Proceed with migration? (y/N): ")
		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("❌ Migration cancelled.")
			return
		}

		fmt.Println()
		log.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		log.Println("           ⏳ Starting migration...          ")
		log.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")

		err := migrate.CreatePivotTable(db.DB, pivotTableName, pivotTableColumns)
		if err != nil {
			log.Fatalf("❌ Pivot table creation failed: %v", err)
		}

		err = migrate.AlterTable(db.DB, targetTableName, newColumnsForTargetTable, updateColumnsForTargetTable)
		if err != nil {
			log.Fatalf("❌ Alter target table failed: %v", err)
		}

		err = migrate.AlterTable(db.DB, sourceTableName, newColumnsForSourceTable, map[string]string{})
		if err != nil {
			log.Fatalf("❌ Alter source table failed: %v", err)
		}

		tx, err := db.DB.Begin()
		if err != nil {
			log.Fatalf("❌ Failed to start transaction: %v", err)
		}

		err = migrate.MigrateData(tx, insertToTargetQuery, updateSourceQuery, insertToPivotQuery)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Migration failed: %v", err)
		}

		err = migrate.ValidateMigratedData(tx, sourceTableName, targetTableName, pivotTableName, fieldLevelValidationQuery)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Migration validation failed: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			log.Fatalf("❌ Failed to commit transaction: %v", err)
		}

		log.Println("✅ Migration successful!")

	case "undo-migrate":
		fmt.Printf("⚠️  You are about to undo the migration involving these queries:\n\n")

		for _, step := range rollbackSteps {
			fmt.Printf("→ %s %s:\n", step.Description, step.Table)
			fmt.Printf("   %s\n\n", step.Query)
		}

		fmt.Print("Proceed with undo migration? (y/N): ")
		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("❌ Undo migration cancelled.")
			return
		}

		fmt.Println()
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		fmt.Println("           ⚠️  Starting rollback...          ")
		fmt.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")

		err := migrate.RollbackMigration(db.DB, rollbackSteps)
		if err != nil {
			fmt.Println("⚠️  Rollback encountered an issue. See above for details.")
		}

		fmt.Println("\n✅ Undo migration completed successfully!")

	default:
		fmt.Printf("❗ Unknown command: %s\n", command)
		fmt.Println("Available commands: do-migrate, undo-migrate")
	}

}
