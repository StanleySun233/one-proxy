package deleteplan

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type MySQLExecutor struct {
	db *sql.DB
}

func NewMySQLExecutor(db *sql.DB) *MySQLExecutor {
	return &MySQLExecutor{db: db}
}

func (e *MySQLExecutor) Execute(ctx context.Context, plan DeletePlan) (DeleteExecutionResult, error) {
	result := DeleteExecutionResult{
		PlanResourceType: plan.ResourceType,
		PlanResourceID:   plan.ResourceID,
		Steps:            make([]DeleteStepResult, 0, len(plan.Steps)),
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, step := range plan.Steps {
		query, table, err := deleteSQL(step)
		if err != nil {
			return result, err
		}

		execResult, err := tx.ExecContext(ctx, query, step.Args...)
		if err != nil {
			return result, fmt.Errorf("execute delete step %q: %w", step.Name, err)
		}

		rowsAffected, err := execResult.RowsAffected()
		if err != nil {
			return result, fmt.Errorf("read affected rows for delete step %q: %w", step.Name, err)
		}

		result.Steps = append(result.Steps, DeleteStepResult{
			Name:         step.Name,
			Table:        table,
			RowsAffected: rowsAffected,
		})
	}

	if err := tx.Commit(); err != nil {
		return result, err
	}
	committed = true

	return result, nil
}

func deleteSQL(step DeletePlanStep) (string, string, error) {
	table := strings.TrimSpace(step.Table)
	if !validTableName(table) {
		return "", "", fmt.Errorf("delete step %q has invalid table %q", step.Name, step.Table)
	}

	if !strings.EqualFold(strings.TrimSpace(step.Operation), OperationDelete) {
		return "", "", fmt.Errorf("delete step %q has unsupported operation %q", step.Name, step.Operation)
	}

	whereSQL := strings.TrimSpace(step.WhereSQL)
	if whereSQL == "" {
		return "", "", fmt.Errorf("delete step %q has empty where predicate", step.Name)
	}

	return fmt.Sprintf("DELETE FROM %s WHERE %s", table, whereSQL), table, nil
}

func validTableName(table string) bool {
	if table == "" {
		return false
	}
	for index := 0; index < len(table); index++ {
		char := table[index]
		if index == 0 {
			if !asciiLetter(char) && char != '_' {
				return false
			}
			continue
		}
		if !asciiLetter(char) && !asciiDigit(char) && char != '_' {
			return false
		}
	}
	return true
}

func asciiLetter(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}

func asciiDigit(char byte) bool {
	return char >= '0' && char <= '9'
}
