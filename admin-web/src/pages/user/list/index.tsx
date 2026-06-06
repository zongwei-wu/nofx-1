import type { ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormSelect,
  ProTable,
} from '@ant-design/pro-components';
import { Button, message } from 'antd';
import { useEffect, useState } from 'react';
import {
  listPlans,
  listUsers,
  updateUserPlan,
  updateUserRole,
  type AdminUser,
  type PlanInfo,
} from '@/services/admin/api';

const planLabels: Record<string, string> = {
  basic: '基础版',
  standard: '标准版',
  pro: '专业版',
  vip: 'VIP',
};

export default function UserListPage() {
  const [plans, setPlans] = useState<PlanInfo[]>([]);
  const [editingUser, setEditingUser] = useState<AdminUser | null>(null);
  const [tableKey, setTableKey] = useState(0);

  useEffect(() => {
    listPlans().then((res) => {
      if (res.data) {
        setPlans(res.data);
      }
    });
  }, []);

  const columns: ProColumns<AdminUser>[] = [
    { title: 'ID', dataIndex: 'id', copyable: true, width: 180 },
    { title: '邮箱', dataIndex: 'email', copyable: true },
    {
      title: '角色',
      dataIndex: 'role',
      valueEnum: {
        admin: { text: '管理员', status: 'Success' },
        user: { text: '用户', status: 'Default' },
      },
      width: 100,
    },
    {
      title: '套餐',
      dataIndex: 'plan',
      width: 120,
      render: (_, record) => planLabels[record.plan] || record.plan,
    },
    {
      title: 'OTP 已验证',
      dataIndex: 'otp_verified',
      width: 120,
      render: (_, record) => (record.otp_verified ? '是' : '否'),
    },
    {
      title: '注册时间',
      dataIndex: 'created_at',
      valueType: 'dateTime',
      width: 180,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 180,
      render: (_, record) =>
        record.role === 'admin'
          ? []
          : [
              <Button
                key="plan"
                type="link"
                size="small"
                onClick={() => setEditingUser(record)}
              >
                分配套餐
              </Button>,
              <Button
                key="promote"
                type="link"
                size="small"
                onClick={async () => {
                  await updateUserRole(record.id, 'admin');
                  message.success('已设为管理员');
                  setTableKey((k) => k + 1);
                }}
              >
                设为管理员
              </Button>,
            ],
    },
  ];

  return (
    <PageContainer>
      <ProTable<AdminUser>
        key={tableKey}
        rowKey="id"
        columns={columns}
        request={async (params) => {
          const res = await listUsers({
            current: params.current,
            pageSize: params.pageSize,
            email: params.email as string | undefined,
          });
          return {
            data: res.data,
            total: res.total,
            success: res.success,
          };
        }}
        search={{ labelWidth: 'auto' }}
        pagination={{ defaultPageSize: 20 }}
      />

      <ModalForm
        title={`分配套餐：${editingUser?.email ?? ''}`}
        open={!!editingUser}
        modalProps={{ destroyOnClose: true, onCancel: () => setEditingUser(null) }}
        initialValues={{ plan: editingUser?.plan }}
        onFinish={async (values) => {
          if (!editingUser) return false;
          await updateUserPlan(editingUser.id, values.plan);
          message.success('套餐已更新');
          setEditingUser(null);
          setTableKey((k) => k + 1);
          return true;
        }}
      >
        <ProFormSelect
          name="plan"
          label="套餐"
          rules={[{ required: true, message: '请选择套餐' }]}
          options={plans.map((p) => ({
            label: `${p.name}（${p.features.join(', ')}）`,
            value: p.id,
          }))}
        />
      </ModalForm>
    </PageContainer>
  );
}
