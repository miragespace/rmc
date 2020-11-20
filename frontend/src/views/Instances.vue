<template>
  <div>
    <Alert ref="alert" :timeout="5" :toast="true" />
    <b-row v-for="(row, rowIndex) in instancesRC" :key="rowIndex" class="mb-4">
      <b-col v-for="instance in row" :key="instance.id" md="4">
        <Instance
          @error="showError"
          @success="showSuccess"
          @remove="removeInstance"
          :instance="instance"
        />
      </b-col>
    </b-row>
  </div>
</template>

<script>
import Alert from "../components/Alert.vue";
import * as Sentry from "@sentry/browser";
import Instance from "../components/Instance.vue";

export default {
  name: "InstancesView",
  components: {
    Alert,
    Instance,
  },
  data() {
    return {
      instances: [],
    };
  },
  computed: {
    instancesRC() {
      return this.instances.reduce(
        (rows, key, index) =>
          (index % 3 == 0
            ? rows.push([key])
            : rows[rows.length - 1].push(key)) && rows,
        []
      );
    },
  },
  methods: {
    async loadInstances() {
      try {
        let instResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "GET",
          endpoint: "/instances",
        });
        let instJson = await instResp.json();
        if (instResp.status == 200) {
          for (let inst of instJson.result) {
            this.instances.push(inst);
          }
        } else {
          this.$refs.alert.showDismissable(
            "danger",
            instJson.error + ": " + instJson.messages.join(", ")
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
    showError(msg) {
      this.$refs.alert.showAlert("danger", msg);
    },
    showSuccess(msg) {
      this.$refs.alert.showAlert("success", msg);
    },
    removeInstance(id) {
      const index = this.instances.findIndex((i) => i.id === id);
      if (index < 0) return;
      this.instances.splice(index, 1);
    },
  },
  async mounted() {
    await this.loadInstances();
  },
};
</script>