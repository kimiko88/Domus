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

// Engine evaluates incoming events against registered automation rules and executes actions.
type Engine struct {
	autoRepo   automation.Repository
	entityRepo entity.Repository
	bus        events.Bus
	evaluator  *Evaluator
	mu         sync.Mutex
	unsubs     []func()
	running    bool
	activeRuns sync.Map // map[string]context.CancelFunc
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
		evaluator:  NewEvaluator(),
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

	e.running = true
	slog.Info("automation engine started")
	return nil
}

// Stop cleanly unregisters all event subscriptions.
func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	for _, u := range e.unsubs {
		u()
	}
	e.unsubs = nil
	e.running = false
	slog.Info("automation engine stopped")
	return nil
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

	switch mode {
	case "single":
		if _, loaded := e.activeRuns.Load(rule.ID); loaded {
			slog.Debug("automation: execution mode single, skipping run", "automation_id", rule.ID)
			return
		}
	case "restart":
		if prevCancel, loaded := e.activeRuns.Load(rule.ID); loaded {
			slog.Debug("automation: execution mode restart, aborting previous run", "automation_id", rule.ID)
			if cancelFn, ok := prevCancel.(context.CancelFunc); ok {
				cancelFn()
			}
		}
	}

	runCtx, cancel := context.WithCancel(context.Background())
	e.activeRuns.Store(rule.ID, cancel)
	defer e.activeRuns.Delete(rule.ID)

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
