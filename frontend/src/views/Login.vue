<template>
  <div class="login">
    <Alert ref="alert" />
    <b-form @submit="requestLogin">
      <b-form-group
        id="input-email-group"
        label="Email address:"
        label-for="email-input"
        description="This will be tied to your acccount"
      >
        <b-form-input
          id="input-email"
          v-model="form.email"
          type="email"
          required
          placeholder="hello@me.com"
        ></b-form-input>
      </b-form-group>
      <b-overlay
        :show="formControl.submitDisabled"
        rounded
        opacity="0.6"
        spinner-small
        spinner-variant="primary"
        class="d-inline-block"
      >
        <b-button
          type="submit"
          variant="primary"
          :disabled="formControl.submitDisabled"
        >
          Request Login Link
        </b-button>
      </b-overlay>
    </b-form>
  </div>
</template>

<script>
import Alert from "../components/Alert.vue";

const BASE_URL = process.env.VUE_APP_API_ENDPOINT;

export default {
  name: "Login",
  components: {
    Alert,
  },
  data() {
    return {
      form: {
        email: "",
      },
      formControl: {
        submitDisabled: false,
      },
    };
  },
  methods: {
    enableSubmit() {
      this.formControl.submitDisabled = false;
    },
    disableSubmit() {
      this.formControl.submitDisabled = true;
    },
    async requestLogin(evt) {
      evt.preventDefault();
      this.disableSubmit();
      try {
        let req = {
          method: "POST",
          mode: "cors",
          credentials: "include",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(this.form),
        };
        let resp = await fetch(BASE_URL + "/customers", req);

        if (resp.status == 204) {
          this.$refs.alert.showAlert(
            "success",
            "Success! Check your email for the magic link to log in."
          );
        } else {
          let json = await resp.json();
          this.$refs.alert.showAlert("danger", json.error);
        }
      } catch (err) {
        this.$refs.alert.showAlert(
          "danger",
          "An unexpected error has occured: " + err.message
        );
      }
      this.enableSubmit();
    },
  },
};
</script>