package yaml

import "github.com/lyraproj/issue/issue"

const (
	BadParameter      = `WF_BAD_PARAMETER`
	FieldTypeMismatch = `WF_FIELD_TYPE_MISMATCH`
	NotOneDefinition  = `WF_NOT_ONE_DEFINITION`
	NotActivity       = `WF_NOT_ACTIVITY`
)

func init() {
	issue.Hard2(FieldTypeMismatch, `%{activity}: expected %{field} to be a %{expected}, got %{actual}`, issue.HF{`activity`: issue.Label})
	issue.Hard2(BadParameter, `%{activity}: element %{name} is not a valid %{parameterType} parameter`, issue.HF{`activity`: issue.Label})
	issue.Hard(NotOneDefinition, `expected exactly one top level key (the activity name)`)
	issue.Hard(NotActivity, `activity hash must contain workflow "activities" or a resource "state"`)
}
