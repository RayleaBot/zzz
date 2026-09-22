package app

import (
	"bytes"
	"context"
	"encoding/json"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"io"
	"strconv"
)

type CloudExchangeInfo struct {
	UID        string `json:"uid"`
	Characters int    `json:"characters"`
	Direction  string `json:"direction"`
}

func cloudExchangeRequest(game string, input CloudInput) (string, map[string]any, error) {
	if !uidPattern.MatchString(input.UID) {
		return "", nil, gameError("input_invalid", "交换需要有效 UID。")
	}
	body := map[string]any{"version": "0.1.0", "uid": input.UID, "type": cloudGame(game)}
	if input.Mode == "exchange_download" {
		return "panel/download", body, nil
	}
	if input.exchangeData == "" || len(input.exchangeData) > 4*1024*1024 {
		return "", nil, gameError("input_invalid", "请先导入或接收本人的交换档案，再从档案上传。")
	}
	body["data"] = input.exchangeData
	return "panel/upload", body, nil
}
func cloudPlayerObject(value any) (map[string]any, error) {
	if raw, ok := value.(string); ok {
		if len(raw) > 8*1024*1024 {
			return nil, gameError("cloud_invalid", "云交换响应过大。")
		}
		decoder := json.NewDecoder(bytes.NewBufferString(raw))
		decoder.UseNumber()
		if decoder.Decode(&value) != nil || decoder.Decode(new(any)) != io.EOF {
			return nil, gameError("cloud_invalid", "云交换 JSON 无效。")
		}
	}
	data := asObject(value)
	if nested := asObject(data["info"]); nested != nil {
		data = nested
	} else if nested := asObject(data["playerData"]); nested != nil {
		data = nested
	}
	if data == nil {
		return nil, gameError("cloud_invalid", "云交换未返回面板对象。")
	}
	return data, nil
}
func cloudExchangeResult(game Game, input CloudInput, result map[string]any) (CloudResult, error) {
	if input.Mode == "exchange_upload" {
		view := View{Title: game.Name + "面板交换已上传", Subtitle: "UID " + input.UID, Rows: []Row{}, Note: "来源：ark 十分钟交换服务；请在十分钟内用另一实例接收。上传内容为本人明确选择的第三方/导入档案，不代表官方实时面板。"}
		return CloudResult{View: &view, Exchange: &CloudExchangeInfo{UID: input.UID, Direction: "uploaded"}}, nil
	}
	data, err := cloudPlayerObject(result["data"])
	if err != nil {
		return CloudResult{}, err
	}
	avatars, err := cleanCloudPlayer(game.ID, input.UID, data)
	if err != nil {
		return CloudResult{}, err
	}
	raw, err := json.Marshal(map[string]any{"uid": input.UID, "avatars": avatars})
	if err != nil || len(raw) > 4*1024*1024 {
		return CloudResult{}, gameError("history_limit", "云交换档案超过 4 MiB。")
	}
	view := View{Title: game.Name + "交换档案待接收", Subtitle: "UID " + input.UID, Rows: []Row{{Label: "可接收角色", Value: strconv.Itoa(len(avatars))}}, Note: "先选择相同 UID 的已授权账号，再明确合并到本地交换档案。官方实时面板和官方历史保持独立。"}
	return CloudResult{View: &view, Exchange: &CloudExchangeInfo{UID: input.UID, Characters: len(avatars), Direction: "downloaded"}, exchange: raw}, nil
}
func (a *App) cloudArchiveAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	client := a.accountClient(event)
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	role, err := authorizeCloudRole(ctx, client, choice)
	if err != nil {
		return nil, err
	}
	archive, err := a.CloudArchive.Read(client.Provider, choice)
	if err != nil {
		return nil, err
	}
	if archive.UID != "" && (archive.UID != role.UID || archive.Region != role.Region) {
		return nil, gameError("role_missing", "交换档案与授权角色不匹配。")
	}
	if action == "cloud.archive.capture" {
		return a.captureCloudPanel(ctx, client, choice, role, archive, asText(input["character_id"]))
	}
	if action == "cloud.archive.list" {
		return cloudArchiveSummary(archive), nil
	}
	if action == "cloud.archive.get" {
		raw := archive.Avatars[asText(input["character_id"])]
		if len(raw) == 0 {
			return nil, gameError("history_missing", "交换档案没有此角色。")
		}
		data, err := cloudPlayerObject(string(raw))
		if err != nil {
			return nil, err
		}
		result, err := cloudPanelResult(a.Game, CloudInput{Mode: "panel", UID: role.UID}, map[string]any{"data": map[string]any{"uid": role.UID, "avatars": []any{data}}})
		return map[string]any{"panels": result.Panels, "view": result.View}, err
	}

	if action == "cloud.archive.rank" || action == "cloud.archive.refresh" {
		if input["consent"] != true {
			return nil, gameError("cloud_consent_required", "请确认向 ark 发送所选 UID 或面板业务数据。")
		}
		q := CloudInput{Mode: "panel_refresh", UID: role.UID, Consent: true, Authenticated: input["authenticated"] == true}
		if action == "cloud.archive.rank" {
			id := asText(input["character_id"])
			raw := archive.Avatars[id]
			if len(raw) == 0 {
				return nil, gameError("history_missing", "没有所选角色的交换面板。")
			}
			q.Mode = "rank"
			q.CharacterID = id
			q.Query = asText(input["query"])
			q.localPanel = raw
		}
		return a.startCloudJob(ctx, event, q)
	}
	if action == "cloud.archive.upload" {
		if input["consent"] != true {
			return nil, gameError("cloud_consent_required", "请确认向 ark 上传 UID 和全部交换面板。")
		}
		if len(archive.Avatars) == 0 {
			return nil, gameError("history_missing", "没有可上传的交换档案。")
		}
		raw, _ := json.Marshal(map[string]any{"uid": role.UID, "avatars": archive.Avatars})
		return a.startCloudJob(ctx, event, CloudInput{Mode: "exchange_upload", UID: role.UID, Consent: true, Authenticated: input["authenticated"] == true, exchangeData: string(raw)})
	}
	var q struct {
		Revision uint64 `json:"revision"`
		Confirm  bool   `json:"confirm"`
	}
	if decodeObject(input, &q) != nil || !q.Confirm {
		return nil, gameError("input_invalid", "请明确确认交换档案操作。")
	}
	if action == "cloud.archive.remove" {
		err = a.CloudArchive.Update(client.Provider, choice, q.Revision, role, nil, true)
		if err == nil {
			a.CloudTransfers.mu.Lock()
			for ref, t := range a.CloudTransfers.items {
				if t.Provider == client.Provider && t.Selection == choice {
					t.Timer.Stop()
					delete(a.CloudTransfers.items, ref)
				}
			}
			a.CloudTransfers.mu.Unlock()
		}
		return map[string]any{"removed": err == nil}, err
	}
	if action == "cloud.archive.receive" {
		a.Cloud.mu.Lock()
		job := a.Cloud.jobs[asText(input["ref"])]
		if job == nil || job.owner != nil || job.game != a.Game.ID || job.State != "completed" || len(job.exchange) == 0 {
			a.Cloud.mu.Unlock()
			return nil, gameError("cloud_missing", "交换下载已失效，请重新下载。")
		}
		raw := append([]byte(nil), job.exchange...)
		a.Cloud.mu.Unlock()
		data, err := cloudPlayerObject(string(raw))
		if err != nil {
			return nil, err
		}
		avatars, err := cleanCloudPlayer(a.Game.ID, role.UID, data)
		if err != nil {
			return nil, err
		}
		err = a.CloudArchive.Update(client.Provider, choice, q.Revision, role, avatars, false)
		return map[string]any{"received": len(avatars)}, err
	}
	return nil, gameError("operation_denied", "交换档案操作不存在。")
}
