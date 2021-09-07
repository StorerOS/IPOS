package iampolicy

import (
	"github.com/storeros/ipos/pkg/bucket/policy"
)

const (
	PolicyName        = "policy"
	SessionPolicyName = "sessionPolicy"
)

var ReadWrite = Policy{
	Version: DefaultVersion,
	Statements: []Statement{
		{
			SID:       policy.ID(""),
			Effect:    policy.Allow,
			Actions:   NewActionSet(AllActions),
			Resources: NewResourceSet(NewResource("*", "")),
		},
	},
}

var ReadOnly = Policy{
	Version: DefaultVersion,
	Statements: []Statement{
		{
			SID:       policy.ID(""),
			Effect:    policy.Allow,
			Actions:   NewActionSet(GetBucketLocationAction, GetObjectAction),
			Resources: NewResourceSet(NewResource("*", "")),
		},
	},
}

var WriteOnly = Policy{
	Version: DefaultVersion,
	Statements: []Statement{
		{
			SID:       policy.ID(""),
			Effect:    policy.Allow,
			Actions:   NewActionSet(PutObjectAction),
			Resources: NewResourceSet(NewResource("*", "")),
		},
	},
}

var AdminDiagnostics = Policy{
	Version: DefaultVersion,
	Statements: []Statement{
		{
			SID:    policy.ID(""),
			Effect: policy.Allow,
			Actions: NewActionSet(ProfilingAdminAction,
				TraceAdminAction, ConsoleLogAdminAction,
				ServerInfoAdminAction, TopLocksAdminAction,
				OBDInfoAdminAction),
			Resources: NewResourceSet(NewResource("*", "")),
		},
	},
}
