// src/services/api.ts
import axios from 'axios';
import type { 
    Category, 
    LoanResponse,
    Budget, 
    DashboardCard, 
    AnalyticsChartsResponse, 
    GetTransactionsResponse,
    CreateTransactionRequest,
    UpdateLoanRequest,
    DashboardWidgetsResponse
} from '../types';

const apiClient = axios.create({
    baseURL: 'http://localhost:8080/api/v1',
    timeout: 15000,
});

// --- 分类 API ---
export const getCategories = () => apiClient.get<Category[]>('/categories');
export const createCategory = (data: Omit<Category, 'created_at'>) => apiClient.post<Category>('/categories', data);
export const updateCategory = (id: string, data: { name: string; icon: string }) => apiClient.put(`/categories/${id}`, data);
export const deleteCategory = (id: string) => apiClient.delete(`/categories/${id}`);

// --- 仪表盘 & 分析 API ---
export const getDashboardCards = (params?: { year?: number, month?: number }) => apiClient.get<DashboardCard[]>('/dashboard/cards', { params });
export const getAnalyticsCharts = (params?: { year?: number, month?: number }) => apiClient.get<AnalyticsChartsResponse>('/analytics/charts', { params });
// 【修正】让 getDashboardWidgets 支持参数传递
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
export const updateLoanStatus = (id: number, status: 'active' | 'paid') => apiClient.put(`/loans/${id}/status`, { status });

// --- 预算 API ---
export const getBudgets = (params?: { year?: number; month?: number; }) => apiClient.get<Budget[]>('/budgets', { params });
export const createOrUpdateBudget = (budgetData: { category_id: string | null; amount: number; period: 'monthly' | 'yearly'; }) => apiClient.post('/budgets', budgetData);
export const deleteBudget = (id: number) => apiClient.delete(`/budgets/${id}`);

// --- 数据管理 API ---
export const exportData = () => { window.location.href = 'http://localhost:8080/api/v1/data/export'; };
export const importData = (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return apiClient.post('/data/import', formData, { headers: { 'Content-Type': 'multipart/form-data' } });
};

export default apiClient;