package kafka

import (
	"context"

	domain "github.com/aq2208/gorder-api/internal/entity"
	"github.com/aq2208/gorder-api/internal/usecase"
)

type OrderStatusChangedHandler struct {
	Repo  usecase.OrderRepo
	Cache usecase.OrderCache // optional
}

func NewOrderStatusChangedHandler(repo usecase.OrderRepo, cache usecase.OrderCache) *OrderStatusChangedHandler {
	return &OrderStatusChangedHandler{Repo: repo, Cache: cache}
}

func (h *OrderStatusChangedHandler) Handle(ctx context.Context, ev usecase.OrderStatusChangedMsg) error {
	// Map external status -> internal
	var newStatus domain.Status
	switch ev.Status {
	case "SUCCESS":
		newStatus = domain.StatusConfirmed
	default:
		newStatus = domain.StatusFailed
	}

	// Prefer guarded transition if repo supports it (PROCESSING -> target)
	type updater interface {
		UpdateStatusIf(ctx context.Context, id string, fromStatus, toStatus string) (bool, error)
	}
	if u, ok := any(h.Repo).(updater); ok {
		if _, err := u.UpdateStatusIf(ctx, ev.OrderID, string(domain.StatusProcessing), string(newStatus)); err != nil {
			return err
		}
	} else {
		if err := h.Repo.UpdateStatus(ctx, ev.OrderID, string(newStatus)); err != nil {
			return err
		}
	}

	// Cache best-effort
	if h.Cache != nil {
		_ = h.Cache.SetStatus(ctx, ev.OrderID, string(newStatus))
	}
	return nil
}
