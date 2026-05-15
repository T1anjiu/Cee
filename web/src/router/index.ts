import { createRouter, createWebHistory } from 'vue-router'
import Home from '../views/Home.vue'
import Room from '../views/Room.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/r/:roomId', component: Room },
  ],
})

export default router
