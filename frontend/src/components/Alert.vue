<template>
  <div>
    <b-alert
      :show="dismissCountDown"
      v-bind:class="classes"
      v-bind:style="styles"
      dismissible
      fade
      @dismissed="dismissCountDown = 0"
      @dismiss-count-down="countDownChanged"
      :variant="type"
    >
      {{ message }} <small>({{ dismissCountDown }})</small>
    </b-alert>
    <b-alert v-model="show" dismissible fade :variant="type">
      {{ message }}
    </b-alert>
  </div>
</template>

<script>
export default {
  name: "Alert",
  props: {
    toast: {
      type: Boolean,
      default: false,
    },
    timeout: {
      type: Number,
      default: 10,
    },
  },
  data() {
    return {
      show: false,
      type: "success",
      message: "",
      dismissCountDown: 0,
    };
  },
  computed: {
    classes() {
      if (this.toast) {
        return {
          "position-fixed": true,
          "fixed-bottom": true,
          "m-0": true,
          "rounded-0": true,
        };
      } else {
        return {};
      }
    },
    styles() {
      if (this.toast) {
        return {
          "z-index": 2000,
        };
      } else {
        return {};
      }
    },
  },
  methods: {
    hideAlert() {
      this.dismissCountDown = 0;
      this.show = false;
    },
    showAlert(type, message) {
      this.hideAlert();
      this.type = type;
      this.message = message;
      this.dismissCountDown = this.timeout;
    },
    showDismissable(type, message) {
      this.hideAlert();
      this.type = type;
      this.message = message;
      this.show = true;
    },
    countDownChanged(dismissCountDown) {
      this.dismissCountDown = dismissCountDown;
    },
  },
};
</script>