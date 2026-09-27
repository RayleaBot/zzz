package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// sdkHost runs the plugin through the SDK runtime as the host does: it writes
// the host's frames, answers the actions the plugin asks for and sends the
// scheduler triggers of the jobs the plugin created in the host's shape.
type sdkHost struct {
	t      *testing.T
	writer *io.PipeWriter
	lines  chan []byte
	next   int
	// service answers plugin.call: a result, or a failure code. scheduled
	// marks a call of a scheduler trigger.
	service func(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string)
	// jobs are the payloads of the scheduler jobs by task ID; created are
	// the scheduler.create requests and deleted the task IDs
	// scheduler.delete removed.
	jobs    map[string]map[string]any
	created []map[string]any
	deleted []string
	// sent are the message.send actions, not counting terminal replies.
	sent []rayleabot.MessageSendRequest
	// triggers are the request IDs of scheduler triggers.
	triggers map[string]bool
}

// hostAction is an action the plugin asked for during one event.
type hostAction struct {
	Name string
	Data map[string]any
}

// newSDKHost starts the plugin with the host's init: bot "bot" on adapter
// "a", prefix "%" and four concurrent events.
func newSDKHost(t *testing.T, a *App, service func(rayleabot.ServiceCallRequest, bool) (map[string]any, string)) *sdkHost {
	t.Helper()
	inReader, hostWriter := io.Pipe()
	hostReader, outWriter := io.Pipe()
	finished := make(chan error, 1)
	go func() {
		finished <- rayleabot.Run(context.Background(), rayleabot.Options{Stdin: inReader, Stdout: outWriter, Stderr: io.Discard}, a)
		outWriter.Close()
	}()
	h := &sdkHost{t: t, writer: hostWriter, lines: make(chan []byte, 64), service: service, jobs: map[string]map[string]any{}, triggers: map[string]bool{}}
	t.Cleanup(func() {
		hostWriter.Close()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("SDK did not stop")
		}
		hostReader.Close()
	})
	go func() {
		scanner := bufio.NewScanner(hostReader)
		scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			h.lines <- bytes.Clone(scanner.Bytes())
		}
		close(h.lines)
	}()
	h.exchange("init", map[string]any{"type": "init", "request_id": "init", "protocol_version": rayleabot.ProtocolVersion, "plugin_id": "raylea.zzz", "bots": []any{map[string]any{"source_adapter": "a", "source_protocol": "onebot11", "id": "bot"}}, "config": map[string]any{}, "super_admins": []string{}, "command_prefixes": []string{"%"}, "timezone": "Asia/Shanghai", "concurrency": 4})
	return h
}

func (h *sdkHost) write(frame map[string]any) {
	h.t.Helper()
	raw, err := json.Marshal(frame)
	if err != nil {
		h.t.Fatal(err)
	}
	if _, err := h.writer.Write(append(raw, '\n')); err != nil {
		h.t.Fatal(err)
	}
}

// exchange sends a frame and answers the plugin's actions until it ends the
// request; it returns the ending frame and the actions asked for.
func (h *sdkHost) exchange(id string, frame map[string]any) (map[string]any, []hostAction) {
	h.t.Helper()
	h.write(frame)
	var actions []hostAction
	for {
		var raw []byte
		select {
		case raw = <-h.lines:
			if raw == nil {
				h.t.Fatal("SDK closed unexpectedly")
			}
		case <-time.After(5 * time.Second):
			h.t.Fatal("SDK response timed out")
		}
		var response map[string]any
		if json.Unmarshal(raw, &response) != nil {
			h.t.Fatal("invalid SDK frame")
		}
		if response["request_id"] == id {
			return response, actions
		}
		if response["type"] != "action" {
			continue
		}
		data := asObject(response["data"])
		actions = append(actions, hostAction{Name: asText(response["action"]), Data: data})
		result, failure := h.answer(asText(response["action"]), data, h.triggers[asText(response["parent_request_id"])])
		if failure != "" {
			h.write(map[string]any{"type": "error", "request_id": response["request_id"], "code": failure, "message": failure})
			continue
		}
		h.write(map[string]any{"type": "result", "request_id": response["request_id"], "status": "success", "data": result})
	}
}

// answer carries out an action as the host would.
func (h *sdkHost) answer(action string, data map[string]any, scheduled bool) (map[string]any, string) {
	switch action {
	case "plugin.call":
		var request rayleabot.ServiceCallRequest
		if decodeObject(data, &request) != nil {
			h.t.Fatal("invalid plugin.call")
		}
		return h.service(request, scheduled)
	case "scheduler.create":
		h.jobs[asText(data["task_id"])] = maps.Clone(asObject(data["payload"]))
		h.created = append(h.created, data)
		return map[string]any{"task_id": data["task_id"]}, ""
	case "scheduler.delete":
		id := asText(data["task_id"])
		if id == "" {
			// The host stops a plugin that deletes a job without an ID.
			h.t.Error("scheduler.delete without a task ID")
		}
		_, exists := h.jobs[id]
		delete(h.jobs, id)
		h.deleted = append(h.deleted, id)
		return map[string]any{"task_id": id, "deleted": exists}, ""
	case "message.send":
		var request rayleabot.MessageSendRequest
		if decodeObject(data, &request) != nil {
			h.t.Fatal("invalid message.send")
		}
		h.sent = append(h.sent, request)
		return map[string]any{"message_id": fmt.Sprintf("sent-%d", len(h.sent)), "delivery_kind": "send"}, ""
	case "render.image":
		return nil, "platform.render_unavailable"
	case "logger.write":
		return map[string]any{}, ""
	}
	h.t.Fatalf("unexpected action %s", action)
	return nil, ""
}

// chat sends a chat message of user "u", a group administrator in a group,
// into target with the host's parsed command, if any.
func (h *sdkHost) chat(target map[string]any, text, command string, args ...string) (map[string]any, []hostAction) {
	h.t.Helper()
	h.next++
	id := fmt.Sprintf("chat-%d", h.next)
	event := map[string]any{"event_id": id, "event_type": "message." + asText(target["type"]), "source_protocol": "onebot11", "source_adapter": "a", "timestamp": time.Now().Unix(), "actor": map[string]any{"id": "u", "role": "admin"}, "target": target, "message": map[string]any{"plain_text": text, "segments": []any{map[string]any{"type": "text", "data": map[string]any{"text": text}}}}}
	if command != "" {
		event["payload"] = map[string]any{"command": command, "args": args}
	}
	return h.exchange(id, map[string]any{"type": "event", "request_id": id, "event": event})
}

// message sends a private chat message of user "u".
func (h *sdkHost) message(text, command string, args ...string) (map[string]any, []hostAction) {
	h.t.Helper()
	return h.chat(map[string]any{"type": "private", "id": "u"}, text, command, args...)
}

// groupMessage sends a message of user "u" in group "g".
func (h *sdkHost) groupMessage(text, command string, args ...string) (map[string]any, []hostAction) {
	h.t.Helper()
	return h.chat(map[string]any{"type": "group", "id": "g"}, text, command, args...)
}

// manage runs an action of the management page.
func (h *sdkHost) manage(action string, input map[string]any) (map[string]any, []hostAction) {
	h.t.Helper()
	h.next++
	id := fmt.Sprintf("manage-%d", h.next)
	return h.exchange(id, map[string]any{"type": "event", "request_id": id, "event": map[string]any{"event_id": id, "event_type": "management.action", "source_protocol": "management", "source_adapter": "management.ui", "timestamp": time.Now().Unix(), "payload": map[string]any{"action": action, "payload": input}}})
}

// trigger runs a job the plugin created as the host's scheduler does: the
// event has no target, and its payload holds the job's payload (and its
// action) but not the task ID.
func (h *sdkHost) trigger(taskID string) (map[string]any, []hostAction) {
	h.t.Helper()
	job, exists := h.jobs[taskID]
	if !exists {
		h.t.Fatalf("no job %s", taskID)
	}
	h.next++
	id := fmt.Sprintf("scheduler-%d", h.next)
	h.triggers[id] = true
	payload := map[string]any{"payload": maps.Clone(job)}
	if action, ok := job["action"].(string); ok && action != "" {
		payload["action"] = action
	}
	return h.exchange(id, map[string]any{"type": "event", "request_id": id, "event": map[string]any{"event_id": "scheduler-" + taskID + "-" + id, "event_type": "scheduler.trigger", "source_protocol": "scheduler", "source_adapter": "scheduler.internal", "timestamp": time.Now().Unix(), "payload": payload}})
}

// job is the only job of the plugin whose task ID starts with prefix.
func (h *sdkHost) job(prefix string) string {
	h.t.Helper()
	found := []string{}
	for id := range h.jobs {
		if len(id) >= len(prefix) && id[:len(prefix)] == prefix {
			found = append(found, id)
		}
	}
	if len(found) != 1 {
		h.t.Fatalf("jobs %v", found)
	}
	return found[0]
}

// terminalText is the text of a message an event ended with.
func terminalText(frame map[string]any) string {
	var message rayleabot.MessageOut
	_ = decodeObject(asObject(frame["data"])["message"], &message)
	return sentText(message)
}

// sentText is the text of a message.
func sentText(message rayleabot.MessageOut) string {
	var text string
	for _, segment := range message.Segments {
		text += asText(segment.Data["text"])
	}
	return text
}
