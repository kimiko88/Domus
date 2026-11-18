// Package notification includes server-side household expiry and stock reminder scheduling.
package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	domainchore "github.com/domus-hub/domus/internal/domain/chore"
	domainmedicine "github.com/domus-hub/domus/internal/domain/medicine"
	domainnotification "github.com/domus-hub/domus/internal/domain/notification"
	domainpantry "github.com/domus-hub/domus/internal/domain/pantry"
)

type ReminderScheduler struct {
	push       *WebPushService
	chores     domainchore.Repository
	medicines  domainmedicine.Repository
	pantry     domainpantry.PantryRepository
	interval   time.Duration
	logger     *slog.Logger
	mu         sync.Mutex
	cancel     context.CancelFunc
	done       chan struct{}
}

func NewReminderScheduler(push *WebPushService, chores domainchore.Repository, medicines domainmedicine.Repository, pantry domainpantry.PantryRepository, interval time.Duration, logger *slog.Logger) *ReminderScheduler {
	if interval <= 0 { interval = time.Hour }
	if logger == nil { logger = slog.Default() }
	return &ReminderScheduler{push: push, chores: chores, medicines: medicines, pantry: pantry, interval: interval, logger: logger}
}

func (s *ReminderScheduler) Start(parent context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil { return errors.New("web push reminder scheduler is already running") }
	if s.push == nil || !s.push.Enabled() { return nil }
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.loop(ctx, s.done)
	return nil
}

func (s *ReminderScheduler) Stop(ctx context.Context) {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	if cancel == nil { s.mu.Unlock(); return }
	s.cancel = nil
	s.done = nil
	cancel()
	s.mu.Unlock()
	select {
	case <-done:
	case <-ctx.Done():
		s.logger.WarnContext(ctx, "timed out waiting for Web Push reminder scheduler shutdown")
	}
}

func (s *ReminderScheduler) loop(ctx context.Context, done chan struct{}) {
	defer close(done)
	run := func() {
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if err := s.Check(checkCtx); err != nil {
			s.logger.WarnContext(checkCtx, "scheduled Web Push reminder check failed", "error", err)
		}
	}
	run()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

// Check evaluates household inventory against each subscribed user's preferences.
func (s *ReminderScheduler) Check(ctx context.Context) error {
	if s.push == nil || !s.push.Enabled() { return nil }
	var failures []error

	if s.chores != nil {
		overdue, listErr := s.chores.List(ctx, domainchore.Filter{Status: domainchore.StatusOverdue})
		if listErr != nil {
			failures = append(failures, fmt.Errorf("list overdue chores: %w", listErr))
		} else {
			recipients, recipientErr := s.push.Recipients(ctx, "task_alerts")
			if recipientErr != nil { failures = append(failures, recipientErr) }
			for _, task := range overdue {
				for _, recipient := range recipients {
					if task.AssignedUserID != "" && task.AssignedUserID != recipient.UserID { continue }
					key := "task:due:" + task.ID
					data := map[string]any{"kind":"task_due","name":task.Title,"task_id":task.ID,"url":"/chores"}
					if err := s.push.SendOnce(ctx, recipient.UserID, key, "Promemoria faccenda", task.Title, "task_alerts", data); err != nil {
						failures = append(failures, fmt.Errorf("task %s for user %s: %w", task.ID, recipient.UserID, err))
					}
				}
			}
		}
	}

	medicineExpiryRecipients, err := s.push.Recipients(ctx, "medicine_expiry_alerts")
	if err != nil { failures = append(failures, err) }
	medicineStockRecipients, err := s.push.Recipients(ctx, "medicine_low_stock_alerts")
	if err != nil { failures = append(failures, err) }
	medicineUsers := uniqueRecipientIDs(medicineExpiryRecipients, medicineStockRecipients)
	if len(medicineUsers) > 0 && s.medicines != nil {
		items, listErr := s.medicines.List(ctx, domainmedicine.Filter{})
		if listErr != nil {
			failures = append(failures, fmt.Errorf("list medicines: %w", listErr))
		} else {
			for _, userID := range medicineUsers {
				prefs, prefErr := s.push.Preferences(ctx, userID)
				if prefErr != nil { failures = append(failures, prefErr); continue }
				for _, item := range items {
					expiration := item.EffectiveExpirationDate
					if expiration == "" { expiration = item.CalculateEffectiveExpiration() }
					if prefs.MedicineExpiryAlerts && prefs.MedicineExpiryDays > 0 {
						if days, ok := daysUntil(expiration, time.Now()); ok && days <= prefs.MedicineExpiryDays {
							key := "medicine:expiry:" + item.ID + ":" + expiration
							data := map[string]any{"kind":"medicine_expiry","name":item.Name,"days_remaining":days,"medicine_id":item.ID,"url":"/medicines"}
							if err := s.push.SendOnce(ctx, userID, key, "Promemoria scadenza farmaco", item.Name, "medicine_expiry_alerts", data); err != nil { failures = append(failures, err) }
						}
					}
					stockKey := "medicine:low-stock:" + item.ID
					if prefs.MedicineLowStockAlerts && item.MinThreshold > 0 && item.Quantity <= item.MinThreshold {
						data := map[string]any{"kind":"medicine_low_stock","name":item.Name,"medicine_id":item.ID,"quantity":item.Quantity,"unit":string(item.Unit),"url":"/medicines"}
						if err := s.push.SendOnce(ctx, userID, stockKey, "Scorta farmaco bassa", item.Name, "medicine_low_stock_alerts", data); err != nil { failures = append(failures, err) }
					} else if item.MinThreshold <= 0 || item.Quantity > item.MinThreshold {
						if err := s.push.repo.ReleaseWebPushDelivery(ctx, userID, stockKey); err != nil { failures = append(failures, err) }
					}
				}
			}
		}
	}

	foodExpiryRecipients, err := s.push.Recipients(ctx, "food_expiry_alerts")
	if err != nil { failures = append(failures, err) }
	foodStockRecipients, err := s.push.Recipients(ctx, "food_low_stock_alerts")
	if err != nil { failures = append(failures, err) }
	foodUsers := uniqueRecipientIDs(foodExpiryRecipients, foodStockRecipients)
	if len(foodUsers) > 0 && s.pantry != nil {
		items, listErr := s.pantry.ListPantryItems(ctx, "", "")
		if listErr != nil {
			failures = append(failures, fmt.Errorf("list pantry items: %w", listErr))
		} else {
			for _, userID := range foodUsers {
				prefs, prefErr := s.push.Preferences(ctx, userID)
				if prefErr != nil { failures = append(failures, prefErr); continue }
				for _, item := range items {
					expiration := ""
					if item.ExpirationDate != nil { expiration = strings.TrimSpace(*item.ExpirationDate) }
					if prefs.FoodExpiryAlerts && prefs.FoodExpiryDays > 0 && expiration != "" {
						if days, ok := daysUntil(expiration, time.Now()); ok && days <= prefs.FoodExpiryDays {
							key := "food:expiry:" + item.ID + ":" + expiration
							data := map[string]any{"kind":"food_expiry","name":item.Name,"days_remaining":days,"pantry_item_id":item.ID,"url":"/pantry"}
							if err := s.push.SendOnce(ctx, userID, key, "Promemoria scadenza alimento", item.Name, "food_expiry_alerts", data); err != nil { failures = append(failures, err) }
						}
					}
					stockKey := "food:low-stock:" + item.ID
					if prefs.FoodLowStockAlerts && item.MinThreshold > 0 && item.Quantity <= item.MinThreshold {
						data := map[string]any{"kind":"food_low_stock","name":item.Name,"pantry_item_id":item.ID,"quantity":item.Quantity,"unit":string(item.Unit),"url":"/pantry"}
						if err := s.push.SendOnce(ctx, userID, stockKey, "Scorta alimento bassa", item.Name, "food_low_stock_alerts", data); err != nil { failures = append(failures, err) }
					} else if item.MinThreshold <= 0 || item.Quantity > item.MinThreshold {
						if err := s.push.repo.ReleaseWebPushDelivery(ctx, userID, stockKey); err != nil { failures = append(failures, err) }
					}
				}
			}
		}
	}
	return errors.Join(failures...)
}

func uniqueRecipientIDs(groups ...[]domainnotification.WebPushSubscription) []string {
	seen := make(map[string]struct{})
	users := make([]string, 0)
	for _, group := range groups {
		for _, subscription := range group {
			if subscription.UserID == "" { continue }
			if _, ok := seen[subscription.UserID]; ok { continue }
			seen[subscription.UserID] = struct{}{}
			users = append(users, subscription.UserID)
		}
	}
	return users
}

func daysUntil(value string, now time.Time) (int, bool) {
	value = strings.TrimSpace(value)
	if len(value) > 10 { value = value[:10] }
	date, err := time.Parse("2006-01-02", value)
	if err != nil { return 0, false }
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	return int(date.Sub(today).Hours() / 24), true
}
