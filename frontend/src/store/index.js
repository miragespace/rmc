import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex)

const BASE_URL = process.env.VUE_APP_API_ENDPOINT;
const DEFINED_TOKEN = process.env.VUE_APP_BEARER_TOKEN || '';

export default new Vuex.Store({
    state: {
        bearerToken: DEFINED_TOKEN
    },
    getters: {
        isLogin(state) {
            return state.bearerToken !== ''
        }
    },
    mutations: {
        setBearerToken(state, payload) {
            console.log(payload)
            state.bearerToken = payload.token
        }
    },
    actions: {
        async makeAuthenticatedRequest(context, payload) {
            console.log(context)
            let method = payload.method || "GET"
            let req = {
                method: method,
                mode: "cors",
                credentials: "include",
                headers: {
                    "Content-Type": "application/json",
                    "Authorization": "Bearer " + context.state.bearerToken
                },
            };
            if (method != 'GET') {
                req.body = JSON.stringify(payload.body)
            }
            return fetch(BASE_URL + payload.endpoint, req);
        }
    },
    modules: {
    }
})
