// Package automation provides the runtime engine for executing automation rules.
package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

type activeRun struct {
	cancel context.CancelFunc
}

// Engine evaluates incoming events against registered automation rules and executes actions.
type Engine struct {
	autoRepo   automation.Repository
	entityRepo entity.Repository
	bus        events.Bus
	evaluator  *Evaluator
	mu               sync.Mutex
	unsubs           []func()
	running          bool
	schedulerCancel  context.CancelFunc
	schedulerDone    chan struct{}
	scheduleMu       sync.Mutex
	scheduledDates   map[string]string
	activeRuns       sync.Map // map[string]*activeRun
	queuedRuns       sync.Map // map[string]chan struct{}

}

// NewEngine creates an automation Engine instance.
func NewEngine(
	autoRepo automation.Repository,
	entityRepo entity.Repository,
	bus events.Bus,
) *Engine {
	return &Engine{
		autoRepo:   autoRepo,
		entityRepo: entityRepo,
		bus:        bus,
		evaluator:      NewEvaluator(),
		scheduledDates: make(map[string]string),
	}
}

// Start registers subscriptions on the event bus to listen for triggers.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return nil
	}

	u := e.bus.SubscribeAll(e.handleEvent)
	e.unsubs = append(e.unsubs, u)

	schedulerCtx, cancel := context.WithCancel(ctx)
	e.schedulerCancel = cancel
	e.schedulerDone = make(chan struct{})
	done := e.schedulerDone
	go func() {
		defer close(done)
		e.runTimeTriggerScheduler(schedulerCtx)
	}()

	e.running = true
	slog.Info("automation engine started")
	return nil
}

// Stop cleanly unregisters all event subscriptions.
func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}

	for _, u := range e.unsubs {
		u()
	}
	e.unsubs = nil
	cancel := e.schedulerCancel
	done := e.schedulerDone
	e.schedulerCancel = nil
	e.schedulerDone = nil
	e.running = false
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	slog.Info("automation engine stopped")
	return nil
}

// runTimeTriggerScheduler evaluates daily wall-clock triggers independently of
// the event bus. A one-second tick supports both HH:MM and HH:MM:SS schedules.
func (e *Engine) runTimeTriggerScheduler(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	e.processTimeTriggers(ctx, time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			e.processTimeTriggers(ctx, now)
		}
	}
}

// processTimeTriggers executes each configured time trigger at most once per
// local calendar day, even though the scheduler polls every second.
func (e *Engine) processTimeTriggers(ctx context.Context, now time.Time) {
	enabled := true
	rules, err := e.autoRepo.List(ctx, automation.ListFilter{Enabled: &enabled})
	if err != nil {
		slog.Warn("automation: cannot load scheduled rules", "error", err)
		return
	}

	localNow := now.In(time.Local)
	dateKey := localNow.Format("2006-01-02")
	for _, rule := range rules {
		for index, trigger := range rule.Triggers {
			if !e.evaluator.EvaluateTimeTrigger(trigger, localNow) {
				continue
			}
			triggerID := trigger.ID
			if triggerID == "" {
				triggerID = fmt.Sprintf("time-%d", index)
			}
			scheduleKey := rule.ID + ":" + triggerID

			if rule.CooldownSeconds > 0 && rule.LastRunAt != nil &&
				now.Sub(*rule.LastRunAt) < time.Duration(rule.CooldownSeconds)*time.Second {
				continue
			}

			e.scheduleMu.Lock()
			alreadyFired := e.scheduledDates[scheduleKey] == dateKey
			if !alreadyFired {
				e.scheduledDates[scheduleKey] = dateKey
			}
			e.scheduleMu.Unlock()
			if alreadyFired {
				continue
			}

			triggerContextBytes, _ := json.Marshal(map[string]any{
				"trigger":      trigger,
				"scheduled_at": localNow.Format(time.RFC3339),
			})
			go e.launchRuleExecution(rule, triggerID, string(triggerContextBytes))
		}
	}
}

func (e *Engine) handleEvent(ctx context.Context, evt events.Event) error {
	// Skip automation internal event loops to prevent infinite cycles
	if evt.Type == events.AutomationTriggered ||
		evt.Type == events.AutomationRunCompleted ||
		evt.Type == events.AutomationRunFailed {
		return nil
	}

	enabled := true
	rules, err := e.autoRepo.List(ctx, automation.ListFilter{Enabled: &enabled})
	if err != nil {
		return err
	}

	for _, rule := range rules {
		for _, trig := range rule.Triggers {
			if e.evaluator.EvaluateTrigger(trig, evt) {
				// Check cooldown
				if rule.CooldownSeconds > 0 && rule.LastRunAt != nil {
					elapsed := time.Since(*rule.LastRunAt)
					if elapsed < time.Duration(rule.CooldownSeconds)*time.Second {
						slog.Debug("automation: cooldown active, skipping", "automation_id", rule.ID)
						break
					}
				}

				// Execute asynchronously in background
				triggerContextBytes, _ := json.Marshal(map[string]any{
					"trigger": trig,
					"event":   evt,
				})
				go e.launchRuleExecution(rule, trig.ID, string(triggerContextBytes))
				break
			}
		}
	}
	return nil
}

// TriggerAutomation manually invokes an automation rule by ID.
func (e *Engine) TriggerAutomation(ctx context.Context, id string) error {
	rule, err := e.autoRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	triggerContext := `{"manual":true}`
	go e.launchRuleExecution(rule, "", triggerContext)
	return nil
}

func (e *Engine) launchRuleExecution(rule *automation.Automation, triggerID string, triggerContext string) {
	mode := rule.ExecutionMode
	if mode == "" {
		mode = "single"
	}

	runCtx, cancel := context.WithCancel(context.Background())
	currentRun := &activeRun{cancel: cancel}
	var releaseQueue func()

	switch mode {
	case "single":
		if _, loaded := e.activeRuns.LoadOrStore(rule.ID, currentRun); loaded {
			cancel()
			slog.Debug("automation: execution mode single, skipping run", "automation_id", rule.ID)
			return
		}
		defer e.activeRuns.CompareAndDelete(rule.ID, currentRun)

	case "restart":
		previous, loaded := e.activeRuns.Swap(rule.ID, currentRun)
		if loaded {
			if prevRun, ok := previous.(*activeRun); ok && prevRun.cancel != nil {
				prevRun.cancel()
			}
		}
		defer e.activeRuns.CompareAndDelete(rule.ID, currentRun)

	case "queued":
		candidate := make(chan struct{}, 1)
		candidate <- struct{}{}
		stored, _ := e.queuedRuns.LoadOrStore(rule.ID, candidate)
		semaphore := stored.(chan struct{})
		<-semaphore
		releaseQueue = func() { semaphore <- struct{}{} }
		defer releaseQueue()

	case "parallel":
		// Each invocation runs independently; no per-automation lock is held.

	default:
		cancel()
		slog.Warn("automation: unknown execution mode; skipping run", "automation_id", rule.ID, "execution_mode", mode)
		return
	}

	defer cancel()
	e.executeRule(runCtx, rule, triggerID, triggerContext)
}

func (e *Engine) executeRule(ctx context.Context, rule *automation.Automation, triggerID string, triggerContext string) {
	now := time.Now().UTC()

	// 1. Create Run Record
	run := &automation.AutomationRun{
		ID:             uuid.NewString(),
		AutomationID:   rule.ID,
		StartedAt:      now,
		Status:         automation.RunStatusRunning,
		TriggerContext: triggerContext,
	}
	_ = e.autoRepo.CreateRun(ctx, run)
	_ = e.autoRepo.UpdateLastRun(ctx, rule.ID, automation.RunStatusRunning)

	// 2. Evaluate Conditions
	for _, cond := range rule.Conditions {
		ok, err := e.evaluator.EvaluateConditionWithTrigger(ctx, cond, e.entityRepo, triggerID)
		if err != nil || !ok {
			logMsg := "Condition failed"
			if err != nil {
				logMsg = fmt.Sprintf("Condition evaluation error: %v", err)
			}
			run.FinishedAt = &now
			run.Status = automation.RunStatusSkipped
			run.Log = logMsg
			_ = e.autoRepo.UpdateRun(ctx, run)
			_ = e.autoRepo.UpdateLastRun(ctx, rule.ID, automation.RunStatusSkipped)
			return
		}
	}

	// 3. Emit AutomationTriggered
	_ = e.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.AutomationTriggered,
		OccurredAt: now,
		Source:     "automation_engine",
		Payload: map[string]any{
			"automation_id": rule.ID,
			"name":          rule.Name,
			"run_id":        run.ID,
			"trigger_id":    triggerID,
		},
	})

	// 4. Execute Actions
	var actionLogs []string
	for _, act := range rule.Actions {
		if err := ctx.Err(); err != nil {
			e.failRun(ctx, rule, run, "Execution cancelled by restart")
			return
		}
		if err := e.executeAction(ctx, act, rule, run, triggerID, &actionLogs); err != nil {
			e.failRun(ctx, rule, run, fmt.Sprintf("Action error: %v", err))
			return
		}
	}

	// 5. Mark Run Complete
	finished := time.Now().UTC()
	bLogs, _ := json.Marshal(actionLogs)
	run.Complete(finished, string(bLogs))
	_ = e.autoRepo.UpdateRun(ctx, run)
	_ = e.autoRepo.UpdateLastRun(ctx, rule.ID, automation.RunStatusCompleted)

	_ = e.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.AutomationRunCompleted,
		OccurredAt: finished,
		Source:     "automation_engine",
		Payload: map[string]any{
			"automation_id": rule.ID,
			"run_id":        run.ID,
			"duration_ms":   run.DurationMs,
			"status":        "completed",
		},
	})
}

func (e *Engine) executeAction(
	ctx context.Context,
	act automation.Action,
	rule *automation.Automation,
	run *automation.AutomationRun,
	triggerID string,
	actionLogs *[]string,
) error {
	switch act.Type {
	case "choose":
		branchMatched := false
		for idx, branch := range act.Choose {
			allPassed := true
			for _, cond := range branch.Conditions {
				passed, err := e.evaluator.EvaluateConditionWithTrigger(ctx, cond, e.entityRepo, triggerID)
				if err != nil || !passed {
					allPassed = false
					break
				}
			}
			if allPassed {
				branchMatched = true
				*actionLogs = append(*actionLogs, fmt.Sprintf("Choose branch %d matched", idx))
				for _, subAct := range branch.Sequence {
					if err := ctx.Err(); err != nil {
						return err
					}
					if err := e.executeAction(ctx, subAct, rule, run, triggerID, actionLogs); err != nil {
						return err
					}
				}
				break
			}
		}

		if !branchMatched && len(act.Default) > 0 {
			*actionLogs = append(*actionLogs, "Choose default executed")
			for _, subAct := range act.Default {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := e.executeAction(ctx, subAct, rule, run, triggerID, actionLogs); err != nil {
					return err
				}
			}
		}

	case "if", "if_then_else":
		allPassed := true
		for _, cond := range act.If {
			passed, err := e.evaluator.EvaluateConditionWithTrigger(ctx, cond, e.entityRepo, triggerID)
			if err != nil || !passed {
				allPassed = false
				break
			}
		}
		if allPassed {
			*actionLogs = append(*actionLogs, "If branch passed: executing Then")
			for _, subAct := range act.Then {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := e.executeAction(ctx, subAct, rule, run, triggerID, actionLogs); err != nil {
					return err
				}
			}
		} else if len(act.Else) > 0 {
			*actionLogs = append(*actionLogs, "If branch failed: executing Else")
			for _, subAct := range act.Else {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := e.executeAction(ctx, subAct, rule, run, triggerID, actionLogs); err != nil {
					return err
				}
			}
		}

	case "repeat":
		if act.Repeat == nil {
			return nil
		}
		cfg := act.Repeat
		if cfg.Count > 0 {
			for i := 0; i < cfg.Count; i++ {
				if err := ctx.Err(); err != nil {
					return err
				}
				*actionLogs = append(*actionLogs, fmt.Sprintf("Repeat count %d/%d", i+1, cfg.Count))
				for _, subAct := range cfg.Sequence {
					if err := e.executeAction(ctx, subAct, rule, run, triggerID, actionLogs); err != nil {
						return err
					}
				}
			}
		} else if len(cfg.While) > 0 {
			maxIterations := 1000
			iter := 0
			for iter < maxIterations {
				if err := ctx.Err(); err != nil {
					return err
				}
				whilePassed := true
				for _, cond := range cfg.While {
					passed, err := e.evaluator.EvaluateConditionWithTrigger(ctx, cond, e.entityRepo, triggerID)
					if err != nil || !passed {
						whilePassed = false
						break
					}
				}
				if !whilePassed {
					break
				}
				iter++
				*actionLogs = append(*actionLogs, fmt.Sprintf("Repeat while iteration %d", iter))
				for _, subAct := range cfg.Sequence {
					if err := e.executeAction(ctx, subAct, rule, run, triggerID, actionLogs); err != nil {
						return err
					}
				}
			}
		} else if len(cfg.Until) > 0 {
			maxIterations := 1000
			iter := 0
			for iter < maxIterations {
				if err := ctx.Err(); err != nil {
					return err
				}
				iter++
				*actionLogs = append(*actionLogs, fmt.Sprintf("Repeat until iteration %d", iter))
				for _, subAct := range cfg.Sequence {
					if err := e.executeAction(ctx, subAct, rule, run, triggerID, actionLogs); err != nil {
						return err
					}
				}
				untilPassed := true
				for _, cond := range cfg.Until {
					passed, err := e.evaluator.EvaluateConditionWithTrigger(ctx, cond, e.entityRepo, triggerID)
					if err != nil || !passed {
						untilPassed = false
						break
					}
				}
				if untilPassed {
					break
				}
			}
		}

	case "parallel":
		if len(act.Parallel) == 0 {
			return nil
		}
		*actionLogs = append(*actionLogs, fmt.Sprintf("Executing %d parallel actions", len(act.Parallel)))
		var wg sync.WaitGroup
		var parallelErr error
		var errMu sync.Mutex

		for _, subAct := range act.Parallel {
			wg.Add(1)
			go func(a automation.Action) {
				defer wg.Done()
				var subLogs []string
				if err := e.executeAction(ctx, a, rule, run, triggerID, &subLogs); err != nil {
					errMu.Lock()
					if parallelErr == nil {
						parallelErr = err
					}
					errMu.Unlock()
				}
			}(subAct)
		}
		wg.Wait()
		if parallelErr != nil {
			return parallelErr
		}

	case "service":
		cmdEvt := events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: time.Now().UTC(),
			Source:     fmt.Sprintf("automation:%s", rule.ID),
			Payload: map[string]any{
				"entity_key": act.TargetEntityKey,
				"service":    act.Service,
				"params":     act.Params,
			},
		}
		if err := e.bus.Publish(ctx, cmdEvt); err != nil {
			return err
		}
		*actionLogs = append(*actionLogs, fmt.Sprintf("Service %s on %s", act.Service, act.TargetEntityKey))

	case "delay":
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(act.DelayMs) * time.Millisecond):
		}
		*actionLogs = append(*actionLogs, fmt.Sprintf("Delayed %dms", act.DelayMs))

	case "event":
		evt := events.Event{
			ID:         uuid.NewString(),
			Type:       events.EventType(act.EventType),
			OccurredAt: time.Now().UTC(),
			Source:     fmt.Sprintf("automation:%s", rule.ID),
			Payload:    act.EventPayload,
		}
		_ = e.bus.Publish(ctx, evt)
		*actionLogs = append(*actionLogs, fmt.Sprintf("Fired event %s", act.EventType))
	}

	return nil
}

func (e *Engine) failRun(ctx context.Context, rule *automation.Automation, run *automation.AutomationRun, errMsg string) {
	finished := time.Now().UTC()
	run.Fail(finished, errMsg)
	_ = e.autoRepo.UpdateRun(ctx, run)
	_ = e.autoRepo.UpdateLastRun(ctx, rule.ID, automation.RunStatusFailed)

	_ = e.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.AutomationRunFailed,
		OccurredAt: finished,
		Source:     "automation_engine",
		Payload: map[string]any{
			"automation_id": rule.ID,
			"run_id":        run.ID,
			"error":         errMsg,
		},
	})
}
