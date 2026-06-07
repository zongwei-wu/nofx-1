import type { ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { Button, Drawer, message, Popconfirm, Typography } from 'antd';
import { useState } from 'react';
import {
  createPromptTemplate,
  deletePromptTemplate,
  getPromptTemplate,
  listPromptTemplates,
  updatePromptTemplate,
  type PromptTemplateItem,
} from '@/services/admin/api';

export default function PromptTemplateListPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [editItem, setEditItem] = useState<PromptTemplateItem | null>(null);
  const [viewItem, setViewItem] = useState<PromptTemplateItem | null>(null);
  const [tableKey, setTableKey] = useState(0);

  const openEdit = async (record: PromptTemplateItem) => {
    const res = await getPromptTemplate(record.name);
    setEditItem(res.data);
  };

  const openView = async (record: PromptTemplateItem) => {
    const res = await getPromptTemplate(record.name);
    setViewItem(res.data);
  };

  const columns: ProColumns<PromptTemplateItem>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      copyable: true,
      width: 140,
    },
    {
      title: '描述',
      dataIndex: 'description',
      search: false,
      ellipsis: true,
      width: 180,
      render: (_, r) => r.description || '-',
    },
    {
      title: '内容预览',
      dataIndex: 'content_preview',
      search: false,
      ellipsis: true,
    },
    {
      title: '字数',
      dataIndex: 'content_length',
      search: false,
      width: 80,
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      valueType: 'dateTime',
      search: false,
      width: 170,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      render: (_, record) => [
        <Button
          key="view"
          type="link"
          size="small"
          onClick={() => openView(record)}
        >
          查看
        </Button>,
        <Button
          key="edit"
          type="link"
          size="small"
          onClick={() => openEdit(record)}
        >
          编辑
        </Button>,
        <Popconfirm
          key="delete"
          title="确定删除该模板？"
          onConfirm={async () => {
            try {
              await deletePromptTemplate(record.name);
              message.success('已删除');
              setTableKey((k) => k + 1);
            } catch (e: unknown) {
              const err = e as { data?: { error?: string }; message?: string };
              message.error(err?.data?.error || err?.message || '删除失败');
            }
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
          新建模板
        </Button>
      }
    >
      <ProTable<PromptTemplateItem>
        key={tableKey}
        rowKey="name"
        columns={columns}
        request={async (params) => {
          const res = await listPromptTemplates({
            current: params.current,
            pageSize: params.pageSize,
            name: params.name as string | undefined,
          });
          return { data: res.data, total: res.total, success: res.success };
        }}
        search={{ labelWidth: 'auto' }}
        pagination={{ defaultPageSize: 20 }}
      />

      <ModalForm
        title="新建 Prompt 模板"
        open={createOpen}
        modalProps={{
          destroyOnClose: true,
          onCancel: () => setCreateOpen(false),
        }}
        onFinish={async (values) => {
          await createPromptTemplate({
            name: values.name,
            content: values.content,
            description: values.description,
          });
          message.success('模板已创建');
          setCreateOpen(false);
          setTableKey((k) => k + 1);
          return true;
        }}
      >
        <ProFormText
          name="name"
          label="名称"
          rules={[
            { required: true },
            {
              pattern: /^[a-zA-Z0-9_-]+$/,
              message: '仅允许字母、数字、下划线和连字符',
            },
          ]}
          extra="创建后不可修改，与交易员 system_prompt_template 字段对应"
        />
        <ProFormText name="description" label="描述" />
        <ProFormTextArea
          name="content"
          label="模板内容"
          rules={[{ required: true }]}
          fieldProps={{ rows: 12, style: { fontFamily: 'monospace' } }}
        />
      </ModalForm>

      <ModalForm
        title={`编辑模板：${editItem?.name ?? ''}`}
        open={!!editItem}
        key={editItem?.name ?? 'edit'}
        initialValues={{
          description: editItem?.description,
          content: editItem?.content,
        }}
        modalProps={{
          destroyOnClose: true,
          onCancel: () => setEditItem(null),
        }}
        onFinish={async (values) => {
          if (!editItem) return false;
          await updatePromptTemplate(editItem.name, {
            content: values.content,
            description: values.description,
          });
          message.success('模板已更新');
          setEditItem(null);
          setTableKey((k) => k + 1);
          return true;
        }}
      >
        <ProFormTextArea
          name="content"
          label="模板内容"
          rules={[{ required: true }]}
          fieldProps={{ rows: 14, style: { fontFamily: 'monospace' } }}
        />
        <ProFormText name="description" label="描述" />
      </ModalForm>

      <Drawer
        title={viewItem ? `模板：${viewItem.name}` : '模板详情'}
        width={720}
        open={!!viewItem}
        onClose={() => setViewItem(null)}
      >
        {viewItem && (
          <>
            <Typography.Paragraph type="secondary">
              {viewItem.description || '无描述'}
            </Typography.Paragraph>
            <Typography.Paragraph
              style={{
                whiteSpace: 'pre-wrap',
                fontFamily: 'monospace',
                fontSize: 13,
                maxHeight: '70vh',
                overflow: 'auto',
              }}
            >
              {viewItem.content}
            </Typography.Paragraph>
          </>
        )}
      </Drawer>
    </PageContainer>
  );
}
