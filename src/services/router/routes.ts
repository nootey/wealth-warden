import HomeView from "../../components/HomeView.vue";
import AboutView from "../../components/AboutView.vue";

const routes = [
    {
        path: '/',
        name: 'Home',
        component: HomeView
    },
    {
        path: '/about',
        name: 'About',
        component: AboutView
    }
];

export default routes