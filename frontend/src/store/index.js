import Vue from 'vue'
import Vuex from 'vuex'
import Router from '../router'
import { getHeader } from './helper'

Vue.use(Vuex)

const BASE_URL = process.env.VUE_APP_API_ENDPOINT;

export default new Vuex.Store({
    state: {
        brandName: process.env.VUE_APP_BRAND_NAME || 'RMC',
        accessToken: localStorage.getItem("accessToken") || '',
        refreshToken: localStorage.getItem("refreshToken") || ''
    },
    getters: {
        isLogin(state) {
            return state.accessToken !== ''
        }
    },
    mutations: {
        setTokens(state, payload) {
            state.accessToken = payload.accessToken
            state.refreshToken = payload.refreshToken
            localStorage.setItem('accessToken', payload.accessToken)
            localStorage.setItem('refreshToken', payload.refreshToken)
        },
        updateAccessToken(state, payload) {
            state.accessToken = payload.accessToken
            localStorage.setItem('accessToken', payload.accessToken)
        }
    },
    actions: {
        logout(context) {
            context.state.accessToken = ''
            context.state.refreshToken = ''
            localStorage.removeItem('accessToken')
            localStorage.removeItem('refreshToken')
            Router.push({
                name: 'Home'
            })
        },
        async requestTokens(context, payload) {
            let req = {
                method: 'POST',
                mode: 'cors',
                credentials: "include",
                headers: {
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({
                    uid: payload.uid,
                    token: payload.token
                })
            }
            let resp = await fetch(BASE_URL + '/auth/requestTokens', req)
            let json = await resp.json()
            if (resp.status === 200) {
                context.commit({
                    type: "setTokens",
                    accessToken: json.result.accessToken,
                    refreshToken: json.result.refreshToken,
                })
            } else {
                throw {
                    apiError: true,
                    err: json.error,
                    message: json.messages.join(' - ')
                }
            }
        },
        async refreshSession(context) {
            let req = {
                method: 'POST',
                mode: "cors",
                credentials: "include",
                headers: {
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({
                    refreshToken: context.state.refreshToken
                })
            }
            let resp = await fetch(BASE_URL + '/auth/refresh', req)
            let json = await resp.json()
            if (resp.status === 200) {
                context.commit({
                    type: "updateAccessToken",
                    accessToken: json.result.accessToken
                })
            } else {
                throw {
                    apiError: true,
                    error: json.error,
                    message: json.messages.join(' - ')
                }
            }
        },
        async makeAuthenticatedRequest(context, payload) {
            let resp = await fetch(BASE_URL + payload.endpoint, getHeader(context, payload));
            if (resp.status === 401) {
                console.log("expired access token")
                try {
                    await context.dispatch({
                        type: 'refreshSession'
                    })
                    console.log("access token refreshed")
                    return fetch(BASE_URL + payload.endpoint, getHeader(context, payload))
                } catch (err) {
                    console.log("access token refresh failed")
                    return resp
                }
            } else {
                return resp
            }
        }
    },
    modules: {
    }
})
