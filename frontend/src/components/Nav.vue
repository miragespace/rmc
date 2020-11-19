<template>
  <div>
    <b-navbar toggleable="lg" type="dark" variant="primary">
      <b-container>
        <b-navbar-brand :to="{ name: 'Home' }"
          >Rent a Minecraft Server</b-navbar-brand
        >
        <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>
        <b-collapse id="nav-collapse" is-nav>
          <!-- Right aligned nav items -->
          <b-navbar-nav class="ml-auto" right>
            <b-nav-item :to="{ name: 'Login' }">Login</b-nav-item>
          </b-navbar-nav>
          <!-- <b-navbar-nav class="ml-auto">
            <b-nav-item-dropdown right>
              <template slot="button-content"><em>User</em></template>
              <b-dropdown-item href="#">Profile</b-dropdown-item>
              <b-dropdown-item href="#">Sign Out</b-dropdown-item>
            </b-nav-item-dropdown>
          </b-navbar-nav> -->
        </b-collapse>
      </b-container>
    </b-navbar>
  </div>
</template>

<script>
export default {
  data() {
    return {
      meal: "",
    };
  },
  methods: {
    getMeal() {},
  },
};
</script>