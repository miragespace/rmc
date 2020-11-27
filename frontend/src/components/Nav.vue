<template>
  <div>
    <b-navbar toggleable="lg" type="dark" variant="dark">
      <b-container>
        <b-navbar-brand :to="{ name: 'Home' }">
          {{ brand }}
        </b-navbar-brand>
        <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>
        <b-collapse id="nav-collapse" is-nav>
          <!-- Right aligned nav items -->
          <b-navbar-nav class="ml-auto" right v-if="!isLogin">
            <b-nav-item :to="{ name: 'Login' }" :active="routeName == 'Login'">
              Login/Register
            </b-nav-item>
          </b-navbar-nav>

          <b-navbar-nav class="ml-auto" right v-if="isLogin">
            <b-nav-item
              v-for="p in loggedInNav"
              v-bind:key="p"
              :to="{ name: p }"
              :active="routeName == p"
            >
              {{ p }}
            </b-nav-item>
            <b-nav-item @click="logout"> Logout </b-nav-item>
          </b-navbar-nav>
        </b-collapse>
      </b-container>
    </b-navbar>
  </div>
</template>

<script>
export default {
  data() {
    return {
      brand: this.$store.state.brandName,
      loggedInNav: ["Instances", "Plans", "Subscriptions"],
    };
  },
  computed: {
    isLogin() {
      return this.$store.getters.isLogin;
    },
    routeName() {
      return this.$route.name;
    },
  },
  methods: {
    logout() {
      this.$store.dispatch("logout");
    },
  },
};
</script>