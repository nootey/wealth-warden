import { createRouter, createWebHistory, RouteRecordRaw } from "vue-router";
import Home from "../components/HelloWorld.vue";


const routes: RouteRecordRaw[] = [
    { path: "/", name: "Home", component: Home },
];

const router = createRouter({
    history: createWebHistory(),
    routes
});

export default router;