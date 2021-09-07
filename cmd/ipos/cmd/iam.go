package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/auth"
	"github.com/storeros/ipos/pkg/bucket/policy"
	iampolicy "github.com/storeros/ipos/pkg/iam/policy"
	"github.com/storeros/ipos/pkg/madmin"
	"github.com/storeros/ipos/pkg/set"
)

type UsersSysType string

const (
	IPOSUsersSysType UsersSysType = "IPOSUsersSys"
)

const (
	iamConfigPrefix = iposConfigPrefix + "/iam"

	iamConfigUsersPrefix = iamConfigPrefix + "/users/"

	iamConfigServiceAccountsPrefix = iamConfigPrefix + "/service-accounts/"

	iamConfigGroupsPrefix = iamConfigPrefix + "/groups/"

	iamConfigPoliciesPrefix = iamConfigPrefix + "/policies/"

	iamConfigSTSPrefix = iamConfigPrefix + "/sts/"

	iamConfigPolicyDBPrefix                = iamConfigPrefix + "/policydb/"
	iamConfigPolicyDBUsersPrefix           = iamConfigPolicyDBPrefix + "users/"
	iamConfigPolicyDBSTSUsersPrefix        = iamConfigPolicyDBPrefix + "sts-users/"
	iamConfigPolicyDBServiceAccountsPrefix = iamConfigPolicyDBPrefix + "service-accounts/"
	iamConfigPolicyDBGroupsPrefix          = iamConfigPolicyDBPrefix + "groups/"

	iamIdentityFile = "identity.json"

	iamPolicyFile = "policy.json"

	iamGroupMembersFile = "members.json"

	iamFormatFile = "format.json"

	iamFormatVersion1 = 1
)

const (
	statusEnabled  = "enabled"
	statusDisabled = "disabled"
)

type iamFormat struct {
	Version int `json:"version"`
}

func newIAMFormatVersion1() iamFormat {
	return iamFormat{Version: iamFormatVersion1}
}

func getIAMFormatFilePath() string {
	return iamConfigPrefix + SlashSeparator + iamFormatFile
}

func getUserIdentityPath(user string, userType IAMUserType) string {
	var basePath string
	switch userType {
	case srvAccUser:
		basePath = iamConfigServiceAccountsPrefix
	case stsUser:
		basePath = iamConfigSTSPrefix
	default:
		basePath = iamConfigUsersPrefix
	}
	return pathJoin(basePath, user, iamIdentityFile)
}

func getGroupInfoPath(group string) string {
	return pathJoin(iamConfigGroupsPrefix, group, iamGroupMembersFile)
}

func getPolicyDocPath(name string) string {
	return pathJoin(iamConfigPoliciesPrefix, name, iamPolicyFile)
}

func getMappedPolicyPath(name string, userType IAMUserType, isGroup bool) string {
	if isGroup {
		return pathJoin(iamConfigPolicyDBGroupsPrefix, name+".json")
	}
	switch userType {
	case srvAccUser:
		return pathJoin(iamConfigPolicyDBServiceAccountsPrefix, name+".json")
	case stsUser:
		return pathJoin(iamConfigPolicyDBSTSUsersPrefix, name+".json")
	default:
		return pathJoin(iamConfigPolicyDBUsersPrefix, name+".json")
	}
}

type UserIdentity struct {
	Version     int              `json:"version"`
	Credentials auth.Credentials `json:"credentials"`
}

func newUserIdentity(creds auth.Credentials) UserIdentity {
	return UserIdentity{Version: 1, Credentials: creds}
}

type GroupInfo struct {
	Version int      `json:"version"`
	Status  string   `json:"status"`
	Members []string `json:"members"`
}

func newGroupInfo(members []string) GroupInfo {
	return GroupInfo{Version: 1, Status: statusEnabled, Members: members}
}

type MappedPolicy struct {
	Version int    `json:"version"`
	Policy  string `json:"policy"`
}

func newMappedPolicy(policy string) MappedPolicy {
	return MappedPolicy{Version: 1, Policy: policy}
}

type IAMSys struct {
	usersSysType UsersSysType

	iamPolicyDocsMap        map[string]iampolicy.Policy
	iamUsersMap             map[string]auth.Credentials
	iamGroupsMap            map[string]GroupInfo
	iamUserGroupMemberships map[string]set.StringSet
	iamUserPolicyMap        map[string]MappedPolicy
	iamGroupPolicyMap       map[string]MappedPolicy

	store IAMStorageAPI
}

type IAMUserType int

const (
	regularUser IAMUserType = iota
	stsUser
	srvAccUser
)

type IAMStorageAPI interface {
	lock()
	unlock()

	rlock()
	runlock()

	migrateBackendFormat(context.Context) error

	loadPolicyDoc(policy string, m map[string]iampolicy.Policy) error
	loadPolicyDocs(ctx context.Context, m map[string]iampolicy.Policy) error

	loadUser(user string, userType IAMUserType, m map[string]auth.Credentials) error
	loadUsers(ctx context.Context, userType IAMUserType, m map[string]auth.Credentials) error

	loadGroup(group string, m map[string]GroupInfo) error
	loadGroups(ctx context.Context, m map[string]GroupInfo) error

	loadMappedPolicy(name string, userType IAMUserType, isGroup bool, m map[string]MappedPolicy) error
	loadMappedPolicies(ctx context.Context, userType IAMUserType, isGroup bool, m map[string]MappedPolicy) error

	loadAll(context.Context, *IAMSys) error

	saveIAMConfig(item interface{}, path string) error
	loadIAMConfig(item interface{}, path string) error
	deleteIAMConfig(path string) error

	savePolicyDoc(policyName string, p iampolicy.Policy) error
	saveMappedPolicy(name string, userType IAMUserType, isGroup bool, mp MappedPolicy) error
	saveUserIdentity(name string, userType IAMUserType, u UserIdentity) error
	saveGroupInfo(group string, gi GroupInfo) error

	deletePolicyDoc(policyName string) error
	deleteMappedPolicy(name string, userType IAMUserType, isGroup bool) error
	deleteUserIdentity(name string, userType IAMUserType) error
	deleteGroupInfo(name string) error

	watch(context.Context, *IAMSys)
}

func (sys *IAMSys) LoadGroup(objAPI ObjectLayer, group string) error {
	if objAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	err := sys.store.loadGroup(group, sys.iamGroupsMap)
	if err != nil && err != errConfigNotFound {
		return err
	}

	if err == errConfigNotFound {
		sys.removeGroupFromMembershipsMap(group)
		delete(sys.iamGroupsMap, group)
		delete(sys.iamGroupPolicyMap, group)
		return nil
	}

	gi := sys.iamGroupsMap[group]

	sys.removeGroupFromMembershipsMap(group)
	sys.updateGroupMembershipsMap(group, &gi)
	return nil
}

func (sys *IAMSys) LoadPolicy(objAPI ObjectLayer, policyName string) error {
	if objAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	return sys.store.loadPolicyDoc(policyName, sys.iamPolicyDocsMap)
}

func (sys *IAMSys) LoadPolicyMapping(objAPI ObjectLayer, userOrGroup string, isGroup bool) error {
	if objAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	var err error
	if isGroup {
		err = sys.store.loadMappedPolicy(userOrGroup, regularUser, isGroup, sys.iamGroupPolicyMap)
	} else {
		err = sys.store.loadMappedPolicy(userOrGroup, regularUser, isGroup, sys.iamUserPolicyMap)
	}

	if err != nil && err != errConfigNotFound {
		return err
	}

	return nil
}

func (sys *IAMSys) LoadUser(objAPI ObjectLayer, accessKey string, userType IAMUserType) error {
	if objAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	err := sys.store.loadUser(accessKey, userType, sys.iamUsersMap)
	if err != nil {
		return err
	}
	err = sys.store.loadMappedPolicy(accessKey, userType, false, sys.iamUserPolicyMap)
	if err != nil && err != errConfigNotFound {
		return err
	}

	return nil
}

func (sys *IAMSys) LoadServiceAccount(accessKey string) error {
	if sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	err := sys.store.loadUser(accessKey, srvAccUser, sys.iamUsersMap)
	if err != nil {
		return err
	}

	return nil
}

func (sys *IAMSys) doIAMConfigMigration(ctx context.Context) error {
	return sys.store.migrateBackendFormat(ctx)
}

func (sys *IAMSys) Init(ctx context.Context, objAPI ObjectLayer) error {
	if objAPI == nil {
		return errServerNotInitialized
	}

	sys.store = newIAMObjectStore(ctx, objAPI)

	if err := sys.doIAMConfigMigration(ctx); err != nil {
		return err
	}

	err := sys.store.loadAll(ctx, sys)

	globalOldCred = auth.Credentials{}

	go sys.store.watch(ctx, sys)
	return err
}

func (sys *IAMSys) DeletePolicy(policyName string) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	if policyName == "" {
		return errInvalidArgument
	}

	sys.store.lock()
	defer sys.store.unlock()

	err := sys.store.deletePolicyDoc(policyName)
	if err == errNoSuchPolicy {
		err = nil
	}

	delete(sys.iamPolicyDocsMap, policyName)

	var usersToDel []string
	var usersType []IAMUserType
	for u, mp := range sys.iamUserPolicyMap {
		if mp.Policy == policyName {
			cr, ok := sys.iamUsersMap[u]
			if !ok {
				return errNoSuchUser
			}
			if cr.IsTemp() {
				usersType = append(usersType, stsUser)
			} else {
				usersType = append(usersType, regularUser)
			}
			usersToDel = append(usersToDel, u)
		}
	}
	for i, u := range usersToDel {
		sys.policyDBSet(u, "", usersType[i], false)
	}

	var groupsToDel []string
	for g, mp := range sys.iamGroupPolicyMap {
		if mp.Policy == policyName {
			groupsToDel = append(groupsToDel, g)
		}
	}
	for _, g := range groupsToDel {
		sys.policyDBSet(g, "", regularUser, true)
	}

	return err
}

func (sys *IAMSys) InfoPolicy(policyName string) (iampolicy.Policy, error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return iampolicy.Policy{}, errServerNotInitialized
	}

	sys.store.rlock()
	defer sys.store.runlock()

	v, ok := sys.iamPolicyDocsMap[policyName]
	if !ok {
		return iampolicy.Policy{}, errNoSuchPolicy
	}

	return v, nil
}

func (sys *IAMSys) ListPolicies() (map[string]iampolicy.Policy, error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return nil, errServerNotInitialized
	}

	sys.store.rlock()
	defer sys.store.runlock()

	policyDocsMap := make(map[string]iampolicy.Policy, len(sys.iamPolicyDocsMap))
	for k, v := range sys.iamPolicyDocsMap {
		policyDocsMap[k] = v
	}

	return policyDocsMap, nil
}

func (sys *IAMSys) SetPolicy(policyName string, p iampolicy.Policy) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	if p.IsEmpty() || policyName == "" {
		return errInvalidArgument
	}

	sys.store.lock()
	defer sys.store.unlock()

	if err := sys.store.savePolicyDoc(policyName, p); err != nil {
		return err
	}

	sys.iamPolicyDocsMap[policyName] = p
	return nil
}

func (sys *IAMSys) DeleteUser(accessKey string) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	userInfo, getErr := sys.GetUserInfo(accessKey)
	if getErr != nil {
		return getErr
	}

	for _, group := range userInfo.MemberOf {
		removeErr := sys.RemoveUsersFromGroup(group, []string{accessKey})
		if removeErr != nil {
			return removeErr
		}
	}

	sys.store.lock()
	defer sys.store.unlock()

	if sys.usersSysType != IPOSUsersSysType {
		return errIAMActionNotAllowed
	}

	sys.store.deleteMappedPolicy(accessKey, regularUser, false)
	err := sys.store.deleteUserIdentity(accessKey, regularUser)
	if err == errNoSuchUser {
		err = nil
	}

	delete(sys.iamUsersMap, accessKey)
	delete(sys.iamUserPolicyMap, accessKey)

	for _, u := range sys.iamUsersMap {
		if u.IsServiceAccount() {
			if u.ParentUser == accessKey {
				_ = sys.store.deleteUserIdentity(u.AccessKey, srvAccUser)
				delete(sys.iamUsersMap, u.AccessKey)
			}
		}
	}

	return err
}

func (sys *IAMSys) SetTempUser(accessKey string, cred auth.Credentials, policyName string) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	u := newUserIdentity(cred)
	if err := sys.store.saveUserIdentity(accessKey, stsUser, u); err != nil {
		return err
	}

	sys.iamUsersMap[accessKey] = cred
	return nil
}

func (sys *IAMSys) ListUsers() (map[string]madmin.UserInfo, error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return nil, errServerNotInitialized
	}

	var users = make(map[string]madmin.UserInfo)

	sys.store.rlock()
	defer sys.store.runlock()

	if sys.usersSysType != IPOSUsersSysType {
		return nil, errIAMActionNotAllowed
	}

	for k, v := range sys.iamUsersMap {
		if !v.IsTemp() && !v.IsServiceAccount() {
			users[k] = madmin.UserInfo{
				PolicyName: sys.iamUserPolicyMap[k].Policy,
				Status: func() madmin.AccountStatus {
					if v.IsValid() {
						return madmin.AccountEnabled
					}
					return madmin.AccountDisabled
				}(),
			}
		}
	}

	return users, nil
}

func (sys *IAMSys) IsTempUser(name string) (bool, error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return false, errServerNotInitialized
	}

	sys.store.rlock()
	defer sys.store.runlock()

	creds, found := sys.iamUsersMap[name]
	if !found {
		return false, errNoSuchUser
	}

	return creds.IsTemp(), nil
}

func (sys *IAMSys) IsServiceAccount(name string) (bool, string, error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return false, "", errServerNotInitialized
	}

	sys.store.rlock()
	defer sys.store.runlock()

	creds, found := sys.iamUsersMap[name]
	if !found {
		return false, "", errNoSuchUser
	}

	if creds.IsServiceAccount() {
		return true, creds.ParentUser, nil
	}

	return false, "", nil
}

func (sys *IAMSys) GetUserInfo(name string) (u madmin.UserInfo, err error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return u, errServerNotInitialized
	}

	sys.store.rlock()
	defer sys.store.runlock()

	if sys.usersSysType != IPOSUsersSysType {
		mappedPolicy, ok1 := sys.iamUserPolicyMap[name]
		memberships, ok2 := sys.iamUserGroupMemberships[name]
		if !ok1 && !ok2 {
			return u, errNoSuchUser
		}
		return madmin.UserInfo{
			PolicyName: mappedPolicy.Policy,
			MemberOf:   memberships.ToSlice(),
		}, nil
	}

	creds, found := sys.iamUsersMap[name]
	if !found {
		return u, errNoSuchUser
	}

	if creds.IsTemp() {
		return u, errIAMActionNotAllowed
	}

	u = madmin.UserInfo{
		PolicyName: sys.iamUserPolicyMap[name].Policy,
		Status: func() madmin.AccountStatus {
			if creds.IsValid() {
				return madmin.AccountEnabled
			}
			return madmin.AccountDisabled
		}(),
		MemberOf: sys.iamUserGroupMemberships[name].ToSlice(),
	}
	return u, nil
}

func (sys *IAMSys) SetUserStatus(accessKey string, status madmin.AccountStatus) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	if status != madmin.AccountEnabled && status != madmin.AccountDisabled {
		return errInvalidArgument
	}

	sys.store.lock()
	defer sys.store.unlock()

	if sys.usersSysType != IPOSUsersSysType {
		return errIAMActionNotAllowed
	}

	cred, ok := sys.iamUsersMap[accessKey]
	if !ok {
		return errNoSuchUser
	}

	if cred.IsTemp() {
		return errIAMActionNotAllowed
	}

	uinfo := newUserIdentity(auth.Credentials{
		AccessKey: accessKey,
		SecretKey: cred.SecretKey,
		Status: func() string {
			if status == madmin.AccountEnabled {
				return madmin.EnableOn
			}
			return madmin.EnableOff
		}(),
	})

	if err := sys.store.saveUserIdentity(accessKey, regularUser, uinfo); err != nil {
		return err
	}

	sys.iamUsersMap[accessKey] = uinfo.Credentials
	return nil
}

func (sys *IAMSys) NewServiceAccount(ctx context.Context, parentUser string, sessionPolicy *iampolicy.Policy) (auth.Credentials, error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return auth.Credentials{}, errServerNotInitialized
	}

	var policyBuf []byte
	if sessionPolicy != nil {
		err := sessionPolicy.Validate()
		if err != nil {
			return auth.Credentials{}, err
		}
		policyBuf, err = json.Marshal(sessionPolicy)
		if err != nil {
			return auth.Credentials{}, err
		}
		if len(policyBuf) > 16*1024 {
			return auth.Credentials{}, fmt.Errorf("Session policy should not exceed 16 KiB characters")
		}
	}

	sys.store.lock()
	defer sys.store.unlock()

	if sys.usersSysType != IPOSUsersSysType {
		return auth.Credentials{}, errIAMActionNotAllowed
	}

	if parentUser == globalActiveCred.AccessKey {
		return auth.Credentials{}, errIAMActionNotAllowed
	}

	cr, ok := sys.iamUsersMap[parentUser]
	if !ok {
		return auth.Credentials{}, errNoSuchUser
	}

	if cr.IsTemp() {
		return auth.Credentials{}, errIAMActionNotAllowed
	}

	m := make(map[string]interface{})

	if len(policyBuf) > 0 {
		m[iampolicy.SessionPolicyName] = base64.StdEncoding.EncodeToString(policyBuf)
		m[iamPolicyClaimNameSA()] = "embedded-policy"
	} else {
		m[iamPolicyClaimNameSA()] = "inherited-policy"
	}

	secret := globalActiveCred.SecretKey
	cred, err := auth.GetNewCredentialsWithMetadata(m, secret)
	if err != nil {
		return auth.Credentials{}, err
	}

	cred.ParentUser = parentUser
	u := newUserIdentity(cred)

	if err := sys.store.saveUserIdentity(u.Credentials.AccessKey, srvAccUser, u); err != nil {
		return auth.Credentials{}, err
	}

	sys.iamUsersMap[u.Credentials.AccessKey] = u.Credentials

	return cred, nil
}

func (sys *IAMSys) ListServiceAccounts(ctx context.Context, accessKey string) ([]string, error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return nil, errServerNotInitialized
	}

	sys.store.rlock()
	defer sys.store.runlock()

	if sys.usersSysType != IPOSUsersSysType {
		return nil, errIAMActionNotAllowed
	}

	var serviceAccounts []string

	for k, v := range sys.iamUsersMap {
		if v.IsServiceAccount() && v.ParentUser == accessKey {
			serviceAccounts = append(serviceAccounts, k)
		}
	}

	return serviceAccounts, nil
}

func (sys *IAMSys) GetServiceAccountParent(ctx context.Context, accessKey string) (string, error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return "", errServerNotInitialized
	}

	sys.store.rlock()
	defer sys.store.runlock()

	if sys.usersSysType != IPOSUsersSysType {
		return "", errIAMActionNotAllowed
	}

	sa, ok := sys.iamUsersMap[accessKey]
	if !ok || !sa.IsServiceAccount() {
		return "", errNoSuchServiceAccount
	}

	return sa.ParentUser, nil
}

func (sys *IAMSys) DeleteServiceAccount(ctx context.Context, accessKey string) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	if sys.usersSysType != IPOSUsersSysType {
		return errIAMActionNotAllowed
	}

	sa, ok := sys.iamUsersMap[accessKey]
	if !ok || !sa.IsServiceAccount() {
		return errNoSuchServiceAccount
	}

	err := sys.store.deleteUserIdentity(accessKey, srvAccUser)
	if err != nil {
		if err == errNoSuchUser {
			return nil
		}
		return err
	}

	delete(sys.iamUsersMap, accessKey)
	return nil
}

func (sys *IAMSys) SetUser(accessKey string, uinfo madmin.UserInfo) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	u := newUserIdentity(auth.Credentials{
		AccessKey: accessKey,
		SecretKey: uinfo.SecretKey,
		Status:    string(uinfo.Status),
	})

	sys.store.lock()
	defer sys.store.unlock()

	if sys.usersSysType != IPOSUsersSysType {
		return errIAMActionNotAllowed
	}

	cr, ok := sys.iamUsersMap[accessKey]
	if cr.IsTemp() && ok {
		return errIAMActionNotAllowed
	}

	if err := sys.store.saveUserIdentity(accessKey, regularUser, u); err != nil {
		return err
	}

	sys.iamUsersMap[accessKey] = u.Credentials

	if uinfo.PolicyName != "" {
		return sys.policyDBSet(accessKey, uinfo.PolicyName, regularUser, false)
	}
	return nil
}

func (sys *IAMSys) SetUserSecretKey(accessKey string, secretKey string) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	if sys.usersSysType != IPOSUsersSysType {
		return errIAMActionNotAllowed
	}

	cred, ok := sys.iamUsersMap[accessKey]
	if !ok {
		return errNoSuchUser
	}

	cred.SecretKey = secretKey
	u := newUserIdentity(cred)
	if err := sys.store.saveUserIdentity(accessKey, regularUser, u); err != nil {
		return err
	}

	sys.iamUsersMap[accessKey] = cred
	return nil
}

func (sys *IAMSys) GetUser(accessKey string) (cred auth.Credentials, ok bool) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return cred, false
	}

	sys.store.rlock()
	defer sys.store.runlock()

	cred, ok = sys.iamUsersMap[accessKey]
	return cred, ok && cred.IsValid()
}

func (sys *IAMSys) AddUsersToGroup(group string, members []string) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	if group == "" {
		return errInvalidArgument
	}

	sys.store.lock()
	defer sys.store.unlock()

	if sys.usersSysType != IPOSUsersSysType {
		return errIAMActionNotAllowed
	}

	for _, member := range members {
		cr, ok := sys.iamUsersMap[member]
		if !ok {
			return errNoSuchUser
		}
		if cr.IsTemp() {
			return errIAMActionNotAllowed
		}
	}

	gi, ok := sys.iamGroupsMap[group]
	if !ok {
		gi = newGroupInfo(members)
	} else {
		mergedMembers := append(gi.Members, members...)
		uniqMembers := set.CreateStringSet(mergedMembers...).ToSlice()
		gi.Members = uniqMembers
	}

	if err := sys.store.saveGroupInfo(group, gi); err != nil {
		return err
	}

	sys.iamGroupsMap[group] = gi

	for _, member := range members {
		gset := sys.iamUserGroupMemberships[member]
		if gset == nil {
			gset = set.CreateStringSet(group)
		} else {
			gset.Add(group)
		}
		sys.iamUserGroupMemberships[member] = gset
	}

	return nil
}

func (sys *IAMSys) RemoveUsersFromGroup(group string, members []string) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	if group == "" {
		return errInvalidArgument
	}

	sys.store.lock()
	defer sys.store.unlock()

	if sys.usersSysType != IPOSUsersSysType {
		return errIAMActionNotAllowed
	}

	for _, member := range members {
		cr, ok := sys.iamUsersMap[member]
		if !ok {
			return errNoSuchUser
		}
		if cr.IsTemp() {
			return errIAMActionNotAllowed
		}
	}

	gi, ok := sys.iamGroupsMap[group]
	if !ok {
		return errNoSuchGroup
	}

	if len(members) == 0 && len(gi.Members) != 0 {
		return errGroupNotEmpty
	}

	if len(members) == 0 {

		if err := sys.store.deleteMappedPolicy(group, regularUser, true); err != nil && err != errNoSuchPolicy {
			return err
		}
		if err := sys.store.deleteGroupInfo(group); err != nil && err != errNoSuchGroup {
			return err
		}

		delete(sys.iamGroupsMap, group)
		delete(sys.iamGroupPolicyMap, group)
		return nil
	}

	s := set.CreateStringSet(gi.Members...)
	d := set.CreateStringSet(members...)
	gi.Members = s.Difference(d).ToSlice()

	err := sys.store.saveGroupInfo(group, gi)
	if err != nil {
		return err
	}
	sys.iamGroupsMap[group] = gi

	for _, member := range members {
		gset := sys.iamUserGroupMemberships[member]
		if gset == nil {
			continue
		}
		gset.Remove(group)
		sys.iamUserGroupMemberships[member] = gset
	}

	return nil
}

func (sys *IAMSys) SetGroupStatus(group string, enabled bool) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	if sys.usersSysType != IPOSUsersSysType {
		return errIAMActionNotAllowed
	}

	if group == "" {
		return errInvalidArgument
	}

	gi, ok := sys.iamGroupsMap[group]
	if !ok {
		return errNoSuchGroup
	}

	if enabled {
		gi.Status = statusEnabled
	} else {
		gi.Status = statusDisabled
	}

	if err := sys.store.saveGroupInfo(group, gi); err != nil {
		return err
	}
	sys.iamGroupsMap[group] = gi
	return nil
}

func (sys *IAMSys) GetGroupDescription(group string) (gd madmin.GroupDesc, err error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return gd, errServerNotInitialized
	}

	ps, err := sys.PolicyDBGet(group, true)
	if err != nil {
		return gd, err
	}

	policy := ""
	if len(ps) > 0 {
		policy = ps[0]
	}

	if sys.usersSysType != IPOSUsersSysType {
		return madmin.GroupDesc{
			Name:   group,
			Policy: policy,
		}, nil
	}

	sys.store.rlock()
	defer sys.store.runlock()

	gi, ok := sys.iamGroupsMap[group]
	if !ok {
		return gd, errNoSuchGroup
	}

	return madmin.GroupDesc{
		Name:    group,
		Status:  gi.Status,
		Members: gi.Members,
		Policy:  policy,
	}, nil
}

func (sys *IAMSys) ListGroups() (r []string, err error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return r, errServerNotInitialized
	}

	sys.store.rlock()
	defer sys.store.runlock()

	if sys.usersSysType != IPOSUsersSysType {
		return nil, errIAMActionNotAllowed
	}

	for k := range sys.iamGroupsMap {
		r = append(r, k)
	}
	return r, nil
}

func (sys *IAMSys) PolicyDBSet(name, policy string, isGroup bool) error {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return errServerNotInitialized
	}

	sys.store.lock()
	defer sys.store.unlock()

	return sys.policyDBSet(name, policy, regularUser, isGroup)
}

func (sys *IAMSys) policyDBSet(name, policy string, userType IAMUserType, isGroup bool) error {
	if name == "" {
		return errInvalidArgument
	}
	if _, ok := sys.iamPolicyDocsMap[policy]; !ok && policy != "" {
		return errNoSuchPolicy
	}

	if sys.usersSysType == IPOSUsersSysType {
		if !isGroup {
			if _, ok := sys.iamUsersMap[name]; !ok {
				return errNoSuchUser
			}
		} else {
			if _, ok := sys.iamGroupsMap[name]; !ok {
				return errNoSuchGroup
			}
		}
	}

	if policy == "" {
		if err := sys.store.deleteMappedPolicy(name, userType, isGroup); err != nil && err != errNoSuchPolicy {
			return err
		}
		if !isGroup {
			delete(sys.iamUserPolicyMap, name)
		} else {
			delete(sys.iamGroupPolicyMap, name)
		}
		return nil
	}

	mp := newMappedPolicy(policy)
	if err := sys.store.saveMappedPolicy(name, userType, isGroup, mp); err != nil {
		return err
	}
	if !isGroup {
		sys.iamUserPolicyMap[name] = mp
	} else {
		sys.iamGroupPolicyMap[name] = mp
	}
	return nil
}

var iamAccountReadAccessActions = iampolicy.NewActionSet(
	iampolicy.ListMultipartUploadPartsAction,
	iampolicy.ListBucketMultipartUploadsAction,
	iampolicy.ListBucketAction,
	iampolicy.HeadBucketAction,
	iampolicy.GetObjectAction,
	iampolicy.GetBucketLocationAction,
)

var iamAccountWriteAccessActions = iampolicy.NewActionSet(
	iampolicy.AbortMultipartUploadAction,
	iampolicy.CreateBucketAction,
	iampolicy.PutObjectAction,
	iampolicy.DeleteObjectAction,
	iampolicy.DeleteBucketAction,
)

var iamAccountOtherAccessActions = iampolicy.NewActionSet(
	iampolicy.BypassGovernanceRetentionAction,
	iampolicy.PutObjectRetentionAction,
	iampolicy.GetObjectRetentionAction,
	iampolicy.GetObjectLegalHoldAction,
	iampolicy.PutObjectLegalHoldAction,
	iampolicy.GetBucketObjectLockConfigurationAction,
	iampolicy.PutBucketObjectLockConfigurationAction,

	iampolicy.ListenBucketNotificationAction,

	iampolicy.PutBucketLifecycleAction,
	iampolicy.GetBucketLifecycleAction,

	iampolicy.PutBucketNotificationAction,
	iampolicy.GetBucketNotificationAction,

	iampolicy.PutBucketPolicyAction,
	iampolicy.DeleteBucketPolicyAction,
	iampolicy.GetBucketPolicyAction,

	iampolicy.PutBucketEncryptionAction,
	iampolicy.GetBucketEncryptionAction,
)

func (sys *IAMSys) GetAccountAccess(accountName, bucket string) (rd, wr, o bool) {
	policies, err := sys.PolicyDBGet(accountName, false)
	if err != nil {
		logger.LogIf(context.Background(), err)
		return false, false, false
	}

	if len(policies) == 0 {
		return false, false, false
	}

	sys.store.rlock()
	defer sys.store.runlock()

	var availablePolicies []iampolicy.Policy
	for _, pname := range policies {
		p, found := sys.iamPolicyDocsMap[pname]
		if found {
			availablePolicies = append(availablePolicies, p)
		}
	}

	if len(availablePolicies) == 0 {
		return false, false, false
	}

	combinedPolicy := availablePolicies[0]
	for i := 1; i < len(availablePolicies); i++ {
		combinedPolicy.Statements = append(combinedPolicy.Statements,
			availablePolicies[i].Statements...)
	}

	allActions := iampolicy.NewActionSet(iampolicy.AllActions)
	for _, st := range combinedPolicy.Statements {
		if st.Effect != policy.Allow {
			continue
		}
		if !st.Actions.Intersection(allActions).IsEmpty() {
			rd, wr, o = true, true, true
			break
		}
		if !st.Actions.Intersection(iamAccountReadAccessActions).IsEmpty() {
			rd = true
		}
		if !st.Actions.Intersection(iamAccountWriteAccessActions).IsEmpty() {
			wr = true
		}
		if !st.Actions.Intersection(iamAccountOtherAccessActions).IsEmpty() {
			o = true
		}
	}

	return
}

func (sys *IAMSys) PolicyDBGet(name string, isGroup bool) ([]string, error) {
	objectAPI := newObjectLayerWithoutSafeModeFn()
	if objectAPI == nil || sys == nil || sys.store == nil {
		return nil, errServerNotInitialized
	}

	if name == "" {
		return nil, errInvalidArgument
	}

	sys.store.rlock()
	defer sys.store.runlock()

	return sys.policyDBGet(name, isGroup)
}

func (sys *IAMSys) policyDBGet(name string, isGroup bool) ([]string, error) {
	if isGroup {
		if _, ok := sys.iamGroupsMap[name]; !ok {
			return nil, errNoSuchGroup
		}

		policy := sys.iamGroupPolicyMap[name]
		if policy.Policy == "" {
			return nil, nil
		}
		return []string{policy.Policy}, nil
	}

	if u, ok := sys.iamUsersMap[name]; !ok {
		return nil, errNoSuchUser
	} else if u.Status == statusDisabled {
		return nil, nil
	}

	result := []string{}
	policy := sys.iamUserPolicyMap[name]
	if policy.Policy != "" {
		result = append(result, policy.Policy)
	}
	for _, group := range sys.iamUserGroupMemberships[name].ToSlice() {
		gi, ok := sys.iamGroupsMap[group]
		if !ok || gi.Status == statusDisabled {
			continue
		}

		p, ok := sys.iamGroupPolicyMap[group]
		if ok && p.Policy != "" {
			result = append(result, p.Policy)
		}
	}
	return result, nil
}

func (sys *IAMSys) IsAllowedServiceAccount(args iampolicy.Args, parent string) bool {
	return false
}

func (sys *IAMSys) IsAllowedSTS(args iampolicy.Args) bool {
	pnameSlice, ok := args.GetPolicies(iamPolicyClaimNameOpenID())
	if !ok {
		return false
	}

	if len(pnameSlice) == 0 {
		return false
	}

	sys.store.rlock()
	defer sys.store.runlock()

	mp, ok := sys.iamUserPolicyMap[args.AccountName]
	if !ok {
		return false
	}
	name := mp.Policy

	if pnameSlice[0] != name {
		return false
	}

	spolicy, ok := args.Claims[iampolicy.SessionPolicyName]
	if ok {
		spolicyStr, ok := spolicy.(string)
		if !ok {
			return false
		}

		subPolicy, err := iampolicy.ParseConfig(bytes.NewReader([]byte(spolicyStr)))
		if err != nil {
			logger.LogIf(context.Background(), err)
			return false
		}

		if subPolicy.Version == "" {
			return false
		}

		p, ok := sys.iamPolicyDocsMap[pnameSlice[0]]
		return ok && p.IsAllowed(args) && subPolicy.IsAllowed(args)
	}

	p, ok := sys.iamPolicyDocsMap[pnameSlice[0]]
	return ok && p.IsAllowed(args)
}

func (sys *IAMSys) IsAllowed(args iampolicy.Args) bool {
	if args.IsOwner {
		return true
	}

	ok, err := sys.IsTempUser(args.AccountName)
	if err != nil {
		logger.LogIf(context.Background(), err)
		return false
	}
	if ok {
		return sys.IsAllowedSTS(args)
	}

	ok, parentUser, err := sys.IsServiceAccount(args.AccountName)
	if err != nil {
		logger.LogIf(context.Background(), err)
		return false
	}
	if ok {
		return sys.IsAllowedServiceAccount(args, parentUser)
	}

	policies, err := sys.PolicyDBGet(args.AccountName, false)
	if err != nil {
		logger.LogIf(context.Background(), err)
		return false
	}

	if len(policies) == 0 {
		return false
	}

	sys.store.rlock()
	defer sys.store.runlock()

	var availablePolicies []iampolicy.Policy
	for _, pname := range policies {
		p, found := sys.iamPolicyDocsMap[pname]
		if found {
			availablePolicies = append(availablePolicies, p)
		}
	}
	if len(availablePolicies) == 0 {
		return false
	}
	combinedPolicy := availablePolicies[0]
	for i := 1; i < len(availablePolicies); i++ {
		combinedPolicy.Statements = append(combinedPolicy.Statements,
			availablePolicies[i].Statements...)
	}
	return combinedPolicy.IsAllowed(args)
}

func setDefaultCannedPolicies(policies map[string]iampolicy.Policy) {
	_, ok := policies["writeonly"]
	if !ok {
		policies["writeonly"] = iampolicy.WriteOnly
	}
	_, ok = policies["readonly"]
	if !ok {
		policies["readonly"] = iampolicy.ReadOnly
	}
	_, ok = policies["readwrite"]
	if !ok {
		policies["readwrite"] = iampolicy.ReadWrite
	}
	_, ok = policies["diagnostics"]
	if !ok {
		policies["diagnostics"] = iampolicy.AdminDiagnostics
	}
}

func (sys *IAMSys) buildUserGroupMemberships() {
	for group, gi := range sys.iamGroupsMap {
		sys.updateGroupMembershipsMap(group, &gi)
	}
}

func (sys *IAMSys) updateGroupMembershipsMap(group string, gi *GroupInfo) {
	if gi == nil {
		return
	}
	for _, member := range gi.Members {
		v := sys.iamUserGroupMemberships[member]
		if v == nil {
			v = set.CreateStringSet(group)
		} else {
			v.Add(group)
		}
		sys.iamUserGroupMemberships[member] = v
	}
}

func (sys *IAMSys) removeGroupFromMembershipsMap(group string) {
	for member, groups := range sys.iamUserGroupMemberships {
		if !groups.Contains(group) {
			continue
		}
		groups.Remove(group)
		sys.iamUserGroupMemberships[member] = groups
	}
}

func NewIAMSys() *IAMSys {
	return &IAMSys{
		usersSysType:            IPOSUsersSysType,
		iamUsersMap:             make(map[string]auth.Credentials),
		iamPolicyDocsMap:        make(map[string]iampolicy.Policy),
		iamUserPolicyMap:        make(map[string]MappedPolicy),
		iamGroupsMap:            make(map[string]GroupInfo),
		iamUserGroupMemberships: make(map[string]set.StringSet),
	}
}
