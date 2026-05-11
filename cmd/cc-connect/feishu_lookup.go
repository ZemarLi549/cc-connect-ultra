package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ZemarLi549/cc-connect-ultra/config"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type feishuLookupCreds struct {
	appID     string
	appSecret string
	domain    string
}

type feishuUserMatch struct {
	OpenID   string `json:"open_id"`
	Name     string `json:"name"`
	EnName   string `json:"en_name,omitempty"`
	Nickname string `json:"nickname,omitempty"`
}

type feishuChatMatch struct {
	ChatID string `json:"chat_id"`
	Name   string `json:"name"`
}

func lookupFeishuIDs(projectName, query string, limit int) (map[string]any, error) {
	projectName = strings.TrimSpace(projectName)
	query = strings.TrimSpace(query)
	if projectName == "" {
		return nil, fmt.Errorf("project name required")
	}
	if query == "" {
		return nil, fmt.Errorf("query required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	creds, err := findProjectFeishuCreds(projectName)
	if err != nil {
		return nil, err
	}

	var opts []lark.ClientOptionFunc
	if strings.TrimSpace(creds.domain) != "" && !strings.EqualFold(strings.TrimSpace(creds.domain), lark.FeishuBaseUrl) {
		opts = append(opts, lark.WithOpenBaseUrl(strings.TrimSpace(creds.domain)))
	}
	client := lark.NewClient(creds.appID, creds.appSecret, opts...)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	users, userWarnings := lookupFeishuUsers(ctx, client, query, limit)
	chats, chatWarnings, err := lookupFeishuChats(ctx, client, query, limit)
	if err != nil {
		return nil, err
	}

	warnings := make([]string, 0, len(userWarnings)+len(chatWarnings))
	warnings = append(warnings, userWarnings...)
	warnings = append(warnings, chatWarnings...)

	return map[string]any{
		"project":  projectName,
		"query":    query,
		"users":    users,
		"chats":    chats,
		"warnings": warnings,
	}, nil
}

func findProjectFeishuCreds(projectName string) (feishuLookupCreds, error) {
	cfg := config.GetProjectConfigDetails(projectName)
	if cfg == nil {
		return feishuLookupCreds{}, fmt.Errorf("project %q not found in config", projectName)
	}
	raw, ok := cfg["platform_configs"]
	if !ok {
		return feishuLookupCreds{}, fmt.Errorf("project %q has no platform configs", projectName)
	}
	for _, pc := range normalizeConfigSlice(raw) {
		typ := strings.ToLower(strings.TrimSpace(stringAny(pc["type"])))
		if typ != "feishu" && typ != "lark" {
			continue
		}
		opts, _ := pc["options"].(map[string]any)
		appID := strings.TrimSpace(stringAny(opts["app_id"]))
		appSecret := strings.TrimSpace(stringAny(opts["app_secret"]))
		if appID == "" || appSecret == "" {
			continue
		}
		return feishuLookupCreds{
			appID:     appID,
			appSecret: appSecret,
			domain:    strings.TrimSpace(stringAny(opts["domain"])),
		}, nil
	}
	return feishuLookupCreds{}, fmt.Errorf("project %q has no usable feishu/lark app_id + app_secret", projectName)
}

func lookupFeishuChats(ctx context.Context, client *lark.Client, query string, limit int) ([]feishuChatMatch, []string, error) {
	resp, err := client.Im.Chat.Search(ctx, larkim.NewSearchChatReqBuilder().
		UserIdType("open_id").
		Query(query).
		PageSize(limit).
		Build())
	if err != nil {
		return nil, nil, fmt.Errorf("feishu chat search failed: %w", err)
	}
	if !resp.Success() {
		return nil, nil, fmt.Errorf("feishu chat search failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	out := make([]feishuChatMatch, 0, limit)
	if resp.Data != nil {
		for _, item := range resp.Data.Items {
			if item == nil || item.ChatId == nil || strings.TrimSpace(*item.ChatId) == "" {
				continue
			}
			name := ""
			if item.Name != nil {
				name = *item.Name
			}
			out = append(out, feishuChatMatch{
				ChatID: strings.TrimSpace(*item.ChatId),
				Name:   strings.TrimSpace(name),
			})
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil, nil
}

func lookupFeishuUsers(ctx context.Context, client *lark.Client, query string, limit int) ([]feishuUserMatch, []string) {
	const (
		maxDepartments = 25
		pageSize       = 100
	)
	warnings := []string{}
	usersByID := map[string]feishuUserMatch{}

	depts := []string{}
	deptResp, err := client.Contact.Department.List(ctx, larkcontact.NewListDepartmentReqBuilder().
		DepartmentIdType("department_id").
		ParentDepartmentId("0").
		FetchChild(true).
		PageSize(pageSize).
		Build())
	if err != nil || !deptResp.Success() || deptResp.Data == nil {
		warnings = append(warnings, "user lookup: failed to list departments, check contact permissions")
		return []feishuUserMatch{}, warnings
	}
	for _, dep := range deptResp.Data.Items {
		if dep == nil {
			continue
		}
		if dep.DepartmentId != nil && strings.TrimSpace(*dep.DepartmentId) != "" {
			depts = append(depts, strings.TrimSpace(*dep.DepartmentId))
		}
		if len(depts) >= maxDepartments {
			break
		}
	}
	if len(depts) == 0 {
		depts = []string{"0"}
	}

	needle := strings.ToLower(query)
	for _, depID := range depts {
		resp, err := client.Contact.User.FindByDepartment(ctx, larkcontact.NewFindByDepartmentUserReqBuilder().
			DepartmentIdType("department_id").
			DepartmentId(depID).
			UserIdType("open_id").
			PageSize(pageSize).
			Build())
		if err != nil || !resp.Success() || resp.Data == nil {
			continue
		}
		for _, u := range resp.Data.Items {
			if u == nil || u.OpenId == nil || strings.TrimSpace(*u.OpenId) == "" {
				continue
			}
			name := strings.TrimSpace(stringPtr(u.Name))
			enName := strings.TrimSpace(stringPtr(u.EnName))
			nickname := strings.TrimSpace(stringPtr(u.Nickname))
			joined := strings.ToLower(name + " " + enName + " " + nickname)
			if !strings.Contains(joined, needle) {
				continue
			}
			id := strings.TrimSpace(*u.OpenId)
			usersByID[id] = feishuUserMatch{
				OpenID:   id,
				Name:     name,
				EnName:   enName,
				Nickname: nickname,
			}
			if len(usersByID) >= limit {
				break
			}
		}
		if len(usersByID) >= limit {
			break
		}
	}

	out := make([]feishuUserMatch, 0, len(usersByID))
	for _, u := range usersByID {
		out = append(out, u)
	}
	if len(out) == 0 {
		warnings = append(warnings, "user lookup: no matches or insufficient contact permission scope")
	}
	return out, warnings
}

func normalizeConfigSlice(raw any) []map[string]any {
	switch v := raw.(type) {
	case []map[string]any:
		return v
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func stringAny(v any) string {
	s, _ := v.(string)
	return s
}

func stringPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
