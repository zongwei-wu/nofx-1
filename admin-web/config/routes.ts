export default [
  {
    path: '/user',
    layout: false,
    routes: [
      {
        path: '/user/login',
        component: './user/login',
      },
    ],
  },
  {
    path: '/user/list',
    name: '用户管理',
    icon: 'user',
    access: 'canAdmin',
    component: './user/list',
  },
  {
    path: '/copy-trade/records',
    name: '跟单记录',
    icon: 'table',
    access: 'canAdmin',
    component: './copy-trade/records',
  },
  {
    path: '/system/config',
    name: '系统配置',
    icon: 'setting',
    access: 'canAdmin',
    component: './system/config',
  },
  {
    path: '/',
    redirect: '/user/list',
  },
  {
    path: '*',
    layout: false,
    component: './user/login',
  },
];
