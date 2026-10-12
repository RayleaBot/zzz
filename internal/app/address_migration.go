package app

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/localdata"
)

// Before handling the first event, migrate with the host's bot snapshot.
// A changed snapshot retries unresolved identities. Failed migrations are
// retried before the next event; no event sends while migration has failed.
type addressMigration struct {
	mu    sync.Mutex
	ready bool
	bots  []rayleabot.Bot
}

func (a *App) migrateAddresses(ctx context.Context, event *rayleabot.EventContext) error {
	m := &a.addresses
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ready && slices.Equal(m.bots, event.Bots) {
		return nil
	}
	warn := func(ref string) {
		_, _ = event.Actions().LoggerWrite(ctx, rayleabot.LoggerWriteRequest{Level: "warn", Message: "旧记录缺少机器人账号，原记录已保留；请恢复原连接或重新选择机器人账号。", Fields: map[string]any{"record": ref}})
	}
	if err := a.Groups.migrate(event.Bots, warn); err != nil {
		return err
	}
	if err := migrateTaskOwners(&a.Reminders.taskFiles, event.Bots, func(t *Reminder) *Subject { return &t.Owner }, warn); err != nil {
		return err
	}
	if err := migrateTaskOwners(&a.DailySyncs.taskFiles, event.Bots, func(t *DailySync) *Subject { return &t.Owner }, warn); err != nil {
		return err
	}
	if err := a.Interactions.migrate(event.Bots, warn); err != nil {
		return err
	}
	if err := a.ChallengePrefs.migrate(event.Bots, warn); err != nil {
		return err
	}
	if err := a.Subscriptions.migrate(a.Game.ID, event.Bots, warn, func(ref string) error {
		_, err := event.Actions().SchedulerCreate(ctx, a.contentJob(ref))
		return err
	}, func(ref string) error {
		_, err := event.Actions().SchedulerDelete(ctx, ref)
		return err
	}); err != nil {
		return err
	}
	m.ready, m.bots = true, event.Bots
	return nil
}

func addressFiles(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && !strings.HasPrefix(entry.Name(), ".") && filepath.Ext(entry.Name()) == ".json" {
			paths = append(paths, filepath.Join(directory, entry.Name()))
		}
	}
	return paths, nil
}

func migrateTaskOwners[T storedTask](s *taskFiles[T], bots []rayleabot.Bot, owner func(*T) *Subject, warn func(string)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	paths, err := addressFiles(s.Directory)
	if err != nil {
		return err
	}
	for _, path := range paths {
		var task T
		if err := localdata.Read(path, &task); err != nil {
			return err
		}
		before := *owner(&task)
		*owner(&task) = before.recover(bots)
		if owner(&task).BotID == "" {
			warn(task.taskRef())
		}
		if before != *owner(&task) {
			if err := localdata.Write(path, task); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *GroupStore) migrate(bots []rayleabot.Bot, warn func(string)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	paths, err := addressFiles(s.Directory)
	if err != nil {
		return err
	}
	// Higher revisions win conflicts between old instances. A file already
	// under the account key wins over any legacy file, including after a
	// crash between writing the merged data and archiving its source.
	type legacyGroup struct {
		path string
		data GroupData
	}
	var groups []legacyGroup
	for _, path := range paths {
		var data GroupData
		if err := localdata.Read(path, &data); err != nil {
			return err
		}
		data.Scope.BotID = recoverBot(data.Scope.Protocol, data.Scope.Adapter, data.Scope.BotID, bots)
		if data.Scope.BotID == "" {
			warn(filepath.Base(path))
			continue
		}
		data.Scope = data.Scope.identity()
		if path != s.file(data.Scope) {
			groups = append(groups, legacyGroup{path, data})
		}
	}
	slices.SortStableFunc(groups, func(a, b legacyGroup) int {
		if a.data.Revision > b.data.Revision {
			return -1
		}
		if a.data.Revision < b.data.Revision {
			return 1
		}
		return strings.Compare(a.path, b.path)
	})
	for _, old := range groups {
		path := s.file(old.data.Scope)
		data := old.data
		var current GroupData
		if err := localdata.Read(path, &current); err != nil {
			return err
		}
		if current.Scope.BotID != "" {
			data = mergeGroup(current, data)
		}
		data.Revision++
		if err := localdata.Write(path, data); err != nil {
			return err
		}
		delete(s.configs, data.Scope)
		// Keep all source settings, including conflicting values, reviewable.
		if err := os.Rename(old.path, old.path+".legacy"); err != nil {
			return err
		}
	}
	return nil
}

func mergeGroup(current, old GroupData) GroupData {
	current.Revision = max(current.Revision, old.Revision)
	if current.Config.Enabled == nil {
		current.Config.Enabled = old.Config.Enabled
	}
	if current.Config.ImageReplies == nil {
		current.Config.ImageReplies = old.Config.ImageReplies
	}
	if current.Config.Aliases == nil {
		current.Config.Aliases = map[string]string{}
	}
	for key, value := range old.Config.Aliases {
		if _, exists := current.Config.Aliases[key]; !exists {
			current.Config.Aliases[key] = value
		}
	}
	for _, entry := range old.Rank {
		if !slices.ContainsFunc(current.Rank, func(v RankEntry) bool { return v.UID == entry.UID && v.CharacterID == entry.CharacterID }) {
			current.Rank = append(current.Rank, entry)
		}
	}
	if current.QueryRanks == nil {
		current.QueryRanks = map[string]map[string]QueryRankMember{}
	}
	for kind, members := range old.QueryRanks {
		if current.QueryRanks[kind] == nil {
			current.QueryRanks[kind] = map[string]QueryRankMember{}
		}
		for uid, member := range members {
			if kept, exists := current.QueryRanks[kind][uid]; exists {
				kept.Hidden = kept.Hidden || member.Hidden
				current.QueryRanks[kind][uid] = kept
			} else {
				current.QueryRanks[kind][uid] = member
			}
		}
	}
	return current
}

func (s *ContentSubscriptions) migrate(game string, bots []rayleabot.Bot, warn func(string), create, remove func(string) error) error {
	s.pushMu.Lock()
	defer s.pushMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	merged := []ContentSubscription{}
	changed := false
	for _, item := range items {
		owner := item.Owner.recover(bots)
		changed = changed || item.Owner != owner
		item.Owner = owner
		if owner.BotID == "" {
			warn(item.Ref)
			merged = append(merged, item)
			continue
		}
		ref := contentRef(game, owner, item.GroupID)
		if item.Ref != ref {
			item.RetiredRefs = append(item.RetiredRefs, item.Ref)
			item.Ref, changed = ref, true
		}
		index := slices.IndexFunc(merged, func(v ContentSubscription) bool { return v.Ref == ref })
		if index < 0 {
			merged = append(merged, item)
			continue
		}
		changed = true
		kept := &merged[index]
		for _, kind := range item.Kinds {
			if !slices.Contains(kept.Kinds, kind) {
				kept.Kinds = append(kept.Kinds, kind)
			}
		}
		if kept.Sent == nil {
			kept.Sent = map[string]int64{}
		}
		for post, at := range item.Sent {
			kept.Sent[post] = max(kept.Sent[post], at)
		}
		kept.RetiredRefs = append(kept.RetiredRefs, item.RetiredRefs...)
	}
	// Create replacements before switching refs. A failed create leaves old
	// records intact. Once switched, old triggers cannot find a live record.
	for _, item := range merged {
		if len(item.RetiredRefs) > 0 {
			if err := create(item.Ref); err != nil {
				return err
			}
		}
	}
	if changed {
		if err := localdata.Write(s.Path, merged); err != nil {
			return err
		}
	}
	retired := false
	for i := range merged {
		for _, ref := range merged[i].RetiredRefs {
			if err := remove(ref); err != nil {
				return err
			}
			retired = true
		}
		merged[i].RetiredRefs = nil
	}
	if retired {
		return localdata.Write(s.Path, merged)
	}
	return nil
}

func (s *InteractionStore) migrate(bots []rayleabot.Bot, warn func(string)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	merged := []InteractionProfile{}
	changed := false
	for _, item := range items {
		owner := item.Owner.recover(bots)
		changed = changed || item.Owner != owner
		item.Owner = owner
		if owner.BotID == "" {
			warn(item.Ref)
			merged = append(merged, item)
			continue
		}
		ref := interactionRef(owner)
		changed = changed || item.Ref != ref
		item.Ref = ref
		i := slices.IndexFunc(merged, func(v InteractionProfile) bool { return v.Ref == ref })
		if i < 0 {
			merged = append(merged, item)
		} else {
			changed = true
			if item.LastImageMS > merged[i].LastImageMS {
				merged[i] = item
			}
		}
	}
	if changed {
		return localdata.Write(s.Path, merged)
	}
	return nil
}

func (s *ChallengePreferences) migrate(bots []rayleabot.Bot, warn func(string)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	changed := false
	merged := []ChallengePreference{}
	for _, item := range items {
		owner := item.Owner.recover(bots)
		changed = changed || item.Owner != owner
		item.Owner = owner
		if owner.BotID == "" {
			warn(filepath.Base(s.Path))
			merged = append(merged, item)
			continue
		}
		i := slices.IndexFunc(merged, func(v ChallengePreference) bool { return v.Owner == owner })
		if i < 0 {
			merged = append(merged, item)
		} else {
			changed = true
			// No timestamps exist here: later explicit values win, while
			// missing values preserve the other instance's preferences.
			if item.AbyssLevel != nil {
				merged[i].AbyssLevel = item.AbyssLevel
			}
			if item.DeadlyStars != nil {
				merged[i].DeadlyStars = item.DeadlyStars
			}
			if item.RemindTime != "" {
				merged[i].RemindTime = item.RemindTime
			}
		}
	}
	if changed {
		return localdata.Write(s.Path, merged)
	}
	return nil
}
