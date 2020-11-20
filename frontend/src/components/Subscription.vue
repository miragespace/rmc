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
        class="mt-4 mb-2"
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
            title="Refresh subscription status"
          >
            {{ data.state }}
            <b-icon icon="arrow-repeat"></b-icon>
          </b-button>
        </template>
        <b-card-body>
          <b-list-group flush>
            <b-list-group-item>
              <b-icon icon="play-fill"></b-icon>
              Current Period Start:
              <b>{{ new Date(data.periodStart).toLocaleDateString() }}</b>
            </b-list-group-item>
            <b-list-group-item>
              <b-icon icon="stop-fill"></b-icon>
              Current Period End:
              <b>{{ new Date(data.periodEnd).toLocaleDateString() }}</b>
            </b-list-group-item>
          </b-list-group>

          <Plan :plan="data.plan" :showCreate="false" />

          <b-list-group flush>
            <b-list-group-item
              :to="{ name: 'Subscription', params: { id: data.id } }"
              v-if="!isSingle"
            >
              <b-icon icon="link"></b-icon>
              Details
            </b-list-group-item>
          </b-list-group>
        </b-card-body>
        <template #footer>
          <div>
            <small>
              Created on {{ new Date(data.createdAt).toLocaleString() }}
            </small>
          </div>
        </template>
      </b-card>
    </b-overlay>
    <div v-if="isSingle">
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
          title="Usages"
          sub-title="Showing the variable usage for this subscription"
        >
          <b-table striped hover :fields="usages.fields" :items="usages.data">
            <template #cell(startDate)="data">
              {{ new Date(data.value).toLocaleDateString() }}
            </template>
            <template #cell(endDate)="data">
              {{ new Date(data.value).toLocaleDateString() }}
            </template>
            <template #cell(unit)> second </template>
          </b-table>
        </b-card>
      </b-overlay>
    </div>
  </div>
</template>

<script>
import Plan from "./Plan.vue";
import * as Sentry from "@sentry/browser";
export default {
  components: {
    Plan,
  },
  props: {
    isSingle: {
      type: Boolean,
      default: false,
    },
    subscription: {
      type: Object,
    },
  },
  data() {
    return {
      data: this.subscription,
      resLink: "/subscriptions/" + this.subscription.id,
      formControl: {
        showBusy: false,
      },
      usages: {
        fields: [
          {
            key: "subscriptionItem.part.name",
            label: "Type",
          },
          {
            key: "startDate",
            label: "Start",
          },
          {
            key: "endDate",
            label: "End",
          },
          {
            key: "aggregateTotal",
            label: "Aggregate Total",
          },
          "unit",
        ],
        data: [],
      },
    };
  },
  computed: {
    headerBg() {
      switch (this.data.state) {
        case "Active":
          return "success";
        case "Inactive":
          return "secondary";
        case "Cancelled":
        case "Overdue":
          return "danger";
        default:
          return "warning";
      }
    },
  },
  methods: {
    delay(ms) {
      return new Promise((resolve) => {
        setTimeout(resolve, ms);
      });
    },
    async getUsage() {
      try {
        let subResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "GET",
          endpoint: this.resLink + "/usages",
        });
        if (subResp.status === 200) {
          let subJson = await subResp.json();
          this.usages.data = subJson.result;
        }
      } catch (err) {
        Sentry.captureException(err);
        this.$emit("error", "Unable to load subscription usages");
      }
    },
    async delayedReload() {
      this.formControl.showBusy = true;
      await this.delay(1000);
      await this.doReload();
      this.formControl.showBusy = false;
    },
    async doReload() {
      try {
        let subResp = await this.$store.dispatch({
          type: "makeAuthenticatedRequest",
          method: "GET",
          endpoint: this.resLink,
        });
        if (subResp.status === 200) {
          let subJson = await subResp.json();
          this.data = subJson.result;
          await this.getUsage();
        }
      } catch (err) {
        Sentry.captureException(err);
        this.$emit("error", "Unable to load subscription");
      }
    },
  },
  async mounted() {
    if (this.isSingle) {
      await this.doReload();
    }
  },
};
</script>