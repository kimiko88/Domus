// Package notification contains authenticated browser Web Push delivery.
package notification

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"net/http"
	"net/url"
	"strings"
	"time"

	domain "github.com/domus-hub/domus/internal/domain/notification"
	"github.com/google/uuid"
	"github.com/domus-hub/domus/internal/domain/events"
	webpush "github.com/SherClockHolmes/webpush-go"
)

type WebPushConfig struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

func WebPushConfigFromEnv() WebPushConfig {
	publicKey := strings.TrimSpace(envOr("DOMUS_VAPID_PUBLIC_KEY", "VAPID_PUBLIC_KEY"))
	privateKey := strings.TrimSpace(envOr("DOMUS_VAPID_PRIVATE_KEY", "VAPID_PRIVATE_KEY"))
	subject := strings.TrimSpace(envOr("DOMUS_VAPID_SUBJECT", "VAPID_SUBJECT"))
	return WebPushConfig{PublicKey: publicKey, PrivateKey: privateKey, Subject: subject}
}

func envOr(primary, fallback string) string {
	if value := strings.TrimSpace(getenv(primary)); value != "" {
		return value
	}
	return getenv(fallback)
}

// getenv is isolated to keep configuration loading easy to reason about.
var getenv = func(key string) string { return os.Getenv(key) }

type WebPushService struct {
	repo   domain.WebPushRepository
	config WebPushConfig
	client *http.Client
	logger *slog.Logger
}

func NewWebPushService(repo domain.WebPushRepository, cfg WebPushConfig, logger *slog.Logger) *WebPushService {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebPushService{
		repo: repo, config: cfg,
		client: &http.Client{Timeout: 15 * time.Second},
		logger: logger,
	}
}

func (s *WebPushService) Enabled() bool {
	if s == nil || s.repo == nil || s.config.PublicKey == "" || s.config.PrivateKey == "" {
		return false
	}
	if _, err := url.ParseRequestURI(s.config.Subject); err != nil {
		return false
	}
	return strings.HasPrefix(s.config.Subject, "mailto:") || strings.HasPrefix(s.config.Subject, "https://")
}

func (s *WebPushService) PublicKey() (string, bool) {
	if !s.Enabled() {
		return "", false
	}
	return s.config.PublicKey, true
}

func (s *WebPushService) RegisterSubscription(ctx context.Context, subscription *domain.WebPushSubscription) error {
	if !s.Enabled() {
		return errors.New("web push is not configured on this server")
	}
	if subscription == nil || subscription.UserID == "" || subscription.Endpoint == "" {
		return errors.New("subscription endpoint and authenticated user are required")
	}
	if err := validatePushEndpoint(subscription.Endpoint); err != nil {
		return err
	}
	if !validSubscriptionKey(subscription.P256DH, 65) {
		return errors.New("invalid Push API p256dh key")
	}
	if !validSubscriptionKey(subscription.Auth, 16) {
		return errors.New("invalid Push API auth key")
	}
	if subscription.Locale == "" {
		subscription.Locale = "it"
	}
	return s.repo.RegisterWebPushSubscription(ctx, subscription)
}

func (s *WebPushService) Unsubscribe(ctx context.Context, userID, endpoint string) error {
	if s == nil || s.repo == nil {
		return errors.New("web push storage is not available")
	}
	if userID == "" || endpoint == "" {
		return errors.New("user and subscription endpoint are required")
	}
	if err := validatePushEndpoint(endpoint); err != nil {
		return err
	}
	return s.repo.DeleteWebPushSubscription(ctx, userID, endpoint)
}

func (s *WebPushService) Preferences(ctx context.Context, userID string) (domain.WebPushPreferences, error) {
	if s == nil || s.repo == nil {
		return domain.WebPushPreferences{}, errors.New("web push storage is not available")
	}
	return s.repo.GetWebPushPreferences(ctx, userID)
}

func (s *WebPushService) SavePreferences(ctx context.Context, userID string, preferences domain.WebPushPreferences) error {
	if s == nil || s.repo == nil {
		return errors.New("web push storage is not available")
	}
	if userID == "" {
		return errors.New("authenticated user is required")
	}
	if preferences.MedicineExpiryDays < 0 || preferences.MedicineExpiryDays > 365 ||
		preferences.FoodExpiryDays < 0 || preferences.FoodExpiryDays > 365 {
		return errors.New("expiry notification windows must be between 0 and 365 days")
	}
	return s.repo.SaveWebPushPreferences(ctx, userID, preferences)
}

func (s *WebPushService) SendUser(ctx context.Context, userID, title, body, category string, data map[string]any) (int, error) {
	if !s.Enabled() {
		return 0, errors.New("web push is not configured on this server")
	}
	subscriptions, err := s.repo.ListWebPushSubscriptions(ctx, userID)
	if err != nil {
		return 0, err
	}

	sent := 0
	var firstErr error
	for _, subscription := range subscriptions {
		if err := ctx.Err(); err != nil {
			return sent, err
		}
		localizedTitle, localizedBody := localizedWebPushMessage(subscription.Locale, title, body, data)
		payload := map[string]any{
			"title": localizedTitle,
			"body": localizedBody,
			"category": category,
			"url": safeNotificationURL(data),
			"tag": pushTag(data),
			"data": data,
		}
		message, err := json.Marshal(payload)
		if err != nil {
			return sent, fmt.Errorf("web push: encode payload: %w", err)
		}
		webSubscription := &webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{P256dh: subscription.P256DH, Auth: subscription.Auth},
		}
		response, sendErr := webpush.SendNotificationWithContext(ctx, message, webSubscription, &webpush.Options{
			HTTPClient: s.client,
			Subscriber: s.config.Subject,
			VAPIDPublicKey: s.config.PublicKey,
			VAPIDPrivateKey: s.config.PrivateKey,
			TTL: 3600,
			Urgency: webpush.UrgencyHigh,
			Topic: pushTopic(data),
		})
		if sendErr != nil {
			s.logger.WarnContext(ctx, "web push delivery failed", "user_id", userID, "error", sendErr)
			if firstErr == nil { firstErr = sendErr }
			continue
		}
		if response == nil {
			if firstErr == nil { firstErr = errors.New("web push provider returned an empty response") }
			continue
		}
		_ = response.Body.Close()
		if response.StatusCode == http.StatusGone || response.StatusCode == http.StatusNotFound {
			_ = s.repo.DeleteWebPushSubscription(ctx, userID, subscription.Endpoint)
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			err := fmt.Errorf("web push provider returned HTTP %d", response.StatusCode)
			s.logger.WarnContext(ctx, "web push provider rejected notification", "user_id", userID, "status", response.StatusCode)
			if firstErr == nil { firstErr = err }
			continue
		}
		sent++
	}
	if sent == 0 && firstErr != nil {
		return 0, firstErr
	}
	return sent, nil
}


func (s *WebPushService) Recipients(ctx context.Context, category string) ([]domain.WebPushSubscription, error) {
	if !s.Enabled() { return nil, nil }
	return s.repo.ListWebPushRecipients(ctx, category)
}

func (s *WebPushService) SendOnce(ctx context.Context, userID, dedupeKey, title, body, category string, data map[string]any) error {
	if !s.Enabled() { return nil }
	claimed, err := s.repo.ClaimWebPushDelivery(ctx, userID, dedupeKey)
	if err != nil || !claimed { return err }
	if data == nil { data = make(map[string]any) }
	data["dedupe_key"] = dedupeKey
	sent, sendErr := s.SendUser(ctx, userID, title, body, category, data)
	if sendErr != nil || sent == 0 {
		if releaseErr := s.repo.ReleaseWebPushDelivery(ctx, userID, dedupeKey); sendErr == nil { sendErr = releaseErr }
	}
	return sendErr
}

// RegisterEventHandlers connects task due notifications to browser push delivery.
func (s *WebPushService) RegisterEventHandlers(bus events.Bus) func() {
	if bus == nil || !s.Enabled() { return func() {} }
	return bus.Subscribe(events.NotificationCreated, func(_ context.Context, event events.Event) error {
		payload, ok := event.Payload.(map[string]any)
		if !ok || payload["notification_kind"] != "task_due" { return nil }
		choreID, _ := payload["chore_id"].(string)
		taskName, _ := payload["task_name"].(string)
		targetUser, _ := payload["user_id"].(string)
		if choreID == "" { return nil }
		go s.sendTaskDueReminder(choreID, taskName, targetUser)
		return nil
	})
}

func (s *WebPushService) sendTaskDueReminder(choreID, taskName, targetUser string) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	recipients, err := s.Recipients(ctx, "task_alerts")
	if err != nil { s.logger.WarnContext(ctx, "could not load task Web Push recipients", "error", err); return }
	seen := make(map[string]struct{})
	for _, recipient := range recipients {
		if targetUser != "" && recipient.UserID != targetUser { continue }
		if _, exists := seen[recipient.UserID]; exists { continue }
		seen[recipient.UserID] = struct{}{}
		key := "task:due:" + choreID
		data := map[string]any{"kind":"task_due","name":taskName,"task_id":choreID,"url":"/chores"}
		if err := s.SendOnce(ctx, recipient.UserID, key, "Promemoria faccenda", "La faccenda è arrivata alla scadenza: "+taskName, "task_alerts", data); err != nil {
			s.logger.WarnContext(ctx, "task Web Push reminder failed", "user_id", recipient.UserID, "chore_id", choreID, "error", err)
		}
	}
}

func validatePushEndpoint(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("push endpoint must be an HTTPS provider URL")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return errors.New("push endpoint must use HTTPS port 443")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	allowed := host == "fcm.googleapis.com" ||
		host == "updates.push.services.mozilla.com" ||
		host == "push.services.mozilla.com" ||
		host == "wns.windows.com" ||
		strings.HasSuffix(host, ".push.apple.com") ||
		strings.HasSuffix(host, ".notify.windows.com")
	if !allowed || net.ParseIP(host) != nil {
		return errors.New("push endpoint host is not an approved browser push provider")
	}
	return nil
}

func validSubscriptionKey(value string, expectedLength int) bool {
	for _, encoding := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		decoded, err := encoding.DecodeString(value)
		if err == nil && len(decoded) == expectedLength {
			return true
		}
	}
	return false
}

func safeNotificationURL(data map[string]any) string {
	if data != nil {
		if target, ok := data["url"].(string); ok && strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "//") {
			return target
		}
	}
	return "/"
}

func pushTag(data map[string]any) string {
	if data != nil {
		if value, ok := data["dedupe_key"].(string); ok && value != "" {
			return "domus-" + uuid.NewSHA1(uuid.NameSpaceURL, []byte(value)).String()[:16]
		}
	}
	return "domus-" + uuid.NewString()[:16]
}

func pushTopic(data map[string]any) string {
	if data != nil {
		if value, ok := data["dedupe_key"].(string); ok && value != "" {
			return uuid.NewSHA1(uuid.NameSpaceURL, []byte(value)).String()[:32]
		}
	}
	return ""
}

func localizedWebPushMessage(locale, title, body string, data map[string]any) (string, string) {
	lang := strings.ToLower(strings.Split(locale, "-")[0])
	name, _ := data["name"].(string)
	days, _ := data["days_remaining"].(int)
	kind, _ := data["kind"].(string)
	switch kind {
	case "task_due":
		switch lang {
		case "en": return "Task overdue", "The task “" + name + "” is due."
		case "fr": return "Tâche en retard", "La tâche « " + name + " » est arrivée à échéance."
		case "es": return "Tarea vencida", "La tarea «" + name + "» ha vencido."
		case "de": return "Aufgabe überfällig", "Die Aufgabe „" + name + "“ ist fällig."
		case "ru": return "Срок задачи истёк", "Наступил срок выполнения задачи «" + name + "»."
		}
	case "medicine_expiry":
		switch lang {
		case "en": return "Medicine expiry reminder", fmt.Sprintf("%s expires in %d days.", name, days)
		case "fr": return "Rappel de péremption du médicament", fmt.Sprintf("%s expire dans %d jours.", name, days)
		case "es": return "Aviso de caducidad del medicamento", fmt.Sprintf("%s caduca en %d días.", name, days)
		case "de": return "Arzneimittel läuft ab", fmt.Sprintf("%s läuft in %d Tagen ab.", name, days)
		case "ru": return "Срок годности лекарства", fmt.Sprintf("Срок годности «%s» истекает через %d дн.", name, days)
		}
	case "food_expiry":
		switch lang {
		case "en": return "Food expiry reminder", fmt.Sprintf("%s expires in %d days.", name, days)
		case "fr": return "Rappel de péremption alimentaire", fmt.Sprintf("%s expire dans %d jours.", name, days)
		case "es": return "Aviso de caducidad de alimentos", fmt.Sprintf("%s caduca en %d días.", name, days)
		case "de": return "Lebensmittel laufen ab", fmt.Sprintf("%s läuft in %d Tagen ab.", name, days)
		case "ru": return "Срок годности продукта", fmt.Sprintf("Срок годности «%s» истекает через %d дн.", name, days)
		}
	case "medicine_low_stock":
		switch lang {
		case "en": return "Medicine stock is low", "The remaining amount of " + name + " has reached its minimum level."
		case "fr": return "Stock de médicament faible", "La quantité restante de " + name + " a atteint le seuil minimal."
		case "es": return "Pocas existencias de medicamento", "La cantidad restante de " + name + " ha alcanzado el mínimo."
		case "de": return "Arzneimittelbestand niedrig", "Der Restbestand von " + name + " hat den Mindestwert erreicht."
		case "ru": return "Заканчивается лекарство", "Остаток препарата «" + name + "» достиг минимального уровня."
		}
	case "food_low_stock":
		switch lang {
		case "en": return "Food stock is low", "The remaining amount of " + name + " has reached its minimum level."
		case "fr": return "Stock alimentaire faible", "La quantité restante de " + name + " a atteint le seuil minimal."
		case "es": return "Pocas existencias de alimentos", "La cantidad restante de " + name + " ha alcanzado el mínimo."
		case "de": return "Lebensmittelbestand niedrig", "Der Restbestand von " + name + " hat den Mindestwert erreicht."
		case "ru": return "Заканчиваются продукты", "Остаток продукта «" + name + "» достиг минимального уровня."
		}
	}
	return title, body
}
