<template>
  <div>
    <b-card class="mb-4">
      <b-form @submit="setupPaymentMethod">
        <b-row>
          <b-col md="8">
            <b-form-group id="cc-name-group" label="Name on card">
              <b-form-input
                id="input-cc-name"
                required
                v-model="stripe.ccName"
                placeholder="Jane Doe"
              ></b-form-input>
            </b-form-group>
            <b-form-group
              id="stripe-card-element-group"
              label="Credit or debit card"
            >
              <div ref="card" class="form-control">
                <!-- A Stripe Element will be inserted here. -->
              </div>
            </b-form-group>
          </b-col>
          <b-col md="4">
            <b-card-body>
              <p>
                Your card will not be charged until an subscription/instance is
                created.
              </p>
              <p><small>Payment is processed securely via Stripe.</small></p>
              <b-overlay
                :show="formControl.submitDisabled"
                rounded
                opacity="0.6"
                spinner-small
                spinner-variant="primary"
                class="d-inline-block"
              >
                <b-button
                  type="submit"
                  variant="primary"
                  :disabled="formControl.submitDisabled"
                >
                  {{ buttonText }}
                </b-button>
              </b-overlay>
            </b-card-body>
          </b-col>
        </b-row>
      </b-form>
      <Alert ref="alert" />
    </b-card>
  </div>
</template>

<script>
import * as Sentry from "@sentry/browser";
import Alert from "../components/Alert.vue";

const STRIPE_API_TOKEN = process.env.VUE_APP_STRIPE_PUBLISHABLE_KEY;

export default {
  components: {
    Alert,
  },
  props: {
    initialMessage: {
      type: Object,
      default: null,
    },
    buttonText: {
      type: String,
      default: "Setup",
    },
    handleSubscription: {
      type: Object,
      default: null,
    },
  },
  data() {
    return {
      stripe: {
        instance: null,
        elements: null,
        card: null,
        ccName: "",

        paymentMethod: null,
      },
      formControl: {
        submitDisabled: false,
      },
    };
  },
  methods: {
    enableSubmit() {
      this.formControl.submitDisabled = false;
    },
    disableSubmit() {
      this.formControl.submitDisabled = true;
    },
    async handleCardSetupRequired({ subscription, paymentMethod }) {
      let setupIntent = subscription.pending_setup_intent;
      if (setupIntent && setupIntent.status === "requires_action") {
        console.log("card setup required");
        let stripeResp = await this.stripe.instance.confirmCardSetup(
          // notice that this is confirmCard*Setup*
          setupIntent.client_secret,
          {
            payment_method: paymentMethod.id,
          }
        );
        if (stripeResp.error) {
          throw stripeResp;
        } else {
          return {
            subscription,
            paymentMethod,
          };
        }
      } else {
        return {
          subscription,
          paymentMethod,
        };
      }
    },
    async handlePaymentThatRequiresCustomerAction({
      subscription,
      invoice,
      paymentMethod,
    }) {
      let paymentIntent = invoice
        ? invoice.payment_intent
        : subscription.latest_invoice.payment_intent;
      if (!paymentIntent) {
        return {
          subscription,
          paymentMethod,
        };
      }
      if (paymentIntent.status === "requires_action") {
        console.log("payment confirm required");
        let stripeResp = await this.stripe.instance.confirmCardPayment(
          // notice that this is confirmCard*Payment*
          paymentIntent.client_secret,
          {
            payment_method: paymentMethod.id,
          }
        );
        if (stripeResp.error) {
          throw stripeResp;
        } else {
          return {
            subscription,
            paymentMethod,
          };
        }
      } else {
        // no actions required
        return {
          subscription,
          paymentMethod,
        };
      }
    },
    async userAction() {
      try {
        let subscription = this.handleSubscription;
        let paymentMethod = await this.getPaymentMethod();
        if (paymentMethod === null) return;

        let cardSetupResp = await this.handleCardSetupRequired({
          subscription,
          paymentMethod,
        });
        let actionResp = await this.handlePaymentThatRequiresCustomerAction(
          cardSetupResp
        );
        console.log("user action succeed");
        console.log(actionResp.subscription);
        this.$emit("subscriptionSetup", actionResp.subscription);
      } catch (err) {
        console.log("user action failed");
        this.$refs.alert.showDismissable("danger", err.error.message);
      }
      this.formControl.submitDisabled = false;
    },
    async setupPaymentMethod(evt) {
      evt.preventDefault();
      this.disableSubmit();

      try {
        let stripeResp = await this.stripe.instance.createPaymentMethod({
          type: "card",
          card: this.stripe.card,
          billing_details: {
            name: this.stripe.ccName,
          },
        });
        if (stripeResp.error) {
          this.$refs.alert.showDismissable("danger", stripeResp.error.message);
        } else {
          let paymentMethod = stripeResp.paymentMethod;
          let setupResp = await this.$store.dispatch({
            type: "makeAuthenticatedRequest",
            method: "POST",
            endpoint: "/subscriptions/initialSetup",
            body: {
              paymentMethodID: paymentMethod.id,
            },
          });
          if (setupResp.status !== 200) {
            let setupJson = await setupResp.json();
            if (setupJson.result && setupJson.result.message) {
              this.$refs.alert.showDismissable(
                "danger",
                setupJson.result.message
              );
            } else {
              this.$refs.alert.showDismissable(
                "danger",
                setupJson.error + ": " + setupJson.messages.join("; ")
              );
            }
          } else {
            this.$emit("paymentSetup", paymentMethod);
          }
        }
      } catch (err) {
        Sentry.captureException(err);
        console.log(err);
      }

      this.enableSubmit();
    },
    async getPaymentMethod() {
      let custResp = await this.$store.dispatch({
        type: "makeAuthenticatedRequest",
        method: "GET",
        endpoint: "/customers/stripe",
      });
      if (custResp.status !== 200) {
        this.$refs.alert.showDismissable(
          "danger",
          "Unable to fetch payment details, please try again later"
        );
        return null;
      } else {
        let custJson = await custResp.json();
        return custJson.result.invoice_settings.default_payment_method;
      }
    },
    configureStripe() {
      this.stripe.instance = window.Stripe(STRIPE_API_TOKEN);
      this.stripe.elements = this.stripe.instance.elements();
      this.stripe.card = this.stripe.elements.create("card");
      this.stripe.card.mount(this.$refs.card);
      this.stripe.card.on("change", (event) => {
        if (event.error) {
          this.$refs.alert.showDismissable("danger", event.error.message);
        } else {
          this.$refs.alert.hideAlert();
        }
      });
    },
  },
  async mounted() {
    this.configureStripe();
    if (this.initialMessage) {
      this.$refs.alert.showAlert(
        this.initialMessage.type,
        this.initialMessage.message
      );
    }
    if (this.handleSubscription !== null) {
      this.formControl.submitDisabled = true;
      await this.userAction();
    }
  },
};
</script>