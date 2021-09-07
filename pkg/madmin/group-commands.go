package madmin

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

type GroupAddRemove struct {
	Group    string   `json:"group"`
	Members  []string `json:"members"`
	IsRemove bool     `json:"isRemove"`
}

func (adm *AdminClient) UpdateGroupMembers(ctx context.Context, g GroupAddRemove) error {
	data, err := json.Marshal(g)
	if err != nil {
		return err
	}

	reqData := requestData{
		relPath: adminAPIPrefix + "/update-group-members",
		content: data,
	}

	resp, err := adm.executeMethod(ctx, http.MethodPut, reqData)

	defer closeResponse(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return httpRespToErrorResponse(resp)
	}

	return nil
}

type GroupDesc struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Members []string `json:"members"`
	Policy  string   `json:"policy"`
}

func (adm *AdminClient) GetGroupDescription(ctx context.Context, group string) (*GroupDesc, error) {
	v := url.Values{}
	v.Set("group", group)
	reqData := requestData{
		relPath:     adminAPIPrefix + "/group",
		queryValues: v,
	}

	resp, err := adm.executeMethod(ctx, http.MethodGet, reqData)
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	gd := GroupDesc{}
	if err = json.Unmarshal(data, &gd); err != nil {
		return nil, err
	}

	return &gd, nil
}

func (adm *AdminClient) ListGroups(ctx context.Context) ([]string, error) {
	reqData := requestData{
		relPath: adminAPIPrefix + "/groups",
	}

	resp, err := adm.executeMethod(ctx, http.MethodGet, reqData)
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	groups := []string{}
	if err = json.Unmarshal(data, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

type GroupStatus string

const (
	GroupEnabled  GroupStatus = "enabled"
	GroupDisabled GroupStatus = "disabled"
)

func (adm *AdminClient) SetGroupStatus(ctx context.Context, group string, status GroupStatus) error {
	v := url.Values{}
	v.Set("group", group)
	v.Set("status", string(status))

	reqData := requestData{
		relPath:     adminAPIPrefix + "/set-group-status",
		queryValues: v,
	}

	resp, err := adm.executeMethod(ctx, http.MethodPut, reqData)
	defer closeResponse(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return httpRespToErrorResponse(resp)
	}

	return nil
}
