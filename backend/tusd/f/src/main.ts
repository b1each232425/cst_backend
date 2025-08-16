import { createApp } from 'vue';
import { createPinia } from 'pinia';

import 'ant-design-vue/dist/antd.css';
import { Button, message, Input } from 'ant-design-vue';

import App from './App.vue';
import router from './router';

import './assets/main.css';

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(Button);
app.use(Input);
app.config.globalProperties.$message = message;
app.mount('#app');
