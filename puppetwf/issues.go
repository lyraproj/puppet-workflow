package puppetwf

import "github.com/lyraproj/issue/issue"

const (
	FieldTypeMismatch        = `WF_FIELD_TYPE_MISMATCH`
	ElementNotParameter      = `WF_ELEMENT_NOT_PARAMETER`
	NoDefinition             = `WF_NO_DEFINITION`
	NoServerBuilderInContext = `WF_NO_SERVER_BUILDER_IN_CONTEXT`
	NotActivity              = `WF_NOT_ACTIVITY`
	InvalidFunction          = `WF_INVALID_FUNCTION`
	InvalidTypeName          = `WF_INVALID_TYPE_NAME`
	MissingRequiredFunction  = `WF_MISSING_REQUIRED_FUNCTION`
)

func init() {
	issue.Hard(FieldTypeMismatch, `expected activity %{field} to be a %{expected}, got %{actual}`)
	issue.Hard(ElementNotParameter, `expected activity %{field} element to be a Parameter, got %{type}`)
	issue.Hard(NoDefinition, `expected activity to contain a definition block`)
	issue.Hard(NoServerBuilderInContext, `no ServerBuilder has been registered with the evaluation context`)
	issue.Hard(InvalidFunction, `invalid function '%{function}'. Expected one of 'create', 'read', 'update', or 'delete'`)
	issue.Hard(InvalidTypeName, `invalid type name '%{name}'. A type name must consist of one to many capitalized segments separated with '::'`)
	issue.Hard(MissingRequiredFunction, `missing required '%{function}'`)
	issue.Hard2(NotActivity, `block may only contain workflow activities. %{actual} is not supported here`,
		issue.HF{`actual`: issue.UcAnOrA})
}
