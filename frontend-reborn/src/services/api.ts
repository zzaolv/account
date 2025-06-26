// src/services/api.ts
import axios from 'axios';
import { useAuthStore } from '../stores/authStore';
import type { 
    Category, LoanResponse, Budget, Account, DashboardCard, AnalyticsChartsResponse, 
    GetTransactionsResponse, CreateTransactionRequest, UpdateLoanRequest, 
    DashboardWidgetsResponse, CreateAccountRequest, UpdateAccountRequest,
    SettleLoanRequest, CreateCategoryRequest
} from '../types';

export interface LoginRequest { username: string; password: string; rememberMe: boolean; }
export interface LoginResponse { 
    access_token: string;
    refresh_token?: string;
    must_change_password: boolean;
    username: string;
    is_admin: boolean;
}
export interface UpdatePasswordRequest { old_password?: string; new_password: string; }
export interface SystemStats { user_count: number; transaction_count: number; account_count: number; db_size_bytes: number; }
export interface User { id: number; username: string; is_admin: boolean; must_change_password: boolean; created_at: string; }
export interface RegisterRequest { username: string; password: string; }

const apiClient = axios.create({ baseURL: '/api/v1', timeout: 15000 });

apiClient.interceptors.request.use(
  (config) => {
    const token = useAuthStore.getState().token;
    if (token) { config.headers.Authorization = `Bearer ${token}`; }
    return config;
  },
  (error) => Promise.reject(error)
);


let isRefreshing = false;
let failedQueue: { resolve: (value: any) => void, reject: (reason?: any) => void }[] = [];

const processQueue = (error: any, token: string | null = null) => {
    failedQueue.forEach(prom => {
        if (error) {
            prom.reject(error);
        } else {
            prom.resolve(token);
        }
    });
    failedQueue = [];
};

apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    if (!error.response) {
      return Promise.reject(error);
    }

    const { status } = error.response;
    const { logout, refreshToken, setNewAccessToken } = useAuthStore.getState();

    // 【修改】增加条件，忽略登录和刷新接口的401错误，让它们自己处理
    if (status === 401 && originalRequest.url === '/auth/login') {
        return Promise.reject(error);
    }
     if (status === 401 && originalRequest.url === '/auth/refresh') {
        logout(); // 刷新Token失败，直接登出
        window.location.href = '/login';
        return Promise.reject(error);
    }

    // 只有当 Access Token 过期 (401) 且我们有 Refresh Token 时才尝试刷新
    if (status === 401 && !originalRequest._retry && refreshToken) {
        if (isRefreshing) {
            return new Promise((resolve, reject) => {
                failedQueue.push({ resolve, reject });
            }).then(token => {
                originalRequest.headers['Authorization'] = 'Bearer ' + token;
                return apiClient(originalRequest);
            }).catch(err => {
                return Promise.reject(err);
            });
        }

        originalRequest._retry = true;
        isRefreshing = true;

        try {
            const { data } = await apiClient.post('/auth/refresh', { refresh_token: refreshToken });
            const newAccessToken = data.access_token;
            
            setNewAccessToken(newAccessToken);
            apiClient.defaults.headers.common['Authorization'] = 'Bearer ' + newAccessToken;
            originalRequest.headers['Authorization'] = 'Bearer ' + newAccessToken;
            
            processQueue(null, newAccessToken);
            return apiClient(originalRequest);
        } catch (refreshError) {
            processQueue(refreshError, null);
            logout();
            window.location.href = '/login';
            return Promise.reject(refreshError);
        } finally {
            isRefreshing = false;
        }
    } else if (status === 401) {
        logout();
        window.location.href = '/login';
    }

    return Promise.reject(error);
  }
);


export const login = (data: LoginRequest) => apiClient.post<LoginResponse>('/auth/login', data);
export const updatePassword = (data: UpdatePasswordRequest) => apiClient.put('/auth/update_password', data);

export const getCategories = () => apiClient.get<Category[]>('/categories');
export const createCategory = (data: CreateCategoryRequest) => apiClient.post<Category>('/categories', data);
export const updateCategory = (id: string, data: { name: string; icon: string }) => apiClient.put(`/categories/${id}`, data);
export const deleteCategory = (id: string) => apiClient.delete(`/categories/${id}`);

export const getAccounts = () => apiClient.get<Account[]>('/accounts');
export const createAccount = (data: CreateAccountRequest) => apiClient.post('/accounts', data);
export const updateAccount = (id: number, data: UpdateAccountRequest) => apiClient.put(`/accounts/${id}`, data);
export const deleteAccount = (id: number) => apiClient.delete(`/accounts/${id}`);
export const setPrimaryAccount = (id: number) => apiClient.post(`/accounts/${id}/set_primary`);

export const getDashboardCards = (params?: { year?: number, month?: number }) => apiClient.get<DashboardCard[]>('/dashboard/cards', { params });
export const getAnalyticsCharts = (params?: { year?: number, month?: number }) => apiClient.get<AnalyticsChartsResponse>('/analytics/charts', { params });
export const getDashboardWidgets = (params?: { year?: number, month?: number }) => apiClient.get<DashboardWidgetsResponse>('/dashboard/widgets', { params });

export const getTransactions = (params?: { year?: number, month?: number }) => apiClient.get<GetTransactionsResponse>('/transactions', { params });
export const deleteTransaction = (id: number) => apiClient.delete(`/transactions/${id}`);
export const addTransaction = (data: CreateTransactionRequest) => apiClient.post('/transactions', data);

export const getLoans = () => apiClient.get<LoanResponse[]>('/loans');
export const createLoan = (loanData: UpdateLoanRequest) => apiClient.post('/loans', loanData);
export const updateLoan = (id: number, loanData: UpdateLoanRequest) => apiClient.put(`/loans/${id}`, loanData);
export const deleteLoan = (id: number) => apiClient.delete(`/loans/${id}`);
export const updateLoanStatus = (id: number, status: 'active') => apiClient.put(`/loans/${id}/status`, { status });
export const settleLoan = (id: number, data: SettleLoanRequest) => apiClient.post(`/loans/${id}/settle`, data);

export const getBudgets = (params?: { year?: number; month?: number; }) => apiClient.get<Budget[]>('/budgets', { params });
export const createOrUpdateBudget = (budgetData: { category_id: string | null; amount: number; period: 'monthly' | 'yearly'; year: number; month?: number; }) => apiClient.post('/budgets', budgetData);
export const deleteBudget = (id: number) => apiClient.delete(`/budgets/${id}`);

export const exportData = () => { window.location.href = '/api/v1/data/export'; };
export const importData = (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return apiClient.post('/data/import', formData, { headers: { 'Content-Type': 'multipart/form-data' } });
};

export const getSystemStats = () => apiClient.get<SystemStats>('/admin/stats');
export const getUsers = () => apiClient.get<User[]>('/admin/users');
export const registerUser = (data: RegisterRequest) => apiClient.post('/admin/users/register', data);
export const deleteUser = (id: number) => apiClient.delete(`/admin/users/${id}`);

export default apiClient;