// src/services/api.ts
import axios from 'axios';
import type { 
    Account,
    CreateAccountRequest,
    UpdateAccountRequest,
    TransferRequest,
    Category, 
    LoanResponse,
    SettleLoanRequest,
    Budget, 
    DashboardCard, 
    AnalyticsChartsResponse, 
    GetTransactionsResponse,
    CreateTransactionRequest,
    UpdateLoanRequest,
    DashboardWidgetsResponse
} from '../types';

// 【核心修改】将 baseURL 从绝对路径改为相对路径
// 这使得无论应用部署在哪个域名下，API 请求都能正确指向当前域名的 /api/v1 路径
// 然后由 Nginx 进行反向代理
const apiClient = axios.create({
    baseURL: '/api/v1', // 之前是 'http://localhost:8080/api/v1'
    timeout: 15000,
});

// --- 分类 API ---
export const getCategories = () => apiClient.get<Category[]>('/categories');
export const createCategory = (data: Omit<Category, 'created_at' | 'type'> & { type: string }) => apiClient.post<Category>('/categories', data);
export const updateCategory = (id: string, data: { name: string; icon: string }) => apiClient.put(`/categories/${id}`, data);
export const deleteCategory = (id: string) => apiClient.delete(`/categories/${id}`);

// --- 账户 API ---
export const getAccounts = () => apiClient.get<Account[]>('/accounts');
export const createAccount = (data: CreateAccountRequest) => apiClient.post('/accounts', data);
export const updateAccount = (id: number, data: UpdateAccountRequest) => apiClient.put(`/accounts/${id}`, data);
export const deleteAccount = (id: number) => apiClient.delete(`/accounts/${id}`);
export const setPrimaryAccount = (id: number) => apiClient.post(`/accounts/${id}/set_primary`);
export const transferFunds = (data: TransferRequest) => apiClient.post('/accounts/transfer', data);
export const executeMonthlyTransfer = () => apiClient.post('/accounts/execute_monthly_transfer');


// --- 仪表盘 & 分析 API ---
export const getDashboardCards = (params?: { year?: number, month?: number }) => apiClient.get<DashboardCard[]>('/dashboard/cards', { params });
export const getAnalyticsCharts = (params?: { year?: number, month?: number }) => apiClient.get<AnalyticsChartsResponse>('/analytics/charts', { params });
export const getDashboardWidgets = (params?: { year?: number, month?: number }) => apiClient.get<DashboardWidgetsResponse>('/dashboard/widgets', { params });

// --- 流水 API ---
export const getTransactions = (params?: { year?: number, month?: number }) => apiClient.get<GetTransactionsResponse>('/transactions', { params });
export const deleteTransaction = (id: number) => apiClient.delete(`/transactions/${id}`);
export const addTransaction = (data: CreateTransactionRequest) => apiClient.post('/transactions', data);

// --- 借贷 API ---
export const getLoans = () => apiClient.get<LoanResponse[]>('/loans');
export const createLoan = (loanData: UpdateLoanRequest) => apiClient.post('/loans', loanData);
export const updateLoan = (id: number, loanData: UpdateLoanRequest) => apiClient.put(`/loans/${id}`, loanData);
export const deleteLoan = (id: number) => apiClient.delete(`/loans/${id}`);
export const updateLoanStatus = (id: number, status: 'active') => apiClient.put(`/loans/${id}/status`, { status });
export const settleLoan = (id: number, data: SettleLoanRequest) => apiClient.post(`/loans/${id}/settle`, data);


// --- 预算 API ---
export const getBudgets = (params?: { year?: number; month?: number; }) => apiClient.get<Budget[]>('/budgets', { params });
export const createOrUpdateBudget = (budgetData: { category_id: string | null; amount: number; period: 'monthly' | 'yearly'; }) => apiClient.post('/budgets', budgetData);
export const deleteBudget = (id: number) => apiClient.delete(`/budgets/${id}`);

// --- 数据管理 API ---
// 【重要修改】导出功能也需要使用相对路径
export const exportData = () => {
    // window.location.href = 'http://localhost:8080/api/v1/data/export'; // 旧的错误方式
    window.location.href = '/api/v1/data/export'; // 新的正确方式
};
export const importData = (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return apiClient.post('/data/import', formData, { headers: { 'Content-Type': 'multipart/form-data' } });
};

export default apiClient;