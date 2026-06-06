import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { LoginForm, ProFormText } from '@ant-design/pro-components';
import { history, useModel } from '@umijs/max';
import { message } from 'antd';
import { login } from '@/services/admin/api';

const TOKEN_KEY = 'admin_token';

export default function LoginPage() {
  const { setInitialState } = useModel('@@initialState');

  return (
    <div
      style={{
        display: 'flex',
        height: '100vh',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f2f5',
      }}
    >
      <LoginForm
        title="NOFX 管理后台"
        subTitle="使用管理员账号登录"
        onFinish={async (values) => {
          try {
            const res = await login(values.email, values.password);
            if (!res.token) {
              message.error('登录失败');
              return false;
            }
            if (res.role && res.role !== 'admin') {
              message.error('该账号不是管理员');
              return false;
            }
            localStorage.setItem(TOKEN_KEY, res.token);
            await setInitialState({
              currentUser: {
                name: res.email,
                email: res.email,
                role: res.role || 'admin',
                access: res.role || 'admin',
              },
            });
            message.success('登录成功');
            history.push('/user/list');
            return true;
          } catch (err: unknown) {
            const e = err as { data?: { error?: string } };
            message.error(e?.data?.error || '登录失败');
            return false;
          }
        }}
      >
        <ProFormText
          name="email"
          fieldProps={{ size: 'large', prefix: <UserOutlined /> }}
          placeholder="admin@localhost"
          rules={[{ required: true, message: '请输入邮箱' }]}
          initialValue="admin@localhost"
        />
        <ProFormText.Password
          name="password"
          fieldProps={{ size: 'large', prefix: <LockOutlined /> }}
          placeholder="密码"
          rules={[{ required: true, message: '请输入密码' }]}
        />
      </LoginForm>
    </div>
  );
}
