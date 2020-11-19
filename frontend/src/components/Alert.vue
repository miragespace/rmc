<template>
  <div>
    <b-alert
      :show="dismissCountDown"
      dismissible
      @dismissed="dismissCountDown = 0"
      :variant="type"
    >
      {{ message }}
    </b-alert>
    <b-alert v-model="show" dismissible :variant="type">
      {{ message }}
    </b-alert>
  </div>
</template>

<script>
export default {
  name: "Alert",
  data() {
    return {
      show: false,
      type: "success",
      message: "",
      dismissCountDown: 0,
      dismissSecs: 10,
    };
  },
  methods: {
    showAlert(type, message) {
      this.type = type;
      this.message = message;
      this.dismissCountDown = this.dismissSecs;
    },
    showDismissable(type, message) {
      this.type = type;
      this.message = message;
      this.show = true;
    },
  },
};
</script>