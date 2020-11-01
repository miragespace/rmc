package subscription

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"

	"github.com/golang/protobuf/ptypes"
	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
	"go.uber.org/zap"
)

type TaskOptions struct {
	StripeClient        *client.API
	Consumer            broker.Consumer
	SubscriptionManager *Manager
	Logger              *zap.Logger
}

type Task struct {
	TaskOptions
}

func NewTask(option TaskOptions) (*Task, error) {
	if option.StripeClient == nil {
		return nil, fmt.Errorf("nil StripeClient is invalid")
	}
	if option.Consumer == nil {
		return nil, fmt.Errorf("nil Consumer is invalid")
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

func (t *Task) HandleTask(ctx context.Context) error {
	tChan, err := t.Consumer.ReceiveTask(ctx, spec.SubscriptionTask)
	if err != nil {
		return extErrors.Wrap(err, "Cannot get subscription task channel")
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case task := <-tChan:
				if task.GetType() != protocol.Task_Subscription {
					t.Logger.Error("Received non Subscription task")
					continue
				}
				timestamp := task.GetTimestamp()
				if timestamp == nil {
					t.Logger.Error("Received nil Timestamp")
					continue
				}
				subTask := task.GetSubscriptionTask()
				if subTask == nil {
					t.Logger.Error("Task has nil SubscriptionTask")
					continue
				}
				subscriptionItemID := subTask.GetSubscriptionItemID()
				if len(subscriptionItemID) == 0 {
					t.Logger.Error("Received empty SubscriptionItemID")
					continue
				}

				logger := t.Logger.With(zap.String("SubscriptionItemID", subscriptionItemID))

				switch subTask.GetFunction() {
				case protocol.SubscriptionTask_ReportUsage:
					if err := t.reportUsage(ctx, task); err != nil {
						logger.Error("Unable to report usage",
							zap.Error(err),
						)
					}
				case protocol.SubscriptionTask_Synchronize:
					if err := t.synchronizePeriod(ctx, task); err != nil {
						logger.Error("Unable to synchronize with Stripe",
							zap.Error(err),
						)
					}
				default:
					logger.Error("SubscriptionTask received unknown Function")
				}
			}
		}
	}()

	return nil
}

func (t Task) synchronizePeriod(ctx context.Context, pb *protocol.Task) error {
	return nil
}

func (t *Task) reportUsage(ctx context.Context, pb *protocol.Task) error {
	subscriptionItemID := pb.GetSubscriptionTask().GetSubscriptionItemID()
	referenceTime, _ := ptypes.Timestamp(pb.GetTimestamp())

	u, err := t.SubscriptionManager.getUsageBySubscriptionItemID(ctx, usageLookupOption{
		SubscriptionItemID: subscriptionItemID,
		ReferenceTime:      referenceTime,
	})
	if err != nil {
		return extErrors.Wrap(err, "Cannot get usage by subscription item id")
	}

	quantity, err := unitConversion(u)
	if err != nil {
		return extErrors.Wrap(err, "Error converting unit for usage")
	}

	_, err = t.StripeClient.UsageRecords.New(&stripe.UsageRecordParams{
		Params: stripe.Params{
			Context: ctx,
		},
		SubscriptionItem: stripe.String(u.SubscriptionItemID),
		Quantity:         stripe.Int64(quantity),
		Timestamp:        stripe.Int64(referenceTime.Unix()),
		Action:           stripe.String(string(stripe.UsageRecordActionSet)),
	})
	if err != nil {
		return extErrors.Wrap(err, "Cannot report usage to Stripe")
	}

	return nil
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
