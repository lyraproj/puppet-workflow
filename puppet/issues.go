package puppet

import "github.com/lyraproj/issue/issue"

const (
	WF_FIELD_TYPE_MISMATCH       = `WF_FIELD_TYPE_MISMATCH`
	WF_ELEMENT_NOT_PARAMETER     = `WF_ELEMENT_NOT_PARAMETER`
	WF_NO_DEFINITION             = `WF_NO_DEFINITION`
	WF_NOT_ACTIVITY              = `WF_NOT_ACTIVITY`
	WF_INVALID_FUNCTION          = `WF_INVALID_FUNCTION`
	WF_MISSING_REQUIRED_FUNCTION = `WF_MISSING_REQUIRED_FUNCTION`
)

func init() {
	issue.Hard(WF_FIELD_TYPE_MISMATCH, `expected activity %{field} to be a %{expected}, got %{actual}`)
	issue.Hard(WF_ELEMENT_NOT_PARAMETER, `expected activity %{field} element to be a Parameter, got %{type}`)
	issue.Hard(WF_NO_DEFINITION, `expected activity to contain a definition block`)
	issue.Hard(WF_INVALID_FUNCTION, `invalid function '%{function}'. Expected one of 'create', 'read', 'update', or 'delete'`)
	issue.Hard(WF_MISSING_REQUIRED_FUNCTION, `missing required '%{function}'`)
	issue.Hard2(WF_NOT_ACTIVITY, `block may only contain workflow activities. %{actual} is not supported here`,
		issue.HF{`actual`: issue.A_anUc})
}
