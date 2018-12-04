package functions

import "github.com/puppetlabs/go-issues/issue"

const (
	WF_NO_SERVER_BUILDER_IN_CONTEXT = `WF_NO_SERVER_BUILDER_IN_CONTEXT`
)

func init() {
	issue.Hard(WF_NO_SERVER_BUILDER_IN_CONTEXT, `no ServerBuilder has been registered with the evaluation context`)
}
