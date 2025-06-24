// src/types.ts
export interface Category {
  id: string;
  name: string;
  type: 'income' | 'expense' | 'internal';
  icon: string;
  created_at: string;
}

export interface Transaction {
  id: number;
  type: 'income' | 'expense' | 'repayment' | 'transfer' | 'settlement';
  amount: number;
  transaction_date: string;
  description: string;
  related_loan_id?: number;
  category_id?: string;
  category_name?: string;
  created_at: string;
}

export interface Loan {
  id: number;
  principal: number;
  interest_rate: number;
  loan_date: string;
  repayment_date?: string | null;
  description?: string | null;
  status: 'active' | 'paid';
  created_at: string;
}

export interface Budget {
  id: number;
  category_id?: string | null;
  amount: number;
  period: 'monthly' | 'yearly';
  category_name?: string;
  spent: number;
  remaining: number;
  progress: number;
  year: number;
  month: number;
}

export interface Account {
    id: number;
    name: string;
    type: 'wechat' | 'alipay' | 'card' | 'other';
    balance: number;
    icon: string;
    is_primary: boolean;
    created_at: string;
}

// --- API 请求/响应模型 ---

export interface CreateTransactionRequest {
  type: 'income' | 'expense' | 'repayment' | 'transfer' | 'settlement';
  amount: number;
  transaction_date: string;
  description?: string;
  category_id?: string;
  related_loan_id?: number;
  from_account_id?: number;
  to_account_id?: number;
}

export interface UpdateLoanRequest {
  principal: number;
  interest_rate: number;
  loan_date: string;
  repayment_date?: string | null;
  description?: string;
}

// 【新增并导出】这个接口之前被遗漏了
export interface SettleLoanRequest {
  from_account_id: number;
  repayment_date: string;
  description?: string;
}

export interface LoanResponse {
  id: number;
  principal: number;
  interest_rate: number;
  loan_date: string;
  repayment_date?: string | null;
  description?: string | null;
  status: 'active' | 'paid';
  created_at: string;
  total_repaid: number;
  outstanding_balance: number;
}

export interface GetTransactionsResponse {
  transactions: Transaction[];
  summary: {
    total_income: number;
    total_expense: number;
    net_balance: number;
  };
}

export interface CreateAccountRequest {
    name: string;
    type: 'wechat' | 'alipay' | 'card' | 'other';
    balance: number;
    icon: string;
}

export interface UpdateAccountRequest {
    name: string;
    icon: string;
}

export interface TransferRequest {
    from_account_id: number;
    to_account_id: number;
    amount: number;
    date: string;
    description?: string;
}

export interface DashboardCard {
    title: string;
    value: number;
    prev_value: number;
    icon: string;
    meta?: {
        account_count?: number;
    };
}

export interface ChartDataPoint {
    name: string;
    value: number;
}

export interface AnalyticsChartsResponse {
    expense_trend: ChartDataPoint[];
    category_expense: ChartDataPoint[];
}

export interface DashboardBudgetSummary {
    period: 'monthly' | 'yearly';
    amount: number;
    spent: number;
    progress: number;
    is_set: boolean;
}

export interface DashboardLoanInfo {
    id: number;
    description: string;
    principal: number;
    outstanding_balance: number;
    loan_date: string;
    repayment_date?: string | null;
    repayment_amount_progress: number;
}

export interface DashboardWidgetsResponse {
    budgets: DashboardBudgetSummary[];
    loans: DashboardLoanInfo[];
}