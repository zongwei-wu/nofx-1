import {
  PageContainer,
  ProForm,
  ProFormSelect,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { message } from 'antd';
import { useEffect, useState } from 'react';
import {
  getSystemConfig,
  updateSystemConfig,
  type SystemConfig,
} from '@/services/admin/api';

export default function SystemConfigPage() {
  const [initialValues, setInitialValues] = useState<SystemConfig>({});

  useEffect(() => {
    getSystemConfig().then((res) => {
      if (res.data) {
        setInitialValues(res.data);
      }
    });
  }, []);

  return (
    <PageContainer>
      <ProForm<SystemConfig>
        key={JSON.stringify(initialValues)}
        initialValues={initialValues}
        onFinish={async (values) => {
          await updateSystemConfig(values);
          message.success('配置已保存');
          return true;
        }}
        submitter={{ searchConfig: { submitText: '保存配置' } }}
      >
        <ProFormSelect
          name="registration_enabled"
          label="开放注册"
          options={[
            { label: '开启', value: 'true' },
            { label: '关闭', value: 'false' },
          ]}
          rules={[{ required: true }]}
        />
        <ProFormSelect
          name="beta_mode"
          label="内测模式"
          options={[
            { label: '开启', value: 'true' },
            { label: '关闭', value: 'false' },
          ]}
          rules={[{ required: true }]}
        />
        <ProFormSelect
          name="use_default_coins"
          label="使用默认币种"
          options={[
            { label: '是', value: 'true' },
            { label: '否', value: 'false' },
          ]}
          rules={[{ required: true }]}
        />
        <ProFormTextArea
          name="default_coins"
          label="默认币种列表 (JSON 数组)"
          fieldProps={{ rows: 4 }}
          rules={[{ required: true }]}
        />
        <ProFormSelect
          name="btc_eth_leverage"
          label="BTC/ETH 杠杆"
          options={['1', '2', '3', '5', '10', '20'].map((v) => ({
            label: `${v}x`,
            value: v,
          }))}
          rules={[{ required: true }]}
        />
        <ProFormSelect
          name="altcoin_leverage"
          label="山寨币杠杆"
          options={['1', '2', '3', '5', '10', '20'].map((v) => ({
            label: `${v}x`,
            value: v,
          }))}
          rules={[{ required: true }]}
        />
        <ProFormSelect
          name="max_daily_loss"
          label="最大日损失 (%)"
          options={['5', '10', '15', '20'].map((v) => ({
            label: v,
            value: v,
          }))}
        />
        <ProFormSelect
          name="max_drawdown"
          label="最大回撤 (%)"
          options={['10', '15', '20', '30'].map((v) => ({
            label: v,
            value: v,
          }))}
        />
        <ProFormSelect
          name="stop_trading_minutes"
          label="停止交易时间 (分钟)"
          options={['30', '60', '120', '240'].map((v) => ({
            label: v,
            value: v,
          }))}
        />
      </ProForm>
    </PageContainer>
  );
}
