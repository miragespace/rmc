<template>
  <div>
    <b-card class="mb-4">
      <Alert ref="alert" />
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
            <b-form-group id="cc-name-group" label="Credit or debit card">
              <div ref="card" class="form-control">
                <!-- A Stripe Element will be inserted here. -->
              </div>
            </b-form-group>
          </b-col>
          <b-col md="4">
            <b-card-body>
              <p>
                You will not be charged until an instance is created. Payment is
                processed securely via Stripe.
              </p>
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
    buttonText: {
      type: String,
      default: "Setup",
    },
  },
  data() {
    return {
      stripe: {
        instance: null,
        elements: null,
        card: null,
        ccName: "",
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
            this.$refs.alert.showDismissable("danger", setupJson.error);
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
    configureStripe() {
      this.stripe.instance = window.Stripe(STRIPE_API_TOKEN);
      this.stripe.elements = this.stripe.instance.elements();
      this.stripe.card = this.stripe.elements.create("card");
      this.stripe.card.mount(this.$refs.card);
    },
  },
  async mounted() {
    this.configureStripe();
  },
};
</script>