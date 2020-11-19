<template>
  <div>
    <Alert ref="alert" />
    <b-card
      no-body
      class="overflow-hidden mb-4"
      v-for="plan in plans"
      v-bind:key="plan.id"
    >
      <b-row no-gutters>
        <b-col md="3">
          <b-card-img
            src="https://picsum.photos/400/400/?image=20"
            alt="Image"
            class="rounded-0"
          ></b-card-img>
        </b-col>
        <b-col md="9">
          <b-card-body :title="plan.name">
            <b-card-text>
              {{ plan.description }}
            </b-card-text>

            <b-list-group flush>
              <b-list-group-item
                v-for="part in plan.parts"
                v-bind:key="part.id"
              >
                {{ part.name }}: {{ plan.currency.toUpperCase() }}
                <b>{{ part.amountInCents / 100 }}/{{ part.unit }}</b>
              </b-list-group-item>
            </b-list-group>

            <div class="text-right">
              <b-button
                @click="createSubscription(plan.id)"
                variant="success"
                :disabled="formControl.submitDisabled"
              >
                <b-spinner small v-show="formControl.showSpinner"></b-spinner>
                {{ formControl.submitButtonText }}
              </b-button>
            </div>
          </b-card-body>
        </b-col>
      </b-row>
    </b-card>
  </div>
</template>

<script>
import Alert from "../components/Alert.vue";

const text = {
  submitNormal: "Create",
  submitting: "Creating",
};

export default {
  components: {
    Alert,
  },
  data() {
    return {
      plans: [],
      formControl: {
        showSpinner: false,
        submitDisabled: false,
        submitButtonText: text.submitNormal,
      },
    };
  },
  methods: {
    enableSubmit() {
      this.formControl.submitButtonText = text.submitNormal;
      this.formControl.submitDisabled = false;
      this.formControl.showSpinner = false;
    },
    disableSubmit() {
      this.formControl.submitButtonText = text.submitting;
      this.formControl.submitDisabled = true;
      this.formControl.showSpinner = true;
    },
    async createSubscription(planId) {
      this.disableSubmit();
      try {
        let subResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "POST",
          endpoint: "/subscriptions",
          body: {
            planId,
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
            this.$refs.alert.showAlert("danger", "Setup payment!");
            break;
          }
          case 200: {
            let createResp = await this.$store.dispatch({
              type: "makeAuthenticatedRequest",
              method: "POST",
              endpoint: "/instances",
              body: {
                subscriptionId: subJson.subscription.id,
              },
            });
            let createJson = await createResp.json();

            console.log(createJson);
            break;
          }
        }
      } catch (err) {
        this.$refs.alert.showAlert(
          "danger",
          "Unable to create instance at the moment, please try again later"
        );
      }
      this.enableSubmit();
    },
  },
  async mounted() {
    let resp = await this.$store.dispatch({
      type: "makeAuthenticatedRequest",
      method: "GET",
      endpoint: "/subscriptions/plans",
    });
    let json = await resp.json();
    this.plans = json.result;
  },
};
</script>