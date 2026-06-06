import type { ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import {
  Button,
  Descriptions,
  Drawer,
  message,
  Popconfirm,
  Table,
  Tabs,
  Tag,
} from 'antd';
import { useState } from 'react';
import {
  getTraderAccount,
  getTraderDetail,
  getTraderPositions,
  listTraders,
  stopTrader,
  type AdminTrader,
  type AdminTraderDetail,
} from '@/services/admin/api';

export default function AITraderListPage() {
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminTraderDetail | null>(null);
  const [account, setAccount] = useState<Record<string, unknown> | null>(null);
  const [positions, setPositions] = useState<Record<string, unknown>[]>([]);
  const [tableKey, setTableKey] = useState(0);

  const openDetail = async (record: AdminTrader) => {
    const res = await getTraderDetail(record.id);
    setDetail(res.data);
    setAccount(null);
    setPositions([]);
    setDetailOpen(true);
    try {
      const acc = await getTraderAccount(record.id);
      setAccount(acc.data);
    } catch {
      setAccount({ error: '交易员未运行或无法获取账户' });
    }
    try {
      const pos = await getTraderPositions(record.id);
      setPositions(pos.data || []);
    } catch {
      setPositions([]);
    }
  };

  const columns: ProColumns<AdminTrader>[] = [
    { title: '用户邮箱', dataIndex: 'email', copyable: true, width: 180, hideInTable: true },
    { title: '用户邮箱', dataIndex: 'user_email', copyable: true, width: 180, search: false },
    { title: '用户 ID', dataIndex: 'user_id', copyable: true, width: 140 },
    { title: '名称', dataIndex: 'name', width: 120 },
    { title: '交易员 ID', dataIndex: 'id', copyable: true, width: 160, search: false },
    {
      title: '运行状态',
      dataIndex: 'is_running',
      width: 100,
      valueType: 'select',
      valueEnum: {
        true: { text: '运行中', status: 'Processing' },
        false: { text: '已停止', status: 'Default' },
      },
      render: (_, r) =>
        r.is_running ? (
          <Tag color="green">运行中</Tag>
        ) : (
          <Tag>已停止</Tag>
        ),
    },
    { title: '模型', dataIndex: 'ai_model_name', search: false, width: 100 },
    { title: '交易所', dataIndex: 'exchange_name', search: false, width: 120 },
    {
      title: '扫描间隔(分)',
      dataIndex: 'scan_interval_minutes',
      search: false,
      width: 110,
    },
    {
      title: '初始资金',
      dataIndex: 'initial_balance',
      search: false,
      width: 100,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      valueType: 'dateTime',
      search: false,
      width: 170,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 180,
      render: (_, record) => [
        <Button key="detail" type="link" size="small" onClick={() => openDetail(record)}>
          详情
        </Button>,
        record.is_running ? (
          <Popconfirm
            key="stop"
            title="确定强制停止该交易员？"
            onConfirm={async () => {
              await stopTrader(record.id);
              message.success('已停止');
              setTableKey((k) => k + 1);
            }}
          >
            <Button type="link" size="small" danger>
              强停
            </Button>
          </Popconfirm>
        ) : null,
      ],
    },
  ];

  return (
    <PageContainer>
      <ProTable<AdminTrader>
        key={tableKey}
        rowKey="id"
        columns={columns}
        scroll={{ x: 1400 }}
        request={async (params) => {
          const res = await listTraders({
            current: params.current,
            pageSize: params.pageSize,
            user_id: params.user_id as string | undefined,
            email: params.user_email as string | undefined,
            name: params.name as string | undefined,
            is_running:
              params.is_running !== undefined
                ? String(params.is_running)
                : undefined,
          });
          return { data: res.data, total: res.total, success: res.success };
        }}
        search={{ labelWidth: 'auto' }}
        pagination={{ defaultPageSize: 20 }}
      />

      <Drawer
        title={detail ? `交易员：${detail.name}` : '交易员详情'}
        width={720}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
      >
        {detail && (
          <Tabs
            items={[
              {
                key: 'config',
                label: '配置',
                children: (
                  <Descriptions column={1} bordered size="small">
                    <Descriptions.Item label="用户">{detail.user_email}</Descriptions.Item>
                    <Descriptions.Item label="ID">{detail.id}</Descriptions.Item>
                    <Descriptions.Item label="模型">
                      {detail.ai_model_name} ({detail.ai_model_provider})
                    </Descriptions.Item>
                    <Descriptions.Item label="交易所">
                      {detail.exchange_name} ({detail.exchange_type})
                    </Descriptions.Item>
                    <Descriptions.Item label="运行状态">
                      {detail.is_running ? '运行中' : '已停止'}
                    </Descriptions.Item>
                    <Descriptions.Item label="扫描间隔">
                      {detail.scan_interval_minutes} 分钟
                    </Descriptions.Item>
                    <Descriptions.Item label="杠杆">
                      BTC/ETH {detail.btc_eth_leverage}x / 山寨 {detail.altcoin_leverage}x
                    </Descriptions.Item>
                    <Descriptions.Item label="交易对">{detail.trading_symbols || '-'}</Descriptions.Item>
                    <Descriptions.Item label="Prompt 模板">
                      {detail.system_prompt_template}
                    </Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'account',
                label: '账户',
                children: account ? (
                  <Descriptions column={1} bordered size="small">
                    {Object.entries(account).map(([k, v]) => (
                      <Descriptions.Item key={k} label={k}>
                        {String(v)}
                      </Descriptions.Item>
                    ))}
                  </Descriptions>
                ) : (
                  '加载中...'
                ),
              },
              {
                key: 'positions',
                label: '持仓',
                children: (
                  <Table
                    size="small"
                    rowKey={(_, i) => String(i)}
                    dataSource={positions}
                    columns={[
                      { title: '字段', dataIndex: 'symbol', render: (_, r) => JSON.stringify(r) },
                    ]}
                    pagination={false}
                  />
                ),
              },
            ]}
          />
        )}
      </Drawer>
    </PageContainer>
  );
}
