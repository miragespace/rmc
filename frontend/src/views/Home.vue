<template>
  <div class="home">
    <b-jumbotron bg-variant="light">
      <template #header>hello fellow miners</template>

      <template #lead>
        Rent a Minecraft Server provides you with the most affordable Minecraft
        server options. Only playing on the weekends? Only pay for the weekend!
      </template>

      <hr class="my-4" />

      <b-card-sub-title class="mb-2 mt-2">
        <b-icon icon="star-fill"></b-icon> Features
      </b-card-sub-title>
      <b-list-group flush>
        <b-list-group-item>
          <b-icon icon="play-fill"></b-icon>
          Easy to use interface
        </b-list-group-item>
        <b-list-group-item>
          <b-icon icon="credit-card-fill"></b-icon>
          Pay for only what you use
        </b-list-group-item>
        <b-list-group-item>
          <b-icon icon="cpu-fill"></b-icon>
          Intel® Xeon® servers for best Minecraft experience
        </b-list-group-item>
        <b-list-group-item>
          <b-icon icon="arrow-down-square-fill"></b-icon>
          Upload/Save your adventure
          <b-badge>Coming soon</b-badge>
        </b-list-group-item>
        <b-list-group-item>
          <b-icon icon="plug-fill"></b-icon>
          Expand your world with mods and plugins
          <b-badge>Coming soon</b-badge>
        </b-list-group-item>
      </b-list-group>
    </b-jumbotron>
  </div>
</template>

<script>
export default {
  name: "Home",
};
</script>