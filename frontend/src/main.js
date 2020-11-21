import '@babel/polyfill'
import 'mutationobserver-shim'
import "normalize.css"
import Vue from 'vue'
import './plugins/bootstrap-vue'
import './plugins/clipboard'
import App from './App.vue'
import router from './router'
import store from './store'
import * as Sentry from "@sentry/browser";
import { Vue as VueIntegration } from "@sentry/integrations";

Vue.config.productionTip = false

Sentry.init({
    release: process.env.VUE_APP_GIT_COMMIT,
    dsn: process.env.VUE_APP_SENTRY_DSN,
    integrations: [new VueIntegration({ Vue, attachProps: true, logErrors: true })],
});

new Vue({
    router,
    store,
    render: h => h(App)
}).$mount('#app')
