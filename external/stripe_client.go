package external

import "github.com/stripe/stripe-go/v72/client"

func NewStripeClient(key string) *client.API {
	sc := &client.API{}
	sc.Init(key, nil)
	return sc
}
