## ✅ Migration Steps

1. **Adjust Target Table Column Sizes**  
   Modify the column lengths or data types in the target table to accommodate incoming data from the source.  
   _Example: `VARCHAR(100)` → `VARCHAR(255)`_

2. **Add Source Identifier Column**  
   Add a new column in the target table (e.g., `source_table_id`) to store the corresponding source record's primary key, enabling clear row mapping.

3. **Introduce `is_migrated` Flag**  
   Add a boolean column `is_migrated` (default `FALSE`) to the target table to track which records have been migrated. During data insertion, set this flag to `TRUE`.

4. **Migrate Data**  
   Insert all relevant rows from the source table into the target table, ensuring accurate column mapping and proper data transformation where necessary.



## 🛠️ Migration Approaches

We have two ways to do the migration:

### 1. Automated Script
- A script written in Go can handle the migration automatically.
- We just provide a config with table and column names.
- It's automated process.

### 2. Manual SQL Queries
- We can write and run SQL queries by hand for each table.
- Good for small or one-time migrations where automation isn’t needed.


## ⚠️ Key Considerations

- **Mapping Strategies:**  
  Choose one of the following ways to map records between the source and target tables:
  - Use an **intermediary mapping table** (e.g., `migration_map`) to store source and target record IDs.
  - Add the **target table's ID** as a foreign key in the source table.

- **Schema Modifications:**  
  Since the target database schema (column size/type) is being altered, it is recommended to perform these changes through **application-level migrations** (e.g., with a migration tool).  
  This ensures consistency across:
  - Local development environments
  - CI/CD pipelines
  - All production and staging environments

## 🧪 Testing & Rollbacks

- Use the `is_migrated` column to count and verify migrated rows.
- Perform row count validations before and after migration.
- In case of migration failure, use mapping data to safely rollback or retry.


## 💡 Best Practices

- Never modify already applied migration scripts.
- Running the script twice will result in double insertion.
- Maintain strict version control of migration scripts.
- Ensure all scripts are idempotent where possible.
- Run tests on staging before applying to production.
- Properly check migrated data if correctly inserted
