package sets

type SingleRollbackStep struct {
	Query       string
	Description string
	Table       string
}

type MigrationSet struct {
	SourceTableName                  string
	TargetTableName                  string
	PivotTableName                   string
	PivotTableColumns                map[string]string
	NewColumnsForTargetTable         map[string]string
	UpdateColumnsForTargetTable      map[string]string
	NewColumnsForSourceTable         map[string]string
	InsertToTargetQuery              string
	UpdateSourceQuery                string
	InsertToPivotQuery               string
	PivotTableMappingValidationQuery string
	FieldLevelValidationQuery        string
	RollbackSteps                    []SingleRollbackStep
}
