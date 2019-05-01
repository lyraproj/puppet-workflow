package puppetwf

import "github.com/lyraproj/issue/issue"

const (
	FieldTypeMismatch        = `WF_FIELD_TYPE_MISMATCH`
	ElementNotParameter      = `WF_ELEMENT_NOT_PARAMETER`
	NoDefinition             = `WF_NO_DEFINITION`
	NoServerBuilderInContext = `WF_NO_SERVER_BUILDER_IN_CONTEXT`
	NotStep                  = `WF_NOT_STEP`
	InvalidFunction          = `WF_INVALID_FUNCTION`
	InvalidTypeName          = `WF_INVALID_TYPE_NAME`
	MissingRequiredField     = `WF_MISSING_REQUIRED_FIELD`
	MissingRequiredFunction  = `WF_MISSING_REQUIRED_FUNCTION`
)

func init() {
	issue.Hard(FieldTypeMismatch, `expected field %{field} to be a %{expected}, got %{actual}`)
	issue.Hard(ElementNotParameter, `expected field %{field} element to be a Parameter, got %{type}`)
	issue.Hard(NoDefinition, `expected step to contain a definition block`)
	issue.Hard(NoServerBuilderInContext, `no ServerBuilder has been registered with the evaluation context`)
	issue.Hard(InvalidFunction, `invalid function '%{function}'. Expected one of 'create', 'read', 'update', or 'delete'`)
	issue.Hard(InvalidTypeName, `invalid type name '%{name}'. A type name must consist of one to many capitalized segments separated with '::'`)
	issue.Hard(MissingRequiredField, `missing required field '%{field}'`)
	issue.Hard(MissingRequiredFunction, `missing required '%{function}'`)
	issue.Hard2(NotStep, `block may only contain workflow steps. %{actual} is not supported here`,
		issue.HF{`actual`: issue.UcAnOrA})
}
