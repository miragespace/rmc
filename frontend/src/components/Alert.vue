<template>
  <div>
    <b-alert
      :show="dismissCountDown"
      v-bind:class="classes"
      v-bind:style="styles"
      dismissible
      @dismissed="dismissCountDown = 0"
      @dismiss-count-down="countDownChanged"
      :variant="type"
    >
      {{ message }} <small>({{ dismissCountDown }})</small>
    </b-alert>
    <b-alert v-model="show" dismissible :variant="type">
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
    showAlert(type, message) {
      this.type = type;
      this.message = message;
      this.dismissCountDown = this.timeout;
    },
    showDismissable(type, message) {
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