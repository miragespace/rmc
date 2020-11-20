<template>
  <div>
    <Alert ref="alert" :toast="true" />
    <Plan v-for="plan in plans" v-bind:key="plan.id" :plan="plan" />
  </div>
</template>

<script>
import * as Sentry from "@sentry/browser";
import Alert from "../components/Alert.vue";
import Plan from "../components/Plan.vue";

export default {
  components: {
    Alert,
    Plan,
  },
  data() {
    return {
      plans: [],
    };
  },
  methods: {},
  async mounted() {
    try {
      let resp = await this.$store.dispatch({
        type: "makeAuthenticatedRequest",
        method: "GET",
        endpoint: "/subscriptions/plans",
      });
      let json = await resp.json();
      if (resp.status == 200) {
        this.plans = json.result
          .filter((p) => !p.retired)
          .sort((a, b) => a.parameters.Players - b.parameters.Players);
      } else {
        this.$refs.alert.showDismissable(
          "danger",
          "Unable to load plans: " + json.error
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
};
</script>