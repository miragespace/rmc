import Vue from 'vue'
import Vuex from 'vuex'
import Router from '../router'

Vue.use(Vuex)

const BASE_URL = process.env.VUE_APP_API_ENDPOINT;

export default new Vuex.Store({
    state: {
        brandName: process.env.VUE_APP_BRAND_NAME || 'RMC',
        bearerToken: localStorage.getItem("token") || ''
    },
    getters: {
        isLogin(state) {
            return state.bearerToken !== ''
        }
    },
    mutations: {
        setBearerToken(state, payload) {
            state.bearerToken = payload.token
            localStorage.setItem('token', payload.token)
        }
    },
    actions: {
        logout(context) {
            context.state.bearerToken = ''
            localStorage.removeItem('token')
            Router.push({
                name: 'Home'
            })
        },
        async makeAuthenticatedRequest(context, payload) {
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
