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

export async function listAdmins(params: {
  current?: number;
  pageSize?: number;
  email?: string;
}) {
  return request<{ data: AdminUser[]; total: number; success: boolean }>(
    '/api/admin/admins',
    { params },
  );
}

export async function createAdmin(email: string, password: string) {
  return request<{ success: boolean; message: string }>('/api/admin/admins', {
    method: 'POST',
    data: { email, password },
  });
}

export async function updateUserRole(userId: string, role: string) {
  return request<{ success: boolean; message: string }>(
    `/api/admin/users/${userId}/role`,
    { method: 'PUT', data: { role } },
  );
}

export async function resetUserPassword(userId: string, password: string) {
  return request<{ success: boolean; message: string }>(
    `/api/admin/users/${userId}/password`,
    { method: 'PUT', data: { password } },
  );
}

export async function deleteUser(userId: string) {
  return request<{ success: boolean; message: string }>(
    `/api/admin/users/${userId}`,
    { method: 'DELETE' },
  );
}

export type AdminTrader = {
  id: string;
  user_id: string;
  user_email: string;
  name: string;
  ai_model_id: string;
  ai_model_name: string;
  exchange_id: string;
  exchange_name: string;
  exchange_type: string;
  initial_balance: number;
  scan_interval_minutes: number;
  is_running: boolean;
  btc_eth_leverage: number;
  altcoin_leverage: number;
  trading_symbols: string;
  system_prompt_template: string;
  is_cross_margin: boolean;
  use_coin_pool: boolean;
  use_oi_top: boolean;
  created_at: string;
  updated_at: string;
};

export type AdminTraderDetail = AdminTrader & {
  ai_model_provider?: string;
  custom_prompt?: string;
  override_base_prompt?: boolean;
};

export async function listTraders(params: {
  current?: number;
  pageSize?: number;
  user_id?: string;
  email?: string;
  name?: string;
  is_running?: string;
}) {
  return request<{ data: AdminTrader[]; total: number; success: boolean }>(
    '/api/admin/traders',
    { params },
  );
}

export async function getTraderDetail(traderId: string) {
  return request<{ data: AdminTraderDetail; success: boolean }>(
    `/api/admin/traders/${traderId}`,
  );
}

export async function getTraderAccount(traderId: string) {
  return request<{ data: Record<string, unknown>; success: boolean }>(
    `/api/admin/traders/${traderId}/account`,
  );
}

export async function getTraderPositions(traderId: string) {
  return request<{ data: Record<string, unknown>[]; success: boolean }>(
    `/api/admin/traders/${traderId}/positions`,
  );
}

export async function stopTrader(traderId: string) {
  return request<{ success: boolean; message: string }>(
    `/api/admin/traders/${traderId}/stop`,
    { method: 'POST' },
  );
}
