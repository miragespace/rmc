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
      <b-button
        type="submit"
        variant="primary"
        :disabled="formControl.submitDisabled"
      >
        <b-spinner small v-show="formControl.showSpinner"></b-spinner>
        {{ formControl.submitButtonText }}
      </b-button>
    </b-form>
  </div>
</template>

<script>
import Alert from "../components/Alert.vue";

const BASE_URL = process.env.VUE_APP_API_ENDPOINT;
const text = {
  submitNormal: "Request Login Link",
  submitting: "Requesting",
};

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
        showSpinner: false,
        submitDisabled: false,
        submitButtonText: "",
      },
    };
  },
  methods: {
    enableSubmit() {
      this.formControl.submitButtonText = text.submitNormal;
      this.formControl.submitDisabled = false;
      this.formControl.showSpinner = false;
    },
    disableSubmit() {
      this.formControl.submitButtonText = text.submitting;
      this.formControl.submitDisabled = true;
      this.formControl.showSpinner = true;
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
          "Unable to request login link, please try again later"
        );
      }
      this.enableSubmit();
    },
  },
  async mounted() {
    this.formControl.submitButtonText = text.submitNormal;
  },
};
</script>