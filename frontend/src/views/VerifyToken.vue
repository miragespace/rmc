<template>
  <div>
    <Alert ref="alert" />
    <b-container> Logging you in... </b-container>
  </div>
</template>

<script>
import * as Sentry from "@sentry/browser";
import Alert from "../components/Alert.vue";

const BASE_URL = process.env.VUE_APP_API_ENDPOINT;

export default {
  name: "VerifyToken",
  components: {
    Alert,
  },
  data() {
    return {};
  },
  async mounted() {
    let uid = this.$route.params.uid;
    let token = this.$route.params.token;

    let data = BASE_URL + "/customers/" + uid + "/" + token;
    try {
      let resp = await fetch(data);
      let json = await resp.json();

      if (resp.status == 200) {
        this.$refs.alert.showAlert(
          "success",
          "Login successful! Redirecting you to instances page..."
        );

        await this.$store.commit({
          type: "setBearerToken",
          token: json.result.token,
        });

        this.$router.push({
          name: "Instances",
        });
      } else {
        this.$refs.alert.showDismissable("danger", json.error);
      }
    } catch (err) {
      Sentry.captureException(err);
      this.$refs.alert.showDismissable(
        "danger",
        "An unexpected error has occured: " + err.message
      );
    }
  },
  methods: {},
};
</script>