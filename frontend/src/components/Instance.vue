<template>
  <div>
    <b-overlay :show="formControl.showBusy" rounded="lg" opacity="0.6">
      <template #overlay>
        <div class="d-flex align-items-center">
          <b-spinner small type="grow" variant="secondary"></b-spinner>
          <b-spinner type="grow" variant="dark"></b-spinner>
          <b-spinner small type="grow" variant="secondary"></b-spinner>
          <!-- We add an SR only text for screen readers -->
          <span class="sr-only">Please wait...</span>
        </div>
      </template>
      <b-card
        no-body
        class="mb-2"
        header-tagg="header"
        footer-tag="footer"
        :header-bg-variant="headerBg"
      >
        <template #header>
          <b-button
            @click="delayedReload"
            size="sm"
            :variant="headerBg"
            v-b-tooltip.hover="{ placement: 'right', variant: 'info' }"
            title="Refresh instance status"
          >
            {{ data.state }}
            <b-icon icon="arrow-repeat"></b-icon>
          </b-button>
        </template>
        <b-card-body>
          <b-card-sub-title class="mb-2 mt-2">
            <b-icon icon="power"></b-icon> Lifecycle Control
          </b-card-sub-title>
          <b-list-group horizontal="lg">
            <b-list-group-item href="#" @click="lifeCycleControl('Start')">
              <b-icon icon="play-fill"></b-icon>
              Start
            </b-list-group-item>
            <b-list-group-item href="#" @click="lifeCycleControl('Stop')">
              <b-icon icon="stop-fill"></b-icon>
              Stop
            </b-list-group-item>
          </b-list-group>

          <b-card-sub-title class="mb-2 mt-4" v-if="showAddr">
            <b-icon icon="view-stacked"></b-icon> IP and Port
          </b-card-sub-title>
          <b-list-group flush v-if="showAddr">
            <b-list-group-item
              :id="'serverAddr-' + data.id"
              href="#"
              tabindex="0"
              @click="copyToClipboard"
            >
              <b-icon icon="forward-fill"></b-icon>
              {{ data.parameters.ServerAddr }}:{{ data.parameters.ServerPort }}
            </b-list-group-item>
            <b-tooltip
              :target="'serverAddr-' + data.id"
              triggers="hover"
              placement="bottom"
              variant="secondary"
              @hide="restoreText"
            >
              <span v-show="tooltipControl.showCopied"
                ><b-icon icon="check"></b-icon> Copied!</span
              >
              <span v-show="!tooltipControl.showCopied"
                ><b-icon icon="clipboard"></b-icon> Click to copy to
                clipboard</span
              >
            </b-tooltip>
          </b-list-group>

          <b-card-sub-title class="mb-2 mt-4">
            <b-icon icon="cpu"></b-icon> Specs
          </b-card-sub-title>
          <b-list-group flush>
            <b-list-group-item>
              <b-icon icon="people-fill"></b-icon>
              <b> {{ data.parameters.Players }}</b> player slots
            </b-list-group-item>
            <b-list-group-item>
              <b-icon icon="layers-fill"></b-icon>
              <b> {{ data.parameters.RAM }}</b> MB of memory
            </b-list-group-item>
            <b-list-group-item>
              <b-icon icon="server"></b-icon>
              <b> {{ data.parameters.ServerEdition }}</b> /
              {{ data.parameters.ServerVersion }}
            </b-list-group-item>
          </b-list-group>

          <b-card-sub-title class="mb-2 mt-4">
            <b-icon icon="three-dots"></b-icon> More options
          </b-card-sub-title>
          <b-list-group flush>
            <b-list-group-item
              :to="{ name: 'Instance', params: { id: data.id } }"
              v-if="!isSingle"
            >
              <b-icon icon="link"></b-icon>
              Details
            </b-list-group-item>
            <b-list-group-item
              :to="{
                name: 'Subscription',
                params: { id: data.subscriptionId },
              }"
            >
              <b-icon icon="cash"></b-icon>
              Subscription
            </b-list-group-item>
            <b-list-group-item
              href="#"
              class="text-danger"
              tabindex="0"
              :id="'delete-' + data.id"
            >
              <b-icon icon="trash2-fill"></b-icon>
              Delete
            </b-list-group-item>
            <b-popover
              :show.sync="formControl.showDeletePopover"
              :target="'delete-' + data.id"
              triggers="click"
              placement="top"
            >
              <template #title>Are you sure?</template>
              <p>
                This will remove your server and its data. You will receive a
                final bill at the end of the period.
              </p>
              <div>
                <b-button
                  variant="danger"
                  size="sm"
                  class="m-2"
                  @click="formControl.showDeletePopover = false"
                >
                  Take me back
                </b-button>
                <b-button
                  variant="outline-success"
                  size="sm"
                  class="m-2"
                  @click="remove"
                >
                  I understand
                </b-button>
              </div>
            </b-popover>
          </b-list-group>
        </b-card-body>
        <template #footer>
          <div>
            <small>
              Provisioned on host: <b>{{ data.hostName }}</b>
            </small>
          </div>
        </template>
      </b-card>
    </b-overlay>
    <div v-if="isSingle && data.histories">
      <hr class="my-4" />
      <b-overlay :show="formControl.showBusy" rounded="lg" opacity="0.6">
        <template #overlay>
          <div class="d-flex align-items-center">
            <b-spinner small type="grow" variant="secondary"></b-spinner>
            <b-spinner type="grow" variant="dark"></b-spinner>
            <b-spinner small type="grow" variant="secondary"></b-spinner>
            <!-- We add an SR only text for screen readers -->
            <span class="sr-only">Please wait...</span>
          </div>
        </template>
        <b-card
          class="mt-4 mb-2"
          title="Lifecycle history"
          sub-title="Showing only the latest 5 entries."
        >
          <b-table
            striped
            hover
            :fields="formControl.historyFields"
            :items="data.histories"
          >
            <template #cell(timestamp)="data">
              {{ new Date(data.value).toLocaleString() }}
            </template>
          </b-table>
        </b-card>
      </b-overlay>
    </div>
  </div>
</template>

<script>
import * as Sentry from "@sentry/browser";
export default {
  props: {
    isSingle: {
      type: Boolean,
      default: false,
    },
    instance: {
      type: Object,
    },
  },
  data() {
    return {
      data: this.instance,
      resLink: "/instances/" + this.instance.id,
      tooltipControl: {
        showCopied: false,
      },
      formControl: {
        showBusy: false,
        showDeletePopover: false,
        historyFields: [
          {
            key: "timestamp",
            label: "Timestamp",
          },
          {
            key: "state",
            label: "State",
          },
        ],
      },
    };
  },
  computed: {
    headerBg() {
      switch (this.data.state) {
        case "Running":
          return "success";
        case "Stopped":
          return "secondary";
        case "Error":
          return "danger";
        case "Removed":
          return "dark";
        default:
          return "warning";
      }
    },
    showAddr() {
      if (
        (this.data.previousState === "Provisioning" &&
          this.data.state === "Error") ||
        this.data.state === "Provisioning"
      ) {
        return false;
      }
      return true;
    },
  },
  methods: {
    delay(ms) {
      return new Promise((resolve) => {
        setTimeout(resolve, ms);
      });
    },
    async lifeCycleControl(to) {
      this.formControl.showBusy = true;
      await this.delay(1000);
      try {
        let lcResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "POST",
          endpoint: this.resLink,
          body: {
            action: to,
          },
        });
        if (lcResp.status === 202) {
          this.$emit("success", "Lifecycle control successful!");
          this.reload();
        } else {
          let lcJson = await lcResp.json();
          this.$emit("error", lcJson.error + ": " + lcJson.messages.join("; "));
          this.reload();
        }
      } catch (err) {
        Sentry.captureException(err);
        this.$emit("error", "Unable to do instance lifecycle control");
      }
      this.formControl.showBusy = false;
    },
    async remove() {
      this.formControl.showDeletePopover = false;
      this.formControl.showBusy = true;
      await this.delay(1000);
      try {
        let delResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "DELETE",
          endpoint: this.resLink,
        });
        if (delResp.status === 204) {
          this.$emit("success", "Instance removed successfully.");
          this.$emit("remove", this.data.id);
        } else {
          let delJson = await delResp.json();
          this.$emit(
            "error",
            delJson.error + ": " + delJson.messages.join("; ")
          );
        }
      } catch (err) {
        Sentry.captureException(err);
        this.$emit("error", "Unable to remove instance");
      }
      this.formControl.showBusy = false;
    },
    async delayedReload() {
      this.formControl.showBusy = true;
      await this.delay(1000);
      await this.doReload();
      this.formControl.showBusy = false;
    },
    async reload() {
      this.formControl.showBusy = true;
      await this.doReload();
      this.formControl.showBusy = false;
    },
    async doReload() {
      try {
        let instResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "GET",
          endpoint: this.resLink,
        });
        let instJson = await instResp.json();
        if (instResp.status === 200) {
          this.data = instJson.result;
        } else {
          this.$emit(
            "error",
            "Unable to load instance: " +
              instJson.error +
              ": " +
              instJson.messages.join(" - ")
          );
        }
      } catch (err) {
        Sentry.captureException(err);
        this.$emit("error", "Unable to load instance details");
      }
    },
    restoreText() {
      this.tooltipControl.showCopied = false;
    },
    copyToClipboard(evt) {
      evt.preventDefault();
      this.$clipboard(
        this.data.parameters.ServerAddr + ":" + this.data.parameters.ServerPort
      );
      this.tooltipControl.showCopied = true;
    },
  },
  async mounted() {
    if (this.isSingle) {
      await this.doReload();
    }
  },
};
</script>