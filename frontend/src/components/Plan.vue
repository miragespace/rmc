<template>
  <div>
    <b-card no-body class="overflow-hidden mb-4">
      <b-row no-gutters>
        <!-- <b-col md="3">
          <b-card-img
            src="https://via.placeholder.com/400"
            alt="Image"
            class="rounded-0"
          ></b-card-img>
        </b-col> -->
        <b-col md="12">
          <b-card-body :title="plan.name">
            <b-card-text>
              {{ plan.description }}
              <b-badge pill variant="warning" size="sm" v-if="plan.retired">
                Retired
              </b-badge>
            </b-card-text>

            <b-list-group flush>
              <b-list-group-item
                v-for="part in plan.parts"
                v-bind:key="part.id"
              >
                {{ part.name }}:
                <b>
                  {{ plan.currency.toUpperCase() }}
                  {{ part.amountInCents / 100 }}/{{ part.unit }}
                </b>
                <b-badge pill variant="info" size="sm" v-if="!part.primary">
                  Addon
                </b-badge>
              </b-list-group-item>
            </b-list-group>

            <b-row no-gutters v-if="showCreate">
              <b-col lg="6"></b-col>
              <b-col lg="6">
                <b-form @submit="createStripeSubscription">
                  <b-row>
                    <b-col lg="5" class="mb-2">
                      <b-form-select
                        size="sm"
                        v-model="instance.edition"
                        :options="editions"
                        required
                      >
                        <template #first>
                          <b-form-select-option :value="null" disabled>
                            -- Server edition --
                          </b-form-select-option>
                        </template>
                      </b-form-select>
                    </b-col>
                    <b-col lg="5" class="mb-2">
                      <b-form-select
                        size="sm"
                        v-model="instance.version"
                        :options="versions[instance.edition]"
                        required
                      >
                        <template #first>
                          <b-form-select-option :value="null" disabled>
                            -- Server version --
                          </b-form-select-option>
                        </template>
                      </b-form-select>
                    </b-col>
                    <b-col lg="2" class="mb-2">
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
                          variant="success"
                          :disabled="formControl.submitDisabled"
                        >
                          Create
                        </b-button>
                      </b-overlay>
                    </b-col>
                  </b-row>
                </b-form>
              </b-col>
            </b-row>
          </b-card-body>
        </b-col>
      </b-row>
    </b-card>
    <Alert ref="alert" />
    <PaymentSetup
      buttonText="Setup and create"
      :initialMessage="stripe.initialMessage"
      :handleSubscription="stripe.subscription"
      @paymentSetup="paymentSetup"
      @subscriptionSetup="subscriptionSetup"
      v-if="formControl.showPaymentSetup"
    />
  </div>
</template>

<script>
import * as Sentry from "@sentry/browser";
import Alert from "../components/Alert.vue";
import PaymentSetup from "../components/PaymentSetup.vue";

export default {
  components: {
    Alert,
    PaymentSetup,
  },
  props: {
    showCreate: {
      type: Boolean,
      default: true,
    },
    plan: {
      type: Object,
      required: true,
    },
  },
  watch: {
    "instance.edition": {
      handler: function () {
        // must use function syntax
        this.instance.version = null;
      },
    },
  },
  data() {
    return {
      stripe: {
        subscription: null,
        initialMessage: {
          type: "danger",
          message: "placeholder",
        },
      },
      instance: {
        version: null,
        edition: null,
      },
      editions: [
        { text: "Java", value: "java" },
        { text: "Bedrock", value: "bedrock" },
      ],
      // TODO: load supported versions from server
      versions: {
        java: ["1.16.1"],
        bedrock: ["1.16.1"],
      },
      formControl: {
        showPaymentSetup: false,
        submitDisabled: false,
      },
    };
  },
  methods: {
    delay(ms) {
      return new Promise((resolve) => {
        setTimeout(resolve, ms);
      });
    },
    enableSubmit() {
      this.formControl.submitDisabled = false;
    },
    disableSubmit() {
      this.formControl.showPaymentSetup = false;
      this.formControl.submitDisabled = true;
    },
    async paymentSetup(paymentMethod) {
      console.log("updated payment method");
      console.log(paymentMethod);
      this.formControl.subscriptionSetup = false;
      await this.createStripeSubscription();
    },
    async subscriptionSetup(subscription) {
      console.log("user confirmed");
      console.log(subscription);
      this.formControl.subscriptionSetup = false;
      await this.createSubscription(subscription);
    },
    async createStripeSubscription(evt) {
      if (evt) {
        evt.preventDefault();
      }
      this.disableSubmit();
      try {
        let subResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "POST",
          endpoint: "/subscriptions",
          body: {
            planId: this.plan.id,
          },
        });
        let subJson = await subResp.json();

        switch (subResp.status) {
          case 500: {
            this.$refs.alert.showAlert(
              "danger",
              subJson.error + ": " + subJson.messages.join("; ")
            );
            break;
          }
          case 403: {
            this.stripe.initialMessage.type = "danger";
            this.stripe.initialMessage.message =
              "Please setup default payment method first.";
            this.stripe.subscription = null;
            this.formControl.showPaymentSetup = true;
            break;
          }
          case 200: {
            let subscription = subJson.result;
            if (subscription.status !== "active") {
              // requires user actions
              this.stripe.initialMessage.type = "danger";
              this.stripe.initialMessage.message =
                "Your card requires further actions.";
              this.stripe.subscription = subscription;
              this.formControl.showPaymentSetup = true;
            } else {
              await this.createSubscription(subscription);
            }
            break;
          }
        }
      } catch (err) {
        Sentry.captureException(err);
        this.$refs.alert.showAlert(
          "danger",
          "Unable to create Stripe subscription at the moment, please try again later"
        );
      }
      this.enableSubmit();
    },
    async createSubscription(stripeSubscription) {
      try {
        let createResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "PUT",
          endpoint: "/subscriptions/" + stripeSubscription.id,
        });
        let createJson = await createResp.json();

        if (createResp.status !== 200) {
          this.$refs.alert.showDismissable(
            "danger",
            createJson.error + ": " + createJson.messages.join("; ")
          );
        } else {
          await this.createInstance(createJson.result);
        }
      } catch (err) {
        Sentry.captureException(err);
        this.$refs.alert.showAlert(
          "danger",
          "Unable to create subscription at the moment, please try again later"
        );
      }
    },
    async createInstance(subscription) {
      this.$refs.alert.showAlert("success", "Creating your instance...");
      await this.delay(1000);
      try {
        let createResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "POST",
          endpoint: "/instances",
          body: {
            subscriptionId: subscription.id,
            serverVersion: this.instance.version,
            serverEdition: this.instance.edition,
          },
        });
        let createJson = await createResp.json();

        // TODO: detect no available hosts

        if (createResp.status == 200) {
          let instance = createJson.result;

          this.$router.push({
            name: "Instance",
            params: {
              id: instance.id,
            },
          });
        } else {
          this.$refs.alert.showAlert(
            "danger",
            createJson.error + ": " + createJson.messages.join("; ")
          );
        }
      } catch (err) {
        Sentry.captureException(err);
        this.$refs.alert.showAlert(
          "danger",
          "Unable to create instance at the moment, please try again later"
        );
      }
    },
  },
};
</script>