<template>
  <div class="login">
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

      <b-button type="submit" variant="primary">Request Login Link</b-button>
    </b-form>
  </div>
</template>

<script>
export default {
  name: "Login",
  data() {
    return {
      form: {
        email: "",
      },
    };
  },
  methods: {
    requestLogin(evt) {
      evt.preventDefault();
    },
  },
};
</script>