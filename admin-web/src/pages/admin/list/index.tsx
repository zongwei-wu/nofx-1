import type { ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { Button, message, Popconfirm } from 'antd';
import { useState } from 'react';
import {
  createAdmin,
  deleteUser,
  listAdmins,
  resetUserPassword,
  updateUserRole,
  type AdminUser,
} from '@/services/admin/api';

export default function AdminListPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [resetUser, setResetUser] = useState<AdminUser | null>(null);
  const [tableKey, setTableKey] = useState(0);

  const columns: ProColumns<AdminUser>[] = [
    { title: 'ID', dataIndex: 'id', copyable: true, width: 200, search: false },
    { title: '邮箱', dataIndex: 'email', copyable: true },
    {
      title: '注册时间',
      dataIndex: 'created_at',
      valueType: 'dateTime',
      search: false,
      width: 180,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 280,
      render: (_, record) => [
        <Button
          key="reset"
          type="link"
          size="small"
          onClick={() => setResetUser(record)}
        >
          重置密码
        </Button>,
        <Button
          key="demote"
          type="link"
          size="small"
          danger
          onClick={async () => {
            await updateUserRole(record.id, 'user');
            message.success('已降权为普通用户');
            setTableKey((k) => k + 1);
          }}
        >
          降权
        </Button>,
        <Popconfirm
          key="delete"
          title="确定删除该管理员？"
          onConfirm={async () => {
            await deleteUser(record.id);
            message.success('已删除');
            setTableKey((k) => k + 1);
          }}
        >
          <Button type="link" size="small" danger>
            删除
          </Button>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <PageContainer
      extra={
        <Button type="primary" onClick={() => setCreateOpen(true)}>
          新建管理员
        </Button>
      }
    >
      <ProTable<AdminUser>
        key={tableKey}
        rowKey="id"
        columns={columns}
        request={async (params) => {
          const res = await listAdmins({
            current: params.current,
            pageSize: params.pageSize,
            email: params.email as string | undefined,
          });
          return { data: res.data, total: res.total, success: res.success };
        }}
        search={{ labelWidth: 'auto' }}
        pagination={{ defaultPageSize: 20 }}
      />

      <ModalForm
        title="新建管理员"
        open={createOpen}
        modalProps={{
          destroyOnClose: true,
          onCancel: () => setCreateOpen(false),
        }}
        onFinish={async (values) => {
          await createAdmin(values.email, values.password);
          message.success('管理员已创建');
          setCreateOpen(false);
          setTableKey((k) => k + 1);
          return true;
        }}
      >
        <ProFormText
          name="email"
          label="邮箱"
          rules={[{ required: true, type: 'email' }]}
        />
        <ProFormText.Password
          name="password"
          label="密码"
          rules={[{ required: true, min: 6 }]}
        />
      </ModalForm>

      <ModalForm
        title={`重置密码：${resetUser?.email ?? ''}`}
        open={!!resetUser}
        modalProps={{
          destroyOnClose: true,
          onCancel: () => setResetUser(null),
        }}
        onFinish={async (values) => {
          if (!resetUser) return false;
          await resetUserPassword(resetUser.id, values.password);
          message.success('密码已重置');
          setResetUser(null);
          return true;
        }}
      >
        <ProFormText.Password
          name="password"
          label="新密码"
          rules={[{ required: true, min: 6 }]}
        />
      </ModalForm>
    </PageContainer>
  );
}
