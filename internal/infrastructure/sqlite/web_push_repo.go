// Package sqlite implements browser Web Push persistence.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/notification"
	"github.com/google/uuid"
)

// WebPushRepository stores browser subscriptions separately from native APNs/FCM tokens.
type WebPushRepository struct {
	db *sql.DB
}

func NewWebPushRepository(db *sql.DB) *WebPushRepository {
	return &WebPushRepository{db: db}
}

func (r *WebPushRepository) RegisterWebPushSubscription(ctx context.Context, s *notification.WebPushSubscription) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	if s.Locale == "" {
		s.Locale = "it"
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("web push: begin subscription transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO web_push_subscriptions (
			id, user_id, endpoint, p256dh, auth_secret, locale,
			expiration_time, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET
			p256dh = excluded.p256dh,
			auth_secret = excluded.auth_secret,
			locale = excluded.locale,
			expiration_time = excluded.expiration_time,
			updated_at = excluded.updated_at
		WHERE web_push_subscriptions.user_id = excluded.user_id
	`,
		s.ID, s.UserID, s.Endpoint, s.P256DH, s.Auth, s.Locale,
		s.ExpirationTime, s.CreatedAt.Format(time.RFC3339), s.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("web push: register subscription: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("web push: inspect subscription result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("web push: endpoint is already owned by another user")
	}

	// Preferences are created only after explicit browser subscription consent.
	// Existing choices are preserved on subsequent device registrations.
	_, err = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO web_push_preferences (user_id) VALUES (?)
	`, s.UserID)
	if err != nil {
		return fmt.Errorf("web push: create default preferences: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("web push: commit subscription: %w", err)
	}
	return nil
}

func (r *WebPushRepository) DeleteWebPushSubscription(ctx context.Context, userID, endpoint string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM web_push_subscriptions WHERE user_id = ? AND endpoint = ?`,
		userID, endpoint,
	)
	if err != nil {
		return fmt.Errorf("web push: delete subscription: %w", err)
	}
	return nil
}

func (r *WebPushRepository) ListWebPushSubscriptions(ctx context.Context, userID string) ([]notification.WebPushSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, endpoint, p256dh, auth_secret, locale,
			expiration_time, created_at, updated_at
		FROM web_push_subscriptions
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("web push: list user subscriptions: %w", err)
	}
	defer rows.Close()
	return scanWebPushSubscriptions(rows)
}

func (r *WebPushRepository) GetWebPushPreferences(ctx context.Context, userID string) (notification.WebPushPreferences, error) {
	p := notification.DefaultWebPushPreferences()
	var alarm, emergency, offline, automation, command, network, camera, doorbell, pantry, finance int
	var tasks, medicineExpiry, medicineLowStock, foodExpiry, foodLowStock int
	var medicineDays, foodDays int
	err := r.db.QueryRowContext(ctx, `
		SELECT alarm_alerts, emergency_alerts, device_offline_alerts,
			automation_failure_alerts, command_failure_alerts, network_alerts,
			camera_alerts, doorbell_alerts, pantry_alerts, finance_alerts,
			task_alerts, medicine_expiry_alerts, medicine_low_stock_alerts,
			medicine_expiry_days, food_expiry_alerts, food_low_stock_alerts,
			food_expiry_days
		FROM web_push_preferences WHERE user_id = ?
	`, userID).Scan(
		&alarm, &emergency, &offline, &automation, &command, &network, &camera, &doorbell, &pantry, &finance,
		&tasks, &medicineExpiry, &medicineLowStock, &medicineDays, &foodExpiry, &foodLowStock, &foodDays,
	)
	if err == sql.ErrNoRows {
		return p, nil
	}
	if err != nil {
		return p, fmt.Errorf("web push: get preferences: %w", err)
	}
	p.AlarmAlerts = alarm == 1
	p.EmergencyAlerts = emergency == 1
	p.DeviceOfflineAlerts = offline == 1
	p.AutomationFailureAlerts = automation == 1
	p.CommandFailureAlerts = command == 1
	p.NetworkAlerts = network == 1
	p.CameraAlerts = camera == 1
	p.DoorbellAlerts = doorbell == 1
	p.PantryAlerts = pantry == 1
	p.FinanceAlerts = finance == 1
	p.TaskAlerts = tasks == 1
	p.MedicineExpiryAlerts = medicineExpiry == 1
	p.MedicineLowStockAlerts = medicineLowStock == 1
	p.MedicineExpiryDays = medicineDays
	p.FoodExpiryAlerts = foodExpiry == 1
	p.FoodLowStockAlerts = foodLowStock == 1
	p.FoodExpiryDays = foodDays
	return p, nil
}

func (r *WebPushRepository) SaveWebPushPreferences(ctx context.Context, userID string, p notification.WebPushPreferences) error {
	if userID == "" {
		return fmt.Errorf("web push: user id is required")
	}
	if p.MedicineExpiryDays < 0 || p.MedicineExpiryDays > 365 {
		return fmt.Errorf("web push: medicine expiry window must be between 0 and 365 days")
	}
	if p.FoodExpiryDays < 0 || p.FoodExpiryDays > 365 {
		return fmt.Errorf("web push: food expiry window must be between 0 and 365 days")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO web_push_preferences (
			user_id, alarm_alerts, emergency_alerts, device_offline_alerts,
			automation_failure_alerts, command_failure_alerts, network_alerts,
			camera_alerts, doorbell_alerts, pantry_alerts, finance_alerts,
			task_alerts, medicine_expiry_alerts, medicine_low_stock_alerts,
			medicine_expiry_days, food_expiry_alerts, food_low_stock_alerts,
			food_expiry_days, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			alarm_alerts = excluded.alarm_alerts,
			emergency_alerts = excluded.emergency_alerts,
			device_offline_alerts = excluded.device_offline_alerts,
			automation_failure_alerts = excluded.automation_failure_alerts,
			command_failure_alerts = excluded.command_failure_alerts,
			network_alerts = excluded.network_alerts,
			camera_alerts = excluded.camera_alerts,
			doorbell_alerts = excluded.doorbell_alerts,
			pantry_alerts = excluded.pantry_alerts,
			finance_alerts = excluded.finance_alerts,
			task_alerts = excluded.task_alerts,
			medicine_expiry_alerts = excluded.medicine_expiry_alerts,
			medicine_low_stock_alerts = excluded.medicine_low_stock_alerts,
			medicine_expiry_days = excluded.medicine_expiry_days,
			food_expiry_alerts = excluded.food_expiry_alerts,
			food_low_stock_alerts = excluded.food_low_stock_alerts,
			food_expiry_days = excluded.food_expiry_days,
			updated_at = excluded.updated_at
	`,
		userID, boolInt(p.AlarmAlerts), boolInt(p.EmergencyAlerts), boolInt(p.DeviceOfflineAlerts),
		boolInt(p.AutomationFailureAlerts), boolInt(p.CommandFailureAlerts), boolInt(p.NetworkAlerts),
		boolInt(p.CameraAlerts), boolInt(p.DoorbellAlerts), boolInt(p.PantryAlerts), boolInt(p.FinanceAlerts),
		boolInt(p.TaskAlerts), boolInt(p.MedicineExpiryAlerts), boolInt(p.MedicineLowStockAlerts),
		p.MedicineExpiryDays, boolInt(p.FoodExpiryAlerts), boolInt(p.FoodLowStockAlerts),
		p.FoodExpiryDays, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("web push: save preferences: %w", err)
	}
	return nil
}

func (r *WebPushRepository) ListWebPushRecipients(ctx context.Context, category string) ([]notification.WebPushSubscription, error) {
	// The column comes only from this fixed allow-list; never interpolate caller input.
	columnByCategory := map[string]string{
		"alarm_alerts": "alarm_alerts",
		"emergency_alerts": "emergency_alerts",
		"device_offline_alerts": "device_offline_alerts",
		"automation_failure_alerts": "automation_failure_alerts",
		"command_failure_alerts": "command_failure_alerts",
		"network_alerts": "network_alerts",
		"camera_alerts": "camera_alerts",
		"doorbell_alerts": "doorbell_alerts",
		"pantry_alerts": "pantry_alerts",
		"finance_alerts": "finance_alerts",
		"task_alerts": "task_alerts",
		"medicine_expiry_alerts": "medicine_expiry_alerts",
		"medicine_low_stock_alerts": "medicine_low_stock_alerts",
		"food_expiry_alerts": "food_expiry_alerts",
		"food_low_stock_alerts": "food_low_stock_alerts",
	}
	column, ok := columnByCategory[category]
	if !ok {
		return nil, fmt.Errorf("web push: unsupported event category")
	}
	query := `
		SELECT s.id, s.user_id, s.endpoint, s.p256dh, s.auth_secret, s.locale,
			s.expiration_time, s.created_at, s.updated_at
		FROM web_push_subscriptions s
		INNER JOIN web_push_preferences p ON p.user_id = s.user_id
		INNER JOIN users u ON u.id = s.user_id AND u.enabled = 1
		WHERE p.` + column + ` = 1
		ORDER BY s.updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("web push: list recipients: %w", err)
	}
	defer rows.Close()
	return scanWebPushSubscriptions(rows)
}

// ClaimWebPushDelivery atomically reserves one logical notification for a user.
func (r *WebPushRepository) ClaimWebPushDelivery(ctx context.Context, userID, dedupeKey string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO web_push_delivery_log (user_id, dedupe_key, sent_at)
		VALUES (?, ?, ?)
	`, userID, dedupeKey, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return false, fmt.Errorf("web push: reserve delivery: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("web push: inspect delivery reservation: %w", err)
	}
	return rows == 1, nil
}

// ReleaseWebPushDelivery permits retry when no subscription accepted the message.
func (r *WebPushRepository) ReleaseWebPushDelivery(ctx context.Context, userID, dedupeKey string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM web_push_delivery_log WHERE user_id = ? AND dedupe_key = ?`, userID, dedupeKey)
	if err != nil {
		return fmt.Errorf("web push: release delivery reservation: %w", err)
	}
	return nil
}

func scanWebPushSubscriptions(rows *sql.Rows) ([]notification.WebPushSubscription, error) {
	result := make([]notification.WebPushSubscription, 0)
	for rows.Next() {
		var (
			s             notification.WebPushSubscription
			expiration    sql.NullInt64
			createdAtText string
			updatedAtText string
		)
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Endpoint, &s.P256DH, &s.Auth, &s.Locale,
			&expiration, &createdAtText, &updatedAtText,
		); err != nil {
			return nil, fmt.Errorf("web push: scan subscription: %w", err)
		}
		if expiration.Valid {
			value := expiration.Int64
			s.ExpirationTime = &value
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAtText)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtText)
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("web push: iterate subscriptions: %w", err)
	}
	return result, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
