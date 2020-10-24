package external

import "github.com/stripe/stripe-go/v71/client"

func NewStripeClient(key string) *client.API {
	sc := &client.API{}
	sc.Init(key, nil)
	return sc
}
