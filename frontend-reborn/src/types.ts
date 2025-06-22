// src/types.ts
// ✨ 这是完全根据您的 Go 后端模型修正后的版本 ✨

// --- 基本模型 ---

export interface Category {
  id: string; // 修正: ID 是字符串 (例如 "food_dining")
  name: string;
  type: 'income' | 'expense';
  icon: string;
  created_at: string;
}

export interface Transaction {
  id: number;
  type: 'income' | 'expense' | 'repayment';
  amount: number;
  transaction_date: string;
  description: string;
  related_loan_id?: number; // 修正: 类型为 number
  category_id?: string;   // 修正: ID 是字符串
  category_name?: string; // 从后端 JOIN 查询得到
  created_at: string;
}

export interface Loan {
  id: number;
  principal: number;
  interest_rate: number;
  loan_date: string;
  repayment_date?: string | null; // 允许为 null
  description?: string | null;    // 允许为 null
  status: 'active' | 'paid';    // 修正: 状态是 active/paid
  created_at: string;
}

export interface Budget {
  id: number;
  category_id?: string | null; // 修正: ID是字符串, 允许为 null
  amount: number;
  period: 'monthly' | 'yearly';
  category_name?: string;
  spent: number;
  remaining: number;
  progress: number;
  year: number;
  month: number;
}

// --- API 请求/响应模型 ---

export interface CreateTransactionRequest {
  type: 'income' | 'expense' | 'repayment';
  amount: number;
  transaction_date: string;
  description?: string;
  category_id?: string;
  related_loan_id?: number;
}

export interface UpdateLoanRequest {
  principal: number;
  interest_rate: number;
  loan_date: string;
  repayment_date?: string | null;
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


// --- Dashboard & Analytics Types (这些之前是正确的) ---

export interface DashboardCard {
    title: string;
    value: number;
    prev_value: number;
    icon: string;
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