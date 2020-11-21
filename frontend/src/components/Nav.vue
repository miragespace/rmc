<template>
  <div>
    <b-navbar toggleable="lg" type="dark" variant="primary">
      <b-container>
        <b-navbar-brand :to="{ name: 'Home' }">
          Rent a Minecraft Server
        </b-navbar-brand>
        <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>
        <b-collapse id="nav-collapse" is-nav>
          <!-- Right aligned nav items -->
          <b-navbar-nav class="ml-auto" right v-if="!$store.getters.isLogin">
            <b-nav-item
              :to="{ name: 'Login' }"
              :active="$route.name == 'Login'"
            >
              Login/Register
            </b-nav-item>
          </b-navbar-nav>

          <b-navbar-nav class="ml-auto" right v-if="$store.getters.isLogin">
            <b-nav-item
              v-for="p in loggedInNav"
              v-bind:key="p"
              :to="{ name: p }"
              :active="$route.name == p"
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
      loggedInNav: ["Instances", "Plans", "Subscriptions"],
    };
  },
  methods: {
    logout() {
      this.$store.dispatch("logout");
    },
  },
};
</script>