<template>
  <div id="app">
    <Nav />

    <b-container>
      <router-view class="mt-4" />
    </b-container>
    <footer class="footer mt-2">
      <b-container>
        <p class="text-center small">
          {{ $store.state.brandName }} is not associated with Mojang or any of
          its products/brands. "Minecraft" is a trademark of Mojang Synergies
          AB.
        </p>
      </b-container>
    </footer>
  </div>
</template>

<script>
import Nav from "./components/Nav.vue";

export default {
  name: "App",
  components: {
    Nav,
  },
};
</script>