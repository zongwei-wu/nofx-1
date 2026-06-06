import { request } from '@umijs/max';

export type AdminUser = {
  id: string;
  email: string;
  role: string;
  plan: string;
  otp_verified: boolean;
  created_at: string;
};

export type PlanInfo = {
  id: string;
  name: string;
  sort_order: number;
  features: string[];
};

export type CopyTradeRecord = {
  id: number;
  user_id: string;
  portfolio_id: string;
  nickname: string;
  order_id: string;
  symbol: string;
  side: string;
  position_side: string;
  executed_qty: number;
  avg_price: number;
  total_pnl: number;
  status: string;
  lead_order_time: number;
  copy_time: string;
  close_time?: string;
  close_price: number;
  error_message: string;
};

export type SystemConfig = Record<string, string>;

export async function login(email: string, password: string) {
  return request<{
    token: string;
    user_id: string;
    email: string;
    role?: string;
    message: string;
  }>('/api/login', {
    method: 'POST',
    data: { email, password },
    skipErrorHandler: true,
  });
}

export async function adminMe() {
  return request<{ user_id: string; email: string; role: string }>(
    '/api/admin/me',
  );
}

export async function listUsers(params: {
  current?: number;
  pageSize?: number;
  email?: string;
}) {
  return request<{ data: AdminUser[]; total: number; success: boolean }>(
    '/api/admin/users',
    { params },
  );
}

export async function listCopyTradeRecords(params: {
  current?: number;
  pageSize?: number;
  user_id?: string;
  status?: string;
  symbol?: string;
}) {
  return request<{
    data: CopyTradeRecord[];
    total: number;
    success: boolean;
  }>('/api/admin/copy-trade/records', { params });
}

export async function getSystemConfig() {
  return request<{ data: SystemConfig; success: boolean }>(
    '/api/admin/system-config',
  );
}

export async function updateSystemConfig(data: SystemConfig) {
  return request<{ success: boolean; message: string }>(
    '/api/admin/system-config',
    { method: 'PUT', data },
  );
}

export async function listPlans() {
  return request<{ data: PlanInfo[]; success: boolean }>('/api/admin/plans');
}

export async function updateUserPlan(userId: string, plan: string) {
  return request<{ success: boolean; message: string }>(
    `/api/admin/users/${userId}`,
    { method: 'PUT', data: { plan } },
  );
}
