package subscription

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
)

var lookupKeyRegex = regexp.MustCompile("[^a-zA-Z0-9]+")

//				Subscription struct helpers

func (s *Subscription) findSubscriptionItemByPartID(partID string) *SubscriptionItem {
	if s == nil || len(s.SubscriptionItems) == 0 {
		return nil
	}
	for k, item := range s.SubscriptionItems {
		if item.PartID == partID {
			return &s.SubscriptionItems[k]
		}
	}
	return nil
}

func (s *Subscription) findPrimaryVariableItem() *SubscriptionItem {
	for k, item := range s.SubscriptionItems {
		if item.Part.Primary && item.Part.Type == VariableType {
			return &s.SubscriptionItems[k]
		}
	}
	return nil
}

func (s *Subscription) fromStripeResponse(sub *stripe.Subscription, plan *Plan) error {
	items := make([]SubscriptionItem, 0, 2)
	for _, subItem := range sub.Items.Data {
		part := plan.lookupPartByLookupKey(subItem.Price.LookupKey)
		if part == nil {
			return fmt.Errorf("Inconsistent data: no corresponding Price/Part")
		}
		item := SubscriptionItem{
			ID:             subItem.ID,
			PartID:         part.ID,
			SubscriptionID: sub.ID,
			Part:           *part,
		}
		items = append(items, item)
	}

	var subState State
	if sub.Status == stripe.SubscriptionStatusActive && sub.PendingSetupIntent == nil {
		subState = StateActive
	} else {
		subState = StatePending
	}

	*s = Subscription{
		ID:                sub.ID,
		PlanID:            plan.ID,
		CustomerID:        sub.Customer.ID,
		State:             subState,
		SubscriptionItems: items,
		PeriodStart:       time.Unix(sub.CurrentPeriodStart, 0),
		PeriodEnd:         time.Unix(sub.CurrentPeriodEnd, 0),
		Plan:              *plan,
	}

	return nil
}

//				Plan struct helpers

// lookupKey will generate a unique LookupKey on stripe to identify each Part of the Plan
func (p *Plan) lookupKey(part Part) string {
	planName := lookupKeyRegex.ReplaceAllString(p.Name, "-")
	partName := lookupKeyRegex.ReplaceAllString(part.Name, "-")
	amountPart := fmt.Sprintf("%f", part.AmountInCents)
	return strings.ToLower(fmt.Sprintf("%s_%s_%s_%s_%s_%s", planName, p.Interval, part.Type, partName, amountPart, part.Unit))
}

// createPlanOnStripe will create missing Plan as Product on Stripe
func (p *Plan) createPlanOnStripe(ctx context.Context, s *client.API) error {
	if len(p.ID) == 0 {
		// Create corresponding Product
		prodParams := &stripe.ProductParams{
			Params: stripe.Params{
				Context:  ctx,
				Metadata: p.Parameters,
			},
			Active:      stripe.Bool(true),
			Name:        stripe.String(p.Name),
			Description: stripe.String(p.Description),
		}
		stripeProduct, err := s.Products.New(prodParams)
		if err != nil {
			return extErrors.Wrap(err, "Cannot create Plan as Product on Stripe")
		}
		// Populate the Product.ID
		p.ID = stripeProduct.ID
	}

	return p.createPartsOnStripe(ctx, s)
}

// createPartsOnStripe will create missing Parts as Price on Stripe
func (p *Plan) createPartsOnStripe(ctx context.Context, s *client.API) error {

	if len(p.ID) == 0 {
		return fmt.Errorf("Plan ID doesn't exist, please create Plan before Parts")
	}

	fixedRecurring := &stripe.PriceRecurringParams{
		AggregateUsage: nil,
		Interval:       stripe.String(p.Interval),
		IntervalCount:  stripe.Int64(1),
		UsageType:      stripe.String("licensed"),
	}
	variableRecurring := &stripe.PriceRecurringParams{
		AggregateUsage: stripe.String(string(stripe.PriceRecurringAggregateUsageLastDuringPeriod)),
		Interval:       stripe.String(p.Interval),
		IntervalCount:  stripe.Int64(1),
		UsageType:      stripe.String("metered"),
	}

	for k, part := range p.Parts {
		if len(part.ID) > 0 {
			// already exist, don't create
			continue
		}

		pParams := &stripe.PriceParams{
			Params: stripe.Params{
				Context: ctx,
				Metadata: map[string]string{
					"Type":    string(part.Type),
					"Primary": strconv.FormatBool(part.Primary),
				},
			},
			Active:            stripe.Bool(true),
			Nickname:          stripe.String(part.Name),
			BillingScheme:     stripe.String("per_unit"),
			Currency:          stripe.String(p.Currency),
			UnitAmountDecimal: stripe.Float64(part.AmountInCents),
			Product:           stripe.String(p.ID),
			LookupKey:         stripe.String(p.lookupKey(part)),
		}
		switch part.Type {
		case FixedType:
			pParams.Recurring = fixedRecurring
		case VariableType:
			pParams.Recurring = variableRecurring
		}
		partPrice, err := s.Prices.New(pParams)
		if err != nil {
			return extErrors.Wrap(err, "Cannot create Part as Price on Stripe")
		}
		// Populate the ID back to the Part
		p.Parts[k].ID = partPrice.ID
	}
	return nil
}

func (p *Plan) lookupPartByLookupKey(lookupKey string) *Part {
	for k, part := range p.Parts {
		if lookupKey == p.lookupKey(part) {
			return &p.Parts[k]
		}
	}
	return nil
}

func (p *Plan) findVariablePart(primary bool, partID *string) *Part {
	if !primary && partID == nil {
		return nil
	}
	if p == nil || p.ID == "" || len(p.Parts) == 0 {
		return nil
	}
	for _, part := range p.Parts {
		if primary {
			if part.Primary && part.Type == VariableType {
				return &part
			}
		} else {
			if !part.Primary && part.Type == VariableType && part.ID == *partID {
				return &part
			}
		}
	}
	return nil
}

func (p *Plan) toStripeSubscriptionParams(ctx context.Context, customerID string) *stripe.SubscriptionParams {
	sParams := &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
		Customer: stripe.String(customerID),
		Items:    []*stripe.SubscriptionItemsParams{},
	}

	for _, part := range p.Parts {
		pParams := &stripe.SubscriptionItemsParams{
			Price: stripe.String(part.ID),
		}
		switch part.Type {
		case FixedType:
			pParams.Quantity = stripe.Int64(1)
		case VariableType:
			pParams.Quantity = nil
		}
		sParams.Items = append(sParams.Items, pParams)
	}

	return sParams
}
