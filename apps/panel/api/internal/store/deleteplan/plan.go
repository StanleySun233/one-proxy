package deleteplan

const OperationDelete = "delete"

type DeletePlan struct {
	ResourceType string
	ResourceID   string
	Summary      []DeleteImpactItem
	Steps        []DeletePlanStep
}

type DeleteImpactItem struct {
	ResourceType string
	ResourceID   string
	DisplayName  string
	Count        int64
}

type DeletePlanStep struct {
	Name           string
	Table          string
	Operation      string
	WhereSQL       string
	Args           []any
	ExpectedImpact []DeleteImpactItem
}

type DeleteExecutionResult struct {
	PlanResourceType string
	PlanResourceID   string
	Steps            []DeleteStepResult
}

type DeleteStepResult struct {
	Name         string
	Table        string
	RowsAffected int64
}
