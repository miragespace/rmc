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
      formControl: {
        isLoading: false,
        cursor: null,
      },
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
      let ep = "/instances";
      if (this.formControl.cursor) {
        let params = {
          before: this.formControl.cursor,
        };
        ep += "?" + new URLSearchParams(params).toString();
      }
      try {
        let instResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "GET",
          endpoint: ep,
        });
        let instJson = await instResp.json();
        if (instResp.status == 200) {
          this.instances.push(...instJson.result);

          if (instJson.result.length > 0) {
            this.formControl.cursor =
              instJson.result[instJson.result.length - 1].createdAt;
          } else {
            this.formControl.cursor = null;
          }
        } else {
          this.$refs.alert.showDismissable(
            "danger",
            instJson.error + ": " + instJson.messages.join(", ")
          );
        }
      } catch (err) {
        console.log(err);
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
    await this.loadAppend();
  },
};
</script>