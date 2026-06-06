import { LogoutOutlined } from '@ant-design/icons';
import type { RequestConfig, RunTimeLayoutConfig } from '@umijs/max';
import { history } from '@umijs/max';
import { Button, message } from 'antd';
import { adminMe } from './services/admin/api';

const TOKEN_KEY = 'admin_token';

export async function getInitialState(): Promise<{
  currentUser?: API.CurrentUser;
}> {
  const token = localStorage.getItem(TOKEN_KEY);
  if (!token) {
    return { currentUser: undefined };
  }
  try {
    const me = await adminMe();
    return {
      currentUser: {
        name: me.email,
        email: me.email,
        role: me.role,
        access: me.role,
      },
    };
  } catch {
    localStorage.removeItem(TOKEN_KEY);
    return { currentUser: undefined };
  }
}

export const layout: RunTimeLayoutConfig = ({ initialState }) => {
  return {
    onPageChange: () => {
      const { location } = history;
      if (!initialState?.currentUser && location.pathname !== '/user/login') {
        history.push('/user/login');
      }
    },
    avatarProps: {
      title: initialState?.currentUser?.email,
    },
    actionsRender: () => [
      <Button
        key="logout"
        type="link"
        icon={<LogoutOutlined />}
        onClick={() => {
          localStorage.removeItem(TOKEN_KEY);
          history.push('/user/login');
        }}
      >
        退出
      </Button>,
    ],
    menu: { locale: false },
  };
};

export const request: RequestConfig = {
  timeout: 30000,
  requestInterceptors: [
    (url, options) => {
      const token = localStorage.getItem(TOKEN_KEY);
      const headers = { ...(options.headers as Record<string, string>) };
      if (token) {
        headers.Authorization = `Bearer ${token}`;
      }
      return { url, options: { ...options, headers } };
    },
  ],
  errorConfig: {
    errorHandler: (error: { response?: { status?: number } }) => {
      if (error?.response?.status === 401) {
        localStorage.removeItem(TOKEN_KEY);
        history.push('/user/login');
        message.error('登录已过期，请重新登录');
      } else if (error?.response?.status === 403) {
        message.error('无管理员权限');
      }
    },
  },
};
