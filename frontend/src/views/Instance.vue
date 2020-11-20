<template>
  <div>
    <Alert ref="alert" :timeout="5" :toast="true" />
    <Instance
      @error="showError"
      @success="showSuccess"
      @remove="removeInstance"
      :isSingle="true"
      :instance="instance"
      v-if="instance.id !== null"
    />
  </div>
</template>

<script>
import Alert from "../components/Alert.vue";
import Instance from "../components/Instance.vue";

export default {
  name: "InstanceView",
  components: {
    Alert,
    Instance,
  },
  data() {
    return {
      instance: {
        id: null,
        parameters: {},
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
    removeInstance() {
      this.$router.push({
        name: "Instances",
      });
    },
  },
  created() {
    let instanceId = this.$route.params.id;
    this.instance.id = instanceId;
  },
};
</script>