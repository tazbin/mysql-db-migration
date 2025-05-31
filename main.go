package main

import (
	"db-migration/db"
	"db-migration/logger"
	"db-migration/migrate"
	"db-migration/sets"
	set1 "db-migration/sets/set_1"
	set10 "db-migration/sets/set_10"
	set2 "db-migration/sets/set_2"
	set3 "db-migration/sets/set_3"
	set4 "db-migration/sets/set_4"
	set5 "db-migration/sets/set_5"
	set6 "db-migration/sets/set_6"
	set7 "db-migration/sets/set_7"
	set8 "db-migration/sets/set_8"
	set9 "db-migration/sets/set_9"
	"fmt"
	"log"
	"os"
)

func main() {

	logFile := "migration.log"

	err := logger.Init(logFile)
	if err != nil {
		log.Fatalf("❌ Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	log.SetOutput(logger.Logger.Writer())

	if len(os.Args) < 3 {
		fmt.Println("❗ Please provide a command and migration set name: \n   → do-migrate set1\n   → undo-migrate set1")
		return
	}

	command := os.Args[1]
	setName := os.Args[2]

	// Load migration set based on setName
	var migrationSet sets.MigrationSet

	switch setName {
	case "set_1":
		migrationSet = set1.GetMigrationSet()
	case "set_2":
		migrationSet = set2.GetMigrationSet()
	case "set_3":
		migrationSet = set3.GetMigrationSet()
	case "set_4":
		migrationSet = set4.GetMigrationSet()
	case "set_5":
		migrationSet = set5.GetMigrationSet()
	case "set_6":
		migrationSet = set6.GetMigrationSet()
	case "set_7":
		migrationSet = set7.GetMigrationSet()
	case "set_8":
		migrationSet = set8.GetMigrationSet()
	case "set_9":
		migrationSet = set9.GetMigrationSet()
	case "set_10":
		migrationSet = set10.GetMigrationSet()
	default:
		fmt.Printf("❗ Unknown migration set: %s\n", setName)
		return
	}

	dbName := "galaxy_g2"
	if setName == "set_10" {
		dbName = "galaxy_email"
	}

	cfg := db.Config{
		SSHUser:    "ec2-user",
		SSHHost:    "bastion.dev.galaxydigital.com",
		SSHPort:    22,
		SSHKeyPath: "sabbir-ssh-key",
		DBUser:     "galaxy_g2",
		DBPassword: "yourpassword",
		DBHost:     "mysql.dev.galaxydigital.com",
		DBPort:     3306,
		DBName:     dbName,
	}

	db.Connect(cfg)

	switch command {
	case "do-migrate":
		fmt.Printf("⚠️  You are about to migrate data:\n")
		fmt.Printf("   → FROM: %s\n", migrationSet.SourceTableName)
		fmt.Printf("   → TO:   %s\n", migrationSet.TargetTableName)
		if migrationSet.PivotTableName != "" {
			fmt.Printf("   → VIA:  %s\n", migrationSet.PivotTableName)
		}

		fmt.Print("Proceed with migration? (y/N): ")
		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("❌ Migration cancelled.")
			return
		}

		fmt.Println()
		log.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		log.Printf("           ⏳ Starting migration for %s...     \n", setName)
		log.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")

		err := migrate.CreatePivotTable(db.DB, migrationSet.PivotTableName, migrationSet.PivotTableColumns)
		if err != nil {
			log.Fatalf("❌ Pivot table creation failed: %v", err)
		}

		err = migrate.AlterTable(db.DB, migrationSet.TargetTableName, migrationSet.NewColumnsForTargetTable, migrationSet.UpdateColumnsForTargetTable)
		if err != nil {
			log.Fatalf("❌ Alter target table failed: %v", err)
		}

		err = migrate.AlterTable(db.DB, migrationSet.SourceTableName, migrationSet.NewColumnsForSourceTable, map[string]string{})
		if err != nil {
			log.Fatalf("❌ Alter source table failed: %v", err)
		}

		tx, err := db.DB.Begin()
		if err != nil {
			log.Fatalf("❌ Failed to start transaction: %v", err)
		}

		err = migrate.MigrateData(tx, migrationSet.InsertToTargetQuery, migrationSet.UpdateSourceQuery, migrationSet.InsertToPivotQuery)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Migration failed: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			log.Fatalf("❌ Failed to commit transaction: %v", err)
		}

		err = migrate.ValidateMigratedData(db.DB, migrationSet.SourceTableName, migrationSet.TargetTableName, migrationSet.PivotTableName, migrationSet.PivotTableMappingValidationQuery, migrationSet.FieldLevelValidationQuery)
		if err != nil {
			tx.Rollback()
			log.Fatalf("❌ Migration validation failed: %v", err)
		}

		log.Println("✅ Migration successful!")

	case "undo-migrate":
		fmt.Printf("⚠️  You are about to undo the migration involving these queries:\n\n")

		for _, step := range migrationSet.RollbackSteps {
			fmt.Printf("→ %s %s:\n", step.Description, step.Table)
			fmt.Printf("     %s\n", step.Query)
			countQuery := migrate.GenerateCountQuery(step.Query)

			var count int
			if err := db.DB.QueryRow(countQuery).Scan(&count); err != nil {
				fmt.Printf("⚠️  Could not count affected rows for: %s %s\n", step.Description, step.Table)
				fmt.Println("🔎 Error:", err)
				fmt.Printf("\n")
			} else {
				action := migrate.GetActionFromQuery(step.Query)
				fmt.Printf("  🧾 %d rows will be %s from %s\n\n", count, action, step.Table)
			}
		}

		fmt.Print("Proceed with undo migration? (y/N): ")
		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("❌ Undo migration cancelled.")
			return
		}

		fmt.Println()
		log.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		log.Printf("           ⚠️  Starting rollback for %s...        \n", setName)
		log.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")

		err := migrate.RollbackMigration(db.DB, migrationSet.RollbackSteps)
		if err != nil {
			fmt.Println("⚠️  Rollback encountered an issue. See above for details.")
		} else {
			log.Println("\n✅ Undo migration completed successfully!")
		}

	default:
		fmt.Printf("❗ Unknown command: %s\n", command)
		fmt.Println("Available commands: do-migrate, undo-migrate")
	}

}
