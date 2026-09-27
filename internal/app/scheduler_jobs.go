package app

import rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"

// timedTaskLabels name the timed tasks signinTask creates, by kind.
var timedTaskLabels = map[string]string{"signin": "每日签到", "monthly": "每日月报收集", "community": "米游社任务", "cloudgame": "云游戏签到"}

// reminderJob is the scheduler job of a stored reminder or timed task.
func (a *App) reminderJob(task Reminder) rayleabot.SchedulerCreateRequest {
	kind, cron, label := "stamina_reminder", "*/10 * * * *", "体力提醒"
	switch task.Kind {
	case "challenge":
		challenge, _ := challengeKind(task.ChallengeKind)
		label = challenge.Label
		if task.Pair {
			label = "式舆/危局"
		}
		kind, cron, label = "challenge_reminder", "*/5 * * * *", label+"挑战提醒"
	case "signin", "monthly":
		kind, label = task.Kind, timedTaskLabels[task.Kind]
	case "community", "cloudgame":
		kind, cron, label = task.Kind, "* * * * *", timedTaskLabels[task.Kind]
	}
	return rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: cron, LogLabel: a.Game.Name + label, Payload: map[string]any{"kind": kind}}
}

// contentJob is the scheduler job of a group's 米游社 pushes.
func (a *App) contentJob(ref string) rayleabot.SchedulerCreateRequest {
	return rayleabot.SchedulerCreateRequest{TaskID: ref, Cron: "*/5 * * * *", LogLabel: a.Game.Name + "米游社推送", Payload: map[string]any{"kind": "public_content"}}
}
