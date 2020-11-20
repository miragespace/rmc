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
              </b-list-group-item>
            </b-list-group>

            <b-row no-gutters>
              <b-col md="6"></b-col>
              <b-col md="6">
                <b-form @submit="createSubscription">
                  <b-row>
                    <b-col md="5">
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
                    <b-col md="5">
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
                    <b-col md="2">
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

            <div class="text-right"></div>
          </b-card-body>
        </b-col>
      </b-row>
    </b-card>
    <Alert ref="alert" />
    <PaymentSetup
      buttonText="Setup and create"
      @paymentSetup="paymentSetup"
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
    plan: {
      type: Object,
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
      instance: {
        version: null,
        edition: null,
      },
      // TODO: load from server
      editions: [
        { text: "Java", value: "java" },
        { text: "Bedrock", value: "bedrock" },
      ],
      versions: {
        java: ["1.16"],
        bedrock: ["1.1"],
      },
      formControl: {
        showPaymentSetup: false,
        submitDisabled: false,
      },
    };
  },
  methods: {
    enableSubmit() {
      this.formControl.submitDisabled = false;
    },
    disableSubmit() {
      this.formControl.showPaymentSetup = false;
      this.formControl.submitDisabled = true;
    },
    async paymentSetup(paymentMethod) {
      this.$refs.alert.showAlert("success", "Creating your instance...");
      await this.createSubscription();
      console.log(paymentMethod);
    },
    async createSubscription(evt) {
      evt.preventDefault();
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
            this.$refs.alert.showAlert(
              "danger",
              "Please setup default payment method first"
            );
            this.formControl.showPaymentSetup = true;
            break;
          }
          case 200: {
            console.log(subJson);

            let stripeResponse = subJson.result.stripeResponse;

            if (stripeResponse.status !== "active") {
              this.$refs.alert.showDismissable(
                "danger",
                "Stripe setup failed!"
              );
            } else {
              let subscription = subJson.result.subscription;
              await this.createInstance(subscription);
            }
            break;
          }
        }
      } catch (err) {
        console.log(err);
        this.$refs.alert.showAlert(
          "danger",
          "Unable to create instance at the moment, please try again later"
        );
        Sentry.captureException(err);
      }
      this.enableSubmit();
    },
    async createInstance(subscription) {
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
      }
    },
  },
};
</script>