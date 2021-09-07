package iampolicy

import (
	"github.com/storeros/ipos/pkg/bucket/policy/condition"
)

type AdminAction string

const (
	HealAdminAction = "admin:Heal"

	StorageInfoAdminAction         = "admin:StorageInfo"
	AccountingUsageInfoAdminAction = "admin:AccountingUsageInfo"
	DataUsageInfoAdminAction       = "admin:DataUsageInfo"
	TopLocksAdminAction            = "admin:TopLocksInfo"
	ProfilingAdminAction           = "admin:Profiling"
	TraceAdminAction               = "admin:ServerTrace"
	ConsoleLogAdminAction          = "admin:ConsoleLog"
	KMSKeyStatusAdminAction        = "admin:KMSKeyStatus"
	ServerInfoAdminAction          = "admin:ServerInfo"
	OBDInfoAdminAction             = "admin:OBDInfo"

	ServerUpdateAdminAction = "admin:ServerUpdate"

	ConfigUpdateAdminAction = "admin:ConfigUpdate"

	CreateUserAdminAction = "admin:CreateUser"

	DeleteUserAdminAction  = "admin:DeleteUser"
	ListUsersAdminAction   = "admin:ListUsers"
	EnableUserAdminAction  = "admin:EnableUser"
	DisableUserAdminAction = "admin:DisableUser"
	GetUserAdminAction     = "admin:GetUser"

	AddUserToGroupAdminAction      = "admin:AddUserToGroup"
	RemoveUserFromGroupAdminAction = "admin:RemoveUserFromGroup"
	GetGroupAdminAction            = "admin:GetGroup"
	ListGroupsAdminAction          = "admin:ListGroups"
	EnableGroupAdminAction         = "admin:EnableGroup"
	DisableGroupAdminAction        = "admin:DisableGroup"

	CreatePolicyAdminAction     = "admin:CreatePolicy"
	DeletePolicyAdminAction     = "admin:DeletePolicy"
	GetPolicyAdminAction        = "admin:GetPolicy"
	AttachPolicyAdminAction     = "admin:AttachUserOrGroupPolicy"
	ListUserPoliciesAdminAction = "admin:ListUserPolicies"
	AllAdminActions             = "admin:*"
)

var supportedAdminActions = map[AdminAction]struct{}{
	AllAdminActions:                {},
	HealAdminAction:                {},
	ServerInfoAdminAction:          {},
	StorageInfoAdminAction:         {},
	DataUsageInfoAdminAction:       {},
	TopLocksAdminAction:            {},
	ProfilingAdminAction:           {},
	TraceAdminAction:               {},
	OBDInfoAdminAction:             {},
	ConsoleLogAdminAction:          {},
	KMSKeyStatusAdminAction:        {},
	ServerUpdateAdminAction:        {},
	ConfigUpdateAdminAction:        {},
	CreateUserAdminAction:          {},
	DeleteUserAdminAction:          {},
	ListUsersAdminAction:           {},
	EnableUserAdminAction:          {},
	DisableUserAdminAction:         {},
	GetUserAdminAction:             {},
	AddUserToGroupAdminAction:      {},
	RemoveUserFromGroupAdminAction: {},
	ListGroupsAdminAction:          {},
	EnableGroupAdminAction:         {},
	DisableGroupAdminAction:        {},
	CreatePolicyAdminAction:        {},
	DeletePolicyAdminAction:        {},
	GetPolicyAdminAction:           {},
	AttachPolicyAdminAction:        {},
	ListUserPoliciesAdminAction:    {},
}

func parseAdminAction(s string) (AdminAction, error) {
	action := AdminAction(s)
	if action.IsValid() {
		return action, nil
	}

	return action, Errorf("unsupported action '%v'", s)
}

func (action AdminAction) IsValid() bool {
	_, ok := supportedAdminActions[action]
	return ok
}

var adminActionConditionKeyMap = map[Action]condition.KeySet{
	AllAdminActions:                condition.NewKeySet(condition.AllSupportedAdminKeys...),
	HealAdminAction:                condition.NewKeySet(condition.AllSupportedAdminKeys...),
	StorageInfoAdminAction:         condition.NewKeySet(condition.AllSupportedAdminKeys...),
	ServerInfoAdminAction:          condition.NewKeySet(condition.AllSupportedAdminKeys...),
	DataUsageInfoAdminAction:       condition.NewKeySet(condition.AllSupportedAdminKeys...),
	OBDInfoAdminAction:             condition.NewKeySet(condition.AllSupportedAdminKeys...),
	TopLocksAdminAction:            condition.NewKeySet(condition.AllSupportedAdminKeys...),
	ProfilingAdminAction:           condition.NewKeySet(condition.AllSupportedAdminKeys...),
	TraceAdminAction:               condition.NewKeySet(condition.AllSupportedAdminKeys...),
	ConsoleLogAdminAction:          condition.NewKeySet(condition.AllSupportedAdminKeys...),
	KMSKeyStatusAdminAction:        condition.NewKeySet(condition.AllSupportedAdminKeys...),
	ServerUpdateAdminAction:        condition.NewKeySet(condition.AllSupportedAdminKeys...),
	ConfigUpdateAdminAction:        condition.NewKeySet(condition.AllSupportedAdminKeys...),
	CreateUserAdminAction:          condition.NewKeySet(condition.AllSupportedAdminKeys...),
	DeleteUserAdminAction:          condition.NewKeySet(condition.AllSupportedAdminKeys...),
	ListUsersAdminAction:           condition.NewKeySet(condition.AllSupportedAdminKeys...),
	EnableUserAdminAction:          condition.NewKeySet(condition.AllSupportedAdminKeys...),
	DisableUserAdminAction:         condition.NewKeySet(condition.AllSupportedAdminKeys...),
	GetUserAdminAction:             condition.NewKeySet(condition.AllSupportedAdminKeys...),
	AddUserToGroupAdminAction:      condition.NewKeySet(condition.AllSupportedAdminKeys...),
	RemoveUserFromGroupAdminAction: condition.NewKeySet(condition.AllSupportedAdminKeys...),
	ListGroupsAdminAction:          condition.NewKeySet(condition.AllSupportedAdminKeys...),
	EnableGroupAdminAction:         condition.NewKeySet(condition.AllSupportedAdminKeys...),
	DisableGroupAdminAction:        condition.NewKeySet(condition.AllSupportedAdminKeys...),
	CreatePolicyAdminAction:        condition.NewKeySet(condition.AllSupportedAdminKeys...),
	DeletePolicyAdminAction:        condition.NewKeySet(condition.AllSupportedAdminKeys...),
	GetPolicyAdminAction:           condition.NewKeySet(condition.AllSupportedAdminKeys...),
	AttachPolicyAdminAction:        condition.NewKeySet(condition.AllSupportedAdminKeys...),
	ListUserPoliciesAdminAction:    condition.NewKeySet(condition.AllSupportedAdminKeys...),
}
