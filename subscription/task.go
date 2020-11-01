package subscription

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
	"go.uber.org/zap"
)

const (
	batchSize = 10
)

type TaskOptions struct {
	StripeClient        *client.API
	SubscriptionManager *Manager
	Logger              *zap.Logger
}

type Task struct {
	TaskOptions
}

// This background task will:
// 1. Synchronize database state with Stripe
// ~~2. Report aggregate usage~~

func NewTask(option TaskOptions) (*Task, error) {
	if option.StripeClient == nil {
		return nil, fmt.Errorf("nil StripeClient is invalid")
	}
	if option.SubscriptionManager == nil {
		return nil, fmt.Errorf("nil SubscriptionManager is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Task{
		TaskOptions: option,
	}, nil
}

func (t *Task) HandleStripe(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := t.reportLoop(ctx)
				if err != nil {
					t.Logger.Error("Cannot report usage to Stripe",
						zap.Error(err),
					)
				}
				time.Sleep(time.Minute * 15)
			}
		}
	}()
}

func (t *Task) reportLoop(ctx context.Context) error {
	var previousLast *time.Time
	for {
		last, err := t.reportUsage(ctx, previousLast)
		if err != nil {
			return err
		}
		if last == nil {
			return nil
		}
		previousLast = last
	}
}

func (t *Task) reportUsage(ctx context.Context, last *time.Time) (*time.Time, error) {
	now := time.Now()
	usages, err := t.SubscriptionManager.listUsages(ctx, listUsageOption{
		ReferenceTime: now,
		Before:        last,
		Limit:         batchSize,
	})

	if err != nil {
		t.Logger.Error("Cannot list usages from database",
			zap.Error(err),
		)
		return nil, err
	}

	if len(usages) == 0 {
		return nil, nil
	}

	lastEndDate := usages[len(usages)-1].EndDate

	for _, usage := range usages {
		if now.After(usage.SubscriptionItem.PeriodEnd) {
			// can't report to Stripe after period end
			continue
		}
		go func(u Usage) {
			logger := t.Logger.With(zap.String("SubscriptionItemID", u.SubscriptionItemID))

			quantity, err := unitConversion(&u)
			if err != nil {
				logger.Error("Error converting unit",
					zap.Error(err),
				)
			}

			_, err = t.StripeClient.UsageRecords.New(&stripe.UsageRecordParams{
				Params: stripe.Params{
					Context: ctx,
				},
				SubscriptionItem: stripe.String(u.SubscriptionItemID),
				Quantity:         stripe.Int64(quantity),
				Timestamp:        stripe.Int64(now.Unix()),
				Action:           stripe.String(string(stripe.UsageRecordActionSet)),
			})
			if err != nil {
				logger.Error("Unable to report usage to Stripe",
					zap.Error(err),
				)
			}
		}(usage)
	}

	return &lastEndDate, nil
}

func unitConversion(u *Usage) (int64, error) {
	// convert from singular unit to Part unit
	var quantity int64
	switch u.SubscriptionItem.Part.Unit {
	case "minute":
		quantity = int64(math.Ceil(float64(u.AggregateTotal) / 60))
		billablePeriod := u.SubscriptionItem.PeriodEnd.Sub(u.SubscriptionItem.PeriodStart)
		billableMinute := int64(billablePeriod / time.Minute)
		if quantity > billableMinute+1 {
			return 0, fmt.Errorf("Usage exceeds billable amount (max: %d, current: %d)", billableMinute, quantity)
		}
	default:
		return 0, fmt.Errorf("Unsupported Unit: %s", u.SubscriptionItem.Part.Unit)
	}
	return quantity, nil
}
