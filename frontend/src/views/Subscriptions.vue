<template>
  <div>
    <Alert ref="alert" :timeout="5" :toast="true" />
    <b-row
      v-for="(row, rowIndex) in subscriptionsRC"
      :key="rowIndex"
      class="mb-4"
    >
      <b-col v-for="subscription in row" :key="subscription.id" lg="6">
        <Subscription :subscription="subscription" />
      </b-col>
    </b-row>
    <b-overlay
      :show="formControl.isLoading"
      rounded
      opacity="0.6"
      spinner-small
      spinner-variant="primary"
      v-show="formControl.cursor !== null"
    >
      <b-button
        block
        variant="info"
        size="sm"
        href="#"
        @click="delayedLoadAppend"
      >
        Load previous 10 entries
      </b-button>
    </b-overlay>
  </div>
</template>

<script>
import * as Sentry from "@sentry/browser";
import Alert from "../components/Alert.vue";
import Subscription from "../components/Subscription.vue";

export default {
  name: "SubscriptionsView",
  components: {
    Alert,
    Subscription,
  },
  data() {
    return {
      subscriptions: [],
      formControl: {
        cursor: null,
        isLoading: false,
      },
    };
  },
  computed: {
    subscriptionsRC() {
      return this.subscriptions.reduce(
        (rows, key, index) =>
          (index % 2 == 0
            ? rows.push([key])
            : rows[rows.length - 1].push(key)) && rows,
        []
      );
    },
  },
  methods: {
    delay(ms) {
      return new Promise((resolve) => {
        setTimeout(resolve, ms);
      });
    },
    async delayedLoadAppend() {
      this.formControl.isLoading = true;
      await this.delay(1000);
      await this.doLoadAppend();
      this.formControl.isLoading = false;
    },
    async loadAppend() {
      this.formControl.isLoading = true;
      await this.doLoadAppend();
      this.formControl.isLoading = false;
    },
    async doLoadAppend() {
      let ep = "/subscriptions";
      if (this.formControl.cursor) {
        let params = {
          before: this.formControl.cursor,
        };
        ep += "?" + new URLSearchParams(params).toString();
      }
      try {
        let resp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "GET",
          endpoint: ep,
        });
        let json = await resp.json();

        if (resp.status === 200) {
          this.subscriptions.push(...json.result);

          if (json.result.length > 0) {
            this.formControl.cursor =
              json.result[json.result.length - 1].createdAt;
          } else {
            this.formControl.cursor = null;
          }
        } else {
          this.$refs.alert.showAlert(
            "danger",
            json.error + ": " + json.messages.join("; ")
          );
        }
      } catch (err) {
        Sentry.captureException(err);
        this.$refs.alert.showDismissable(
          "danger",
          "An unexpected error has occured: " + err.message
        );
      }
    },
  },
  async mounted() {
    await this.loadAppend();
  },
};
</script>