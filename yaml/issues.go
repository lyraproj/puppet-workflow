package yaml

import "github.com/lyraproj/issue/issue"

const (
	WF_BAD_PARAMETER       = `WF_BAD_PARAMETER`
	WF_FIELD_TYPE_MISMATCH = `WF_FIELD_TYPE_MISMATCH`
	WF_NOT_ONE_DEFINITION  = `WF_NOT_ONE_DEFINITION`
	WF_NOT_ACTIVITY        = `WF_NOT_ACTIVITY`
)

func init() {
	issue.Hard2(WF_FIELD_TYPE_MISMATCH, `%{activity}: expected %{field} to be a %{expected}, got %{actual}`, issue.HF{`activity`: issue.Label})
	issue.Hard2(WF_BAD_PARAMETER, `%{activity}: element %{name} is not a valid %{parameter_type} parameter`, issue.HF{`activity`: issue.Label})
	issue.Hard(WF_NOT_ONE_DEFINITION, `expected exactly one top level key (the activity name)`)
	issue.Hard(WF_NOT_ACTIVITY, `activity hash must contain workflow "activities" or a resource "state"`)
}
