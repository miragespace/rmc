<template>
  <div>
    <Alert ref="alert" :timeout="5" :toast="true" />
    <Subscription
      @error="showError"
      @success="showSuccess"
      :isSingle="true"
      :subscription="subscription"
      v-if="subscription.id !== null"
    />
  </div>
</template>

<script>
import Alert from "../components/Alert.vue";
import Subscription from "../components/Subscription.vue";

export default {
  name: "SubscriptionView",
  components: {
    Alert,
    Subscription,
  },
  data() {
    return {
      subscription: {
        id: null,
      },
    };
  },
  methods: {
    showError(msg) {
      this.$refs.alert.showAlert("danger", msg);
    },
    showSuccess(msg) {
      this.$refs.alert.showAlert("success", msg);
    },
  },
  created() {
    let subscriptionId = this.$route.params.id;
    this.subscription.id = subscriptionId;
  },
};
</script>