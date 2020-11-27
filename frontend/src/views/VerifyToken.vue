<template>
  <div>
    <Alert ref="alert" />
    <b-container> Logging you in... </b-container>
  </div>
</template>

<script>
import * as Sentry from "@sentry/browser";
import Alert from "../components/Alert.vue";

export default {
  name: "VerifyToken",
  components: {
    Alert,
  },
  data() {
    return {};
  },
  methods: {
    delay(ms) {
      return new Promise((resolve) => {
        setTimeout(resolve, ms);
      });
    },
  },
  async mounted() {
    let uid = this.$route.params.uid;
    let token = this.$route.params.token;

    try {
      await this.$store.dispatch({
        type: "requestTokens",
        uid: uid,
        token: token,
      });
      this.$refs.alert.showAlert(
        "success",
        "Login successful! Redirecting you to instances page..."
      );

      await this.delay(2000);

      this.$router.push({
        name: "Instances",
      });
    } catch (err) {
      if (err.apiError === true) {
        this.$refs.alert.showDismissable(
          "danger",
          err.error + ": " + err.message
        );
      } else {
        Sentry.captureException(err);
        this.$refs.alert.showDismissable(
          "danger",
          "An unexpected error has occured: " + err.message
        );
      }
    }
  },
};
</script>