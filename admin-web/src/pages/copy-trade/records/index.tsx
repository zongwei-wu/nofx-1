import type { ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import {
  listCopyTradeRecords,
  type CopyTradeRecord,
} from '@/services/admin/api';

const columns: ProColumns<CopyTradeRecord>[] = [
  { title: 'ID', dataIndex: 'id', width: 70, search: false },
  { title: '用户 ID', dataIndex: 'user_id', copyable: true, width: 140 },
  { title: '带单员', dataIndex: 'nickname', search: false, width: 120 },
  { title: '组合 ID', dataIndex: 'portfolio_id', search: false, width: 120 },
  {
    title: '交易对',
    dataIndex: 'symbol',
    width: 110,
  },
  {
    title: '状态',
    dataIndex: 'status',
    valueEnum: {
      OPEN: { text: '持仓中', status: 'Processing' },
      CLOSED: { text: '已平仓', status: 'Success' },
      FAILED: { text: '失败', status: 'Error' },
    },
    width: 100,
  },
  { title: '方向', dataIndex: 'side', search: false, width: 70 },
  { title: '持仓侧', dataIndex: 'position_side', search: false, width: 80 },
  {
    title: '数量',
    dataIndex: 'executed_qty',
    search: false,
    width: 90,
  },
  {
    title: '均价',
    dataIndex: 'avg_price',
    search: false,
    width: 100,
  },
  {
    title: '盈亏',
    dataIndex: 'total_pnl',
    search: false,
    width: 100,
    render: (_, r) => (
      <span style={{ color: r.total_pnl >= 0 ? '#3f8600' : '#cf1322' }}>
        {r.total_pnl?.toFixed?.(2) ?? r.total_pnl}
      </span>
    ),
  },
  {
    title: '跟单时间',
    dataIndex: 'copy_time',
    valueType: 'dateTime',
    search: false,
    width: 170,
  },
  {
    title: '错误信息',
    dataIndex: 'error_message',
    search: false,
    ellipsis: true,
  },
];

export default function CopyTradeRecordsPage() {
  return (
    <PageContainer>
      <ProTable<CopyTradeRecord>
        rowKey="id"
        columns={columns}
        scroll={{ x: 1400 }}
        request={async (params) => {
          const res = await listCopyTradeRecords({
            current: params.current,
            pageSize: params.pageSize,
            user_id: params.user_id as string | undefined,
            status: params.status as string | undefined,
            symbol: params.symbol as string | undefined,
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
    </PageContainer>
  );
}
