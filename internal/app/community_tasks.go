package app

import (
	"fmt"
	"slices"
	"time"
)

type CommunityStep struct {
	Action string `json:"action"`
	PostID string `json:"post_id,omitempty"`
	Final  bool   `json:"final,omitempty"`
}
type CommunityPlan struct {
	Read      bool            `json:"read"`
	Like      bool            `json:"like"`
	Share     bool            `json:"share"`
	Unlike    bool            `json:"unlike"`
	Steps     []CommunityStep `json:"steps,omitempty"`
	Cursor    int             `json:"cursor"`
	State     string          `json:"state,omitempty"`
	Pending   bool            `json:"pending,omitempty"`
	ReadCount int             `json:"read_count"`
	LikeCount int             `json:"like_count"`
	NeedShare bool            `json:"need_share"`
	Remaining string          `json:"remaining,omitempty"`
}

func stopAccountRound(task *Reminder, now int64, code string) {
	task.LastCode = code
	task.Community.Pending = false
	task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, 0)
	if task.Once {
		task.Enabled = false
	}
	if slices.Contains([]string{"plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_subject_denied", "plugin.upstream_auth_invalid", "plugin.account_cloud_unavailable", "plugin.upstream_cloud_unavailable", "plugin.upstream_community_unavailable"}, code) {
		task.Enabled = false
	}
	if code == "plugin.upstream_device_required" || code == "plugin.upstream_challenge_required" {
		task.NextCheckMS = max(task.NextCheckMS, now+int64(24*time.Hour/time.Millisecond))
	}
}
func communityNeeds(data map[string]any) (bool, int, int, bool) {
	sign, read, like, share := true, 3, 5, true
	for _, raw := range asList(data["states"]) {
		m := asObject(raw)
		done := m["is_get_award"] == true
		n := max(0, number(m["happened_times"]))
		switch number(m["mission_id"]) {
		case 58:
			sign = !done
		case 59:
			read = max(0, 3-n)
			if done {
				read = 0
			}
		case 60:
			like = max(0, 5-n)
			if done {
				like = 0
			}
		case 61:
			share = !done
		}
	}
	return sign, read, like, share
}
func (s *ReminderStore) tickCommunity(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	plan := &task.Community
	today := time.UnixMilli(now).In(time.FixedZone("UTC+8", 28800)).Format("2006-01-02")
	if plan.Pending {
		plan.State = "failed"
		stopAccountRound(task, now, "step_outcome_unknown")
		return s.save(*task)
	}
	if plan.State != "running" {
		if task.LastSignDay == today && !task.Once {
			task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, 0)
			return s.save(*task)
		}
		plan.Steps = []CommunityStep{{Action: "status"}}
		plan.Cursor = 0
		plan.State = "running"
		plan.Remaining = ""
		task.LastSignDay = today
	}
	if plan.Cursor < 0 || plan.Cursor >= len(plan.Steps) || len(plan.Steps) > 20 {
		plan.State = "failed"
		stopAccountRound(task, now, "plugin.game_task_invalid")
		return s.save(*task)
	}
	current := *task
	step := plan.Steps[plan.Cursor]
	plan.Cursor++
	plan.Pending = true
	task.LastCheckedMS = now
	task.NextCheckMS = now + int64(time.Minute/time.Millisecond)
	task.LastCode = "step_attempted"
	if err := s.save(*task); err != nil {
		return err
	}
	result, err := query(current)
	plan.Pending = false
	if err != nil {
		plan.State = "failed"
		stopAccountRound(task, now, PublicError(err).Code)
		return s.save(*task)
	}
	if step.Action == "status" {
		plan.Remaining = asText(result.Data["can_get_points"])
		if plan.Remaining == "" {
			plan.State = "failed"
			stopAccountRound(task, now, "plugin.game_community_invalid")
			return s.save(*task)
		}
		if !step.Final && plan.Remaining != "0" {
			sign, read, like, share := communityNeeds(result.Data)
			plan.ReadCount = 0
			plan.LikeCount = 0
			plan.NeedShare = plan.Share && share
			if plan.Read {
				plan.ReadCount = read
			}
			if plan.Like {
				plan.LikeCount = like
			}
			if sign {
				plan.Steps = append(plan.Steps, CommunityStep{Action: "sign"})
			}
			if plan.ReadCount+plan.LikeCount > 0 || plan.NeedShare {
				plan.Steps = append(plan.Steps, CommunityStep{Action: "posts"})
			} else {
				plan.Steps = append(plan.Steps, CommunityStep{Action: "status", Final: true})
			}
		}
	} else if step.Action == "posts" {
		posts := asList(result.Data["posts"])
		needed := max(plan.ReadCount, plan.LikeCount)
		if plan.NeedShare {
			needed = max(needed, 1)
		}
		if len(posts) < needed {
			plan.State = "failed"
			stopAccountRound(task, now, "plugin.game_community_posts_missing")
			return s.save(*task)
		}
		for i := 0; i < needed; i++ {
			id := asText(asObject(posts[i])["post_id"])
			if id == "" {
				plan.State = "failed"
				stopAccountRound(task, now, "plugin.game_community_posts_missing")
				return s.save(*task)
			}
			if i < plan.ReadCount {
				plan.Steps = append(plan.Steps, CommunityStep{Action: "read", PostID: id})
			}
			if i < plan.LikeCount {
				plan.Steps = append(plan.Steps, CommunityStep{Action: "like", PostID: id})
				if plan.Unlike {
					plan.Steps = append(plan.Steps, CommunityStep{Action: "unlike", PostID: id})
				}
			}
			if i == 0 && plan.NeedShare {
				plan.Steps = append(plan.Steps, CommunityStep{Action: "share", PostID: id})
			}
		}
		plan.Steps = append(plan.Steps, CommunityStep{Action: "status", Final: true})
	} else if result.Data["accepted"] != true {
		plan.State = "failed"
		stopAccountRound(task, now, "step_outcome_unknown")
		return s.save(*task)
	}
	task.LastCode = "step_completed"
	if plan.Cursor >= len(plan.Steps) {
		plan.State = "completed"
		code := "completed"
		if plan.Remaining != "0" {
			code = "completed_with_remaining"
		}
		stopAccountRound(task, now, code)
		if task.Notify {
			task.LastAttemptMS = now
		}
		if err := s.save(*task); err != nil {
			return err
		}
		if task.Notify {
			message := fmt.Sprintf("%s社区任务本轮已结束\n账号 %s\n官方显示今日可继续获取 %s 米游币。", game.Name, task.Role.Nickname, plan.Remaining)
			if err := send(*task, message); err != nil {
				task.LastCode += ".notification_failed"
				return s.save(*task)
			}
		}
		return nil
	}
	return s.save(*task)
}
func (s *ReminderStore) tickCloudGame(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	today := time.UnixMilli(now).In(time.FixedZone("UTC+8", 28800)).Format("2006-01-02")
	if task.LastSignDay == today {
		task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, 0)
		if task.Once {
			task.Enabled = false
		}
		return s.save(*task)
	}
	task.LastSignDay = today
	task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, 0)
	task.LastCheckedMS = now
	task.LastCode = "cloud_attempted"
	if err := s.save(*task); err != nil {
		return err
	}
	result, err := query(*task)
	if err != nil {
		stopAccountRound(task, now, PublicError(err).Code)
		return s.save(*task)
	}
	stopAccountRound(task, now, "cloud_completed")
	if task.Notify {
		task.LastAttemptMS = now
	}
	if err = s.save(*task); err != nil {
		return err
	}
	if task.Notify {
		message := game.Name + "云游戏查询已完成\n账号 " + task.Role.Nickname + "\n免费时长 " + asText(result.Data["free_time"]) + " 分钟 · 本次发放 " + asText(result.Data["send_freetime"]) + " 分钟"
		if err = send(*task, message); err != nil {
			task.LastCode += ".notification_failed"
			return s.save(*task)
		}
	}
	return nil
}
