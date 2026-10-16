package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/webhook"
	"github.com/google/uuid"
)

// OutboundDispatcher manages outbound webhook subscriptions and event delivery.
type OutboundDispatcher struct {
	repo       webhook.OutboundRepository
	bus        events.Bus
	logger     *slog.Logger
	httpClient *http.Client

	eventQueue chan events.Event
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	unsub      func()
}

// NewOutboundDispatcher creates a new dispatcher instance.
func NewOutboundDispatcher(repo webhook.OutboundRepository, bus events.Bus, logger *slog.Logger) *OutboundDispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &OutboundDispatcher{
		repo:   repo,
		bus:    bus,
		logger: logger.With(slog.String("component", "outbound_webhook_dispatcher")),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		eventQueue: make(chan events.Event, 512),
	}
}

// Start initiates worker goroutines and subscribes to the domain event bus.
func (d *OutboundDispatcher) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	workerCount := 4
	for i := 0; i < workerCount; i++ {
		d.wg.Add(1)
		go d.worker(ctx, i)
	}

	if d.bus != nil {
		d.unsub = d.bus.SubscribeAll(func(ctx context.Context, event events.Event) error {
			select {
			case d.eventQueue <- event:
			default:
				d.logger.Warn("outbound webhook event queue full, dropping event",
					slog.String("event_id", event.ID),
					slog.String("event_type", string(event.Type)),
				)
			}
			return nil
		})
	}

	d.logger.Info("outbound webhook dispatcher started", slog.Int("workers", workerCount))
}

// Stop cleanly terminates all dispatcher workers.
func (d *OutboundDispatcher) Stop() {
	if d.unsub != nil {
		d.unsub()
	}
	if d.cancel != nil {
		d.cancel()
	}
	d.wg.Wait()
	d.logger.Info("outbound webhook dispatcher stopped")
}

func (d *OutboundDispatcher) worker(ctx context.Context, id int) {
	defer d.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-d.eventQueue:
			if !ok {
				return
			}
			d.processEvent(ctx, event)
		}
	}
}

func (d *OutboundDispatcher) processEvent(ctx context.Context, event events.Event) {
	webhooks, err := d.repo.List(ctx)
	if err != nil {
		d.logger.Error("failed to list outbound webhooks", slog.String("error", err.Error()))
		return
	}

	for i := range webhooks {
		wh := webhooks[i]
		if !wh.MatchesEvent(string(event.Type)) {
			continue
		}

		go func(target webhook.OutboundWebhook, evt events.Event) {
			dispatchCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			err := d.dispatchPayload(dispatchCtx, &target, evt)
			if err != nil {
				d.logger.Warn("outbound webhook delivery failed",
					slog.String("webhook_id", target.ID),
					slog.String("target_url", target.TargetURL),
					slog.String("error", err.Error()),
				)
			}
		}(wh, event)
	}
}

func (d *OutboundDispatcher) dispatchPayload(ctx context.Context, wh *webhook.OutboundWebhook, event events.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.TargetURL, bytes.NewReader(payload))
	if err != nil {
		_ = d.repo.RecordDispatch(context.Background(), wh.ID, 0, true)
		return fmt.Errorf("create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Domus-Hub/1.0 (+https://domushub.local)")
	req.Header.Set("X-Domus-Event", string(event.Type))
	req.Header.Set("X-Domus-Delivery", uuid.NewString())

	if sig := wh.ComputeSignature(payload); sig != "" {
		req.Header.Set("X-Domus-Signature", sig)
		req.Header.Set("X-Hub-Signature-256", sig)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		_ = d.repo.RecordDispatch(context.Background(), wh.ID, 0, true)
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	failed := resp.StatusCode < 200 || resp.StatusCode >= 300
	_ = d.repo.RecordDispatch(context.Background(), wh.ID, resp.StatusCode, failed)

	if failed {
		return fmt.Errorf("target returned status code %d", resp.StatusCode)
	}
	return nil
}

// CreateOutbound registers a new outbound webhook.
func (d *OutboundDispatcher) CreateOutbound(ctx context.Context, name, targetURL, secretToken string, patterns []string) (*webhook.OutboundWebhook, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("webhook: name cannot be empty")
	}

	parsed, err := url.ParseRequestURI(targetURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, webhook.ErrInvalidTargetURL
	}

	if len(patterns) == 0 {
		patterns = []string{"*"}
	}

	wh := &webhook.OutboundWebhook{
		ID:          uuid.NewString(),
		Name:        name,
		TargetURL:   targetURL,
		SecretToken: strings.TrimSpace(secretToken),
		Events:      patterns,
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
	}

	if err := d.repo.Create(ctx, wh); err != nil {
		return nil, err
	}
	return wh, nil
}

// ListOutbound retrieves all configured outbound endpoints.
func (d *OutboundDispatcher) ListOutbound(ctx context.Context) ([]webhook.OutboundWebhook, error) {
	return d.repo.List(ctx)
}

// GetOutbound returns an outbound endpoint by ID.
func (d *OutboundDispatcher) GetOutbound(ctx context.Context, id string) (*webhook.OutboundWebhook, error) {
	return d.repo.GetByID(ctx, id)
}

// DeleteOutbound removes an outbound webhook endpoint.
func (d *OutboundDispatcher) DeleteOutbound(ctx context.Context, id string) error {
	return d.repo.Delete(ctx, id)
}

// TestOutbound sends an immediate ping event to verify connectivity and signature.
func (d *OutboundDispatcher) TestOutbound(ctx context.Context, id string) error {
	wh, err := d.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	testEvent := events.Event{
		ID:         uuid.NewString(),
		Type:       events.EventType("webhook.test"),
		OccurredAt: time.Now().UTC(),
		Source:     "system:outbound_test",
		Payload: map[string]any{
			"message": "Domus Outbound Webhook Connectivity Test",
			"webhook": wh.Name,
			"time":    time.Now().UTC().Format(time.RFC3339),
		},
	}

	return d.dispatchPayload(ctx, wh, testEvent)
}
