package yaml

import "github.com/lyraproj/issue/issue"

const (
	BadParameter         = `WF_BAD_PARAMETER`
	FieldTypeMismatch    = `WF_FIELD_TYPE_MISMATCH`
	InvalidTypeName      = `WF_INVALID_TYPE_NAME`
	MissingRequiredField = `WF_MISSING_REQUIRED_FIELD`
	NotStepDefinition    = `WF_NOT_STEP_DEFINITION`
	NotStep              = `WF_NOT_STEP`
)

func init() {
	issue.Hard2(BadParameter, `%{step}: element %{name} is not a valid %{parameterType} parameter`, issue.HF{`step`: issue.Label})
	issue.Hard2(FieldTypeMismatch, `%{step}: expected %{field} to be a %{expected}, got %{actual}`, issue.HF{`step`: issue.Label})
	issue.Hard(InvalidTypeName, `invalid type name '%{name}'. A type name must consist of one to many capitalized segments separated with '::'`)
	issue.Hard(MissingRequiredField, `missing required field '%{field}'`)
	issue.Hard(NotStepDefinition, `a step definition must be a hash`)
	issue.Hard(NotStep, `step hash must contain workflow "steps" or a resource "state"`)
}
