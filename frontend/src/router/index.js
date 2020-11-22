import Vue from 'vue'
import VueRouter from 'vue-router'
import Store from '../store'
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
        component: () => import('../views/Instance.vue'),
        meta: {
            requireAuthentication: true
        }
    },
    {
        path: '/instances',
        name: 'Instances',
        component: () => import('../views/Instances.vue'),
        meta: {
            requireAuthentication: true
        }
    },
    {
        path: '/subscriptions/plans',
        name: 'Plans',
        component: () => import('../views/Plans.vue'),
        meta: {
            requireAuthentication: true
        }
    },
    {
        path: '/subscriptions/:id',
        name: 'Subscription',
        component: () => import('../views/Subscription.vue'),
        meta: {
            requireAuthentication: true
        }
    },
    {
        path: '/subscriptions',
        name: 'Subscriptions',
        component: () => import('../views/Subscriptions.vue'),
        meta: {
            requireAuthentication: true
        }
    }
]

const router = new VueRouter({
    mode: 'history',
    base: process.env.BASE_URL,
    routes
})

router.beforeEach((to, from, next) => {
    if (to.meta && to.meta.requireAuthentication && !Store.getters.isLogin) {
        // TODO: revisit this
        router.app.$bvToast.toast("Please login first!", {
            title: "Authentication required",
            autoHideDelay: 5000,
            variant: "danger",
            solid: true
        })
        next({ name: 'Login' })
    } else {
        document.title = Store.state.brandName + ' / ' + to.name
        next()
    }
})

export default router
