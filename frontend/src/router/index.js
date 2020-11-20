import Vue from 'vue'
import VueRouter from 'vue-router'
import Home from '../views/Home.vue'

Vue.use(VueRouter)

const routes = [
    {
        path: '/',
        name: 'Home',
        component: Home
    },
    {
        path: '/login/:uid/:token',
        name: 'VerifyToken',
        component: () => import('../views/VerifyToken.vue')
    },
    {
        path: '/login',
        name: 'Login',
        component: () => import('../views/Login.vue')
    },
    {
        path: '/instances/:id',
        name: 'Instance',
        component: () => import('../views/Instance.vue')
    },
    {
        path: '/instances',
        name: 'Instances',
        component: () => import('../views/Instances.vue')
    },
    {
        path: '/subscriptions/plans',
        name: 'Plans',
        component: () => import('../views/Plans.vue')
    },
    {
        path: '/subscriptions/:id',
        name: 'Subscription',
        component: () => import('../views/Subscription.vue')
    },
    {
        path: '/subscriptions',
        name: 'Subscriptions',
        component: () => import('../views/Subscriptions.vue')
    }
]

const router = new VueRouter({
    mode: 'history',
    base: process.env.BASE_URL,
    routes
})

export default router
