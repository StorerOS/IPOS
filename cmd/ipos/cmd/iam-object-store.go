package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	jwtgo "github.com/dgrijalva/jwt-go"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/auth"
	iampolicy "github.com/storeros/ipos/pkg/iam/policy"
	"github.com/storeros/ipos/pkg/madmin"
)

type IAMObjectStore struct {
	sync.RWMutex

	ctx    context.Context
	objAPI ObjectLayer
}

func newIAMObjectStore(ctx context.Context, objAPI ObjectLayer) *IAMObjectStore {
	return &IAMObjectStore{ctx: ctx, objAPI: objAPI}
}

func (iamOS *IAMObjectStore) lock() {
	iamOS.Lock()
}

func (iamOS *IAMObjectStore) unlock() {
	iamOS.Unlock()
}

func (iamOS *IAMObjectStore) rlock() {
	iamOS.RLock()
}

func (iamOS *IAMObjectStore) runlock() {
	iamOS.RUnlock()
}

func (iamOS *IAMObjectStore) migrateUsersConfigToV1(ctx context.Context, isSTS bool) error {
	basePrefix := iamConfigUsersPrefix
	if isSTS {
		basePrefix = iamConfigSTSPrefix
	}

	for item := range listIAMConfigItems(ctx, iamOS.objAPI, basePrefix, true) {
		if item.Err != nil {
			return item.Err
		}

		user := item.Item

		{
			oldPolicyPath := pathJoin(basePrefix, user, iamPolicyFile)
			var policyName string
			if err := iamOS.loadIAMConfig(&policyName, oldPolicyPath); err != nil {
				switch err {
				case errConfigNotFound:
				default:
				}

				goto next
			}

			mp := newMappedPolicy(policyName)
			userType := regularUser
			if isSTS {
				userType = stsUser
			}
			if err := iamOS.saveMappedPolicy(user, userType, false, mp); err != nil {
				return err
			}

			iamOS.deleteIAMConfig(oldPolicyPath)
		}
	next:
		identityPath := pathJoin(basePrefix, user, iamIdentityFile)
		var cred auth.Credentials
		if err := iamOS.loadIAMConfig(&cred, identityPath); err != nil {
			switch err {
			case errConfigNotFound:
			default:
			}
			continue
		}

		var zeroCred auth.Credentials
		if cred == zeroCred {
			continue
		}

		cred.AccessKey = user
		u := newUserIdentity(cred)
		if err := iamOS.saveIAMConfig(u, identityPath); err != nil {
			logger.LogIf(context.Background(), err)
			return err
		}

	}
	return nil

}

func (iamOS *IAMObjectStore) migrateToV1(ctx context.Context) error {
	var iamFmt iamFormat
	path := getIAMFormatFilePath()
	if err := iamOS.loadIAMConfig(&iamFmt, path); err != nil {
		switch err {
		case errConfigNotFound:
		default:
			return err
		}
	} else {
		if iamFmt.Version >= iamFormatVersion1 {
			return nil
		}
		return errors.New("got an invalid IAM format version")
	}

	if err := iamOS.migrateUsersConfigToV1(ctx, false); err != nil {
		logger.LogIf(ctx, err)
		return err
	}
	if err := iamOS.migrateUsersConfigToV1(ctx, true); err != nil {
		logger.LogIf(ctx, err)
		return err
	}
	if err := iamOS.saveIAMConfig(newIAMFormatVersion1(), path); err != nil {
		logger.LogIf(ctx, err)
		return err
	}
	return nil
}

func (iamOS *IAMObjectStore) migrateBackendFormat(ctx context.Context) error {
	return iamOS.migrateToV1(ctx)
}

func (iamOS *IAMObjectStore) saveIAMConfig(item interface{}, path string) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if globalConfigEncrypted {
		data, err = madmin.EncryptData(globalActiveCred.String(), data)
		if err != nil {
			return err
		}
	}
	return saveConfig(context.Background(), iamOS.objAPI, path, data)
}

func (iamOS *IAMObjectStore) loadIAMConfig(item interface{}, path string) error {
	data, err := readConfig(iamOS.ctx, iamOS.objAPI, path)
	if err != nil {
		return err
	}
	if globalConfigEncrypted {
		data, err = madmin.DecryptData(globalActiveCred.String(), bytes.NewReader(data))
		if err != nil {
			return err
		}
	}
	return json.Unmarshal(data, item)
}

func (iamOS *IAMObjectStore) deleteIAMConfig(path string) error {
	return deleteConfig(iamOS.ctx, iamOS.objAPI, path)
}

func (iamOS *IAMObjectStore) loadPolicyDoc(policy string, m map[string]iampolicy.Policy) error {
	var p iampolicy.Policy
	err := iamOS.loadIAMConfig(&p, getPolicyDocPath(policy))
	if err != nil {
		if err == errConfigNotFound {
			return errNoSuchPolicy
		}
		return err
	}
	m[policy] = p
	return nil
}

func (iamOS *IAMObjectStore) loadPolicyDocs(ctx context.Context, m map[string]iampolicy.Policy) error {
	for item := range listIAMConfigItems(ctx, iamOS.objAPI, iamConfigPoliciesPrefix, true) {
		if item.Err != nil {
			return item.Err
		}

		policyName := item.Item
		err := iamOS.loadPolicyDoc(policyName, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (iamOS *IAMObjectStore) loadUser(user string, userType IAMUserType, m map[string]auth.Credentials) error {
	var u UserIdentity
	err := iamOS.loadIAMConfig(&u, getUserIdentityPath(user, userType))
	if err != nil {
		if err == errConfigNotFound {
			return errNoSuchUser
		}
		return err
	}

	if u.Credentials.IsExpired() {
		iamOS.deleteIAMConfig(getUserIdentityPath(user, userType))
		iamOS.deleteIAMConfig(getMappedPolicyPath(user, userType, false))
		return nil
	}

	if globalOldCred.IsValid() && u.Credentials.IsServiceAccount() {
		if !globalOldCred.Equal(globalActiveCred) {
			m := jwtgo.MapClaims{}
			stsTokenCallback := func(t *jwtgo.Token) (interface{}, error) {
				return []byte(globalOldCred.SecretKey), nil
			}
			if _, err := jwtgo.ParseWithClaims(u.Credentials.SessionToken, m, stsTokenCallback); err == nil {
				jwt := jwtgo.NewWithClaims(jwtgo.SigningMethodHS512, jwtgo.MapClaims(m))
				if token, err := jwt.SignedString([]byte(globalActiveCred.SecretKey)); err == nil {
					u.Credentials.SessionToken = token
					err := iamOS.saveIAMConfig(&u, getUserIdentityPath(user, userType))
					if err != nil {
						return err
					}
				}
			}
		}
	}

	if u.Credentials.AccessKey == "" {
		u.Credentials.AccessKey = user
	}
	m[user] = u.Credentials
	return nil
}

func (iamOS *IAMObjectStore) loadUsers(ctx context.Context, userType IAMUserType, m map[string]auth.Credentials) error {
	var basePrefix string
	switch userType {
	case srvAccUser:
		basePrefix = iamConfigServiceAccountsPrefix
	case stsUser:
		basePrefix = iamConfigSTSPrefix
	default:
		basePrefix = iamConfigUsersPrefix
	}

	for item := range listIAMConfigItems(ctx, iamOS.objAPI, basePrefix, true) {
		if item.Err != nil {
			return item.Err
		}

		userName := item.Item
		err := iamOS.loadUser(userName, userType, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (iamOS *IAMObjectStore) loadGroup(group string, m map[string]GroupInfo) error {
	var g GroupInfo
	err := iamOS.loadIAMConfig(&g, getGroupInfoPath(group))
	if err != nil {
		if err == errConfigNotFound {
			return errNoSuchGroup
		}
		return err
	}
	m[group] = g
	return nil
}

func (iamOS *IAMObjectStore) loadGroups(ctx context.Context, m map[string]GroupInfo) error {
	for item := range listIAMConfigItems(ctx, iamOS.objAPI, iamConfigGroupsPrefix, true) {
		if item.Err != nil {
			return item.Err
		}

		group := item.Item
		err := iamOS.loadGroup(group, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (iamOS *IAMObjectStore) loadMappedPolicy(name string, userType IAMUserType, isGroup bool,
	m map[string]MappedPolicy) error {

	var p MappedPolicy
	err := iamOS.loadIAMConfig(&p, getMappedPolicyPath(name, userType, isGroup))
	if err != nil {
		if err == errConfigNotFound {
			return errNoSuchPolicy
		}
		return err
	}
	m[name] = p
	return nil
}

func (iamOS *IAMObjectStore) loadMappedPolicies(ctx context.Context, userType IAMUserType, isGroup bool, m map[string]MappedPolicy) error {
	var basePath string
	if isGroup {
		basePath = iamConfigPolicyDBGroupsPrefix
	} else {
		switch userType {
		case srvAccUser:
			basePath = iamConfigPolicyDBServiceAccountsPrefix
		case stsUser:
			basePath = iamConfigPolicyDBSTSUsersPrefix
		default:
			basePath = iamConfigPolicyDBUsersPrefix
		}
	}
	for item := range listIAMConfigItems(ctx, iamOS.objAPI, basePath, false) {
		if item.Err != nil {
			return item.Err
		}

		policyFile := item.Item
		userOrGroupName := strings.TrimSuffix(policyFile, ".json")
		err := iamOS.loadMappedPolicy(userOrGroupName, userType, isGroup, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (iamOS *IAMObjectStore) loadAll(ctx context.Context, sys *IAMSys) error {
	iamUsersMap := make(map[string]auth.Credentials)
	iamGroupsMap := make(map[string]GroupInfo)
	iamPolicyDocsMap := make(map[string]iampolicy.Policy)
	iamUserPolicyMap := make(map[string]MappedPolicy)
	iamGroupPolicyMap := make(map[string]MappedPolicy)

	isIPOSUsersSys := false
	iamOS.rlock()
	if sys.usersSysType == IPOSUsersSysType {
		isIPOSUsersSys = true
	}
	iamOS.runlock()

	if err := iamOS.loadPolicyDocs(ctx, iamPolicyDocsMap); err != nil {
		return err
	}

	if err := iamOS.loadUsers(ctx, stsUser, iamUsersMap); err != nil {
		return err
	}

	if isIPOSUsersSys {
		if err := iamOS.loadUsers(ctx, regularUser, iamUsersMap); err != nil {
			return err
		}
		if err := iamOS.loadUsers(ctx, srvAccUser, iamUsersMap); err != nil {
			return err
		}
		if err := iamOS.loadGroups(ctx, iamGroupsMap); err != nil {
			return err
		}
	}

	if err := iamOS.loadMappedPolicies(ctx, regularUser, false, iamUserPolicyMap); err != nil {
		return err
	}

	if err := iamOS.loadMappedPolicies(ctx, stsUser, false, iamUserPolicyMap); err != nil {
		return err
	}

	if err := iamOS.loadMappedPolicies(ctx, regularUser, true, iamGroupPolicyMap); err != nil {
		return err
	}

	setDefaultCannedPolicies(iamPolicyDocsMap)

	iamOS.lock()
	defer iamOS.unlock()

	sys.iamUsersMap = iamUsersMap
	sys.iamPolicyDocsMap = iamPolicyDocsMap
	sys.iamUserPolicyMap = iamUserPolicyMap
	sys.iamGroupPolicyMap = iamGroupPolicyMap
	sys.iamGroupsMap = iamGroupsMap
	sys.buildUserGroupMemberships()

	return nil
}

func (iamOS *IAMObjectStore) savePolicyDoc(policyName string, p iampolicy.Policy) error {
	return iamOS.saveIAMConfig(&p, getPolicyDocPath(policyName))
}

func (iamOS *IAMObjectStore) saveMappedPolicy(name string, userType IAMUserType, isGroup bool, mp MappedPolicy) error {
	return iamOS.saveIAMConfig(mp, getMappedPolicyPath(name, userType, isGroup))
}

func (iamOS *IAMObjectStore) saveUserIdentity(name string, userType IAMUserType, u UserIdentity) error {
	return iamOS.saveIAMConfig(u, getUserIdentityPath(name, userType))
}

func (iamOS *IAMObjectStore) saveGroupInfo(name string, gi GroupInfo) error {
	return iamOS.saveIAMConfig(gi, getGroupInfoPath(name))
}

func (iamOS *IAMObjectStore) deletePolicyDoc(name string) error {
	err := iamOS.deleteIAMConfig(getPolicyDocPath(name))
	if err == errConfigNotFound {
		err = errNoSuchPolicy
	}
	return err
}

func (iamOS *IAMObjectStore) deleteMappedPolicy(name string, userType IAMUserType, isGroup bool) error {
	err := iamOS.deleteIAMConfig(getMappedPolicyPath(name, userType, isGroup))
	if err == errConfigNotFound {
		err = errNoSuchPolicy
	}
	return err
}

func (iamOS *IAMObjectStore) deleteUserIdentity(name string, userType IAMUserType) error {
	err := iamOS.deleteIAMConfig(getUserIdentityPath(name, userType))
	if err == errConfigNotFound {
		err = errNoSuchUser
	}
	return err
}

func (iamOS *IAMObjectStore) deleteGroupInfo(name string) error {
	err := iamOS.deleteIAMConfig(getGroupInfoPath(name))
	if err == errConfigNotFound {
		err = errNoSuchGroup
	}
	return err
}

type itemOrErr struct {
	Item string
	Err  error
}

func listIAMConfigItems(ctx context.Context, objAPI ObjectLayer, pathPrefix string, dirs bool) <-chan itemOrErr {
	ch := make(chan itemOrErr)

	dirList := func(lo ListObjectsInfo) []string {
		return lo.Prefixes
	}
	filesList := func(lo ListObjectsInfo) (r []string) {
		for _, o := range lo.Objects {
			r = append(r, o.Name)
		}
		return r
	}

	go func() {
		defer close(ch)

		marker := ""
		for {
			lo, err := objAPI.ListObjects(context.Background(),
				iposMetaBucket, pathPrefix, marker, SlashSeparator, maxObjectList)
			if err != nil {
				select {
				case ch <- itemOrErr{Err: err}:
				case <-ctx.Done():
				}
				return
			}

			marker = lo.NextMarker
			lister := dirList(lo)
			if !dirs {
				lister = filesList(lo)
			}
			for _, itemPrefix := range lister {
				item := strings.TrimPrefix(itemPrefix, pathPrefix)
				item = strings.TrimSuffix(item, SlashSeparator)
				select {
				case ch <- itemOrErr{Item: item}:
				case <-ctx.Done():
					return
				}
			}
			if !lo.IsTruncated {
				return
			}
		}
	}()

	return ch
}

func (iamOS *IAMObjectStore) watch(ctx context.Context, sys *IAMSys) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.NewTimer(globalRefreshIAMInterval).C:
			logger.LogIf(ctx, iamOS.loadAll(ctx, sys))
		}
	}
}
