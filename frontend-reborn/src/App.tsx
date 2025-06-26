// src/App.tsx
import React, { useState, useEffect, useMemo, Suspense, lazy } from 'react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Button, Grid, ConfigProvider, App as AntApp, FloatButton, Drawer, Dropdown, Card, Skeleton } from 'antd';
import { DashboardOutlined, SwapOutlined, CreditCardOutlined, TrophyOutlined, SettingOutlined, MenuFoldOutlined, MenuUnfoldOutlined, PlusOutlined, AppstoreAddOutlined, WalletOutlined, UserOutlined, LogoutOutlined, CrownOutlined } from '@ant-design/icons';
import zhCN from 'antd/locale/zh_CN';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { motion, AnimatePresence } from 'framer-motion';

import { QueryClient, QueryClientProvider, useQueryClient } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';

import LoginPage from './pages/LoginPage';
import AddTransactionModalWrapper from './components/AddTransactionModal';
import ProtectedRoute from './components/ProtectedRoute';
import ForcePasswordChangeModal from './components/ForcePasswordChangeModal';

import { useAuthStore, useCurrentUser, useMustChangePassword } from './stores/authStore';
import apiClient from './services/api';

const DashboardPage = lazy(() => import('./pages/DashboardPage'));
const TransactionPage = lazy(() => import('./pages/TransactionPage'));
const LoanPage = lazy(() => import('./pages/LoanPage'));
const BudgetPage = lazy(() => import('./pages/BudgetsPage'));
const SettingsPage = lazy(() => import('./pages/SettingsPage'));
const CategoryPage = lazy(() => import('./pages/CategoryPage'));
const AccountsPage = lazy(() => import('./pages/AccountsPage'));
const AdminPage = lazy(() => import('./pages/AdminPage'));

dayjs.locale('zh-cn');

const { Header, Sider, Content } = Layout;
const { useBreakpoint } = Grid;

const pageVariants = { initial: { opacity: 0, y: 20 }, in: { opacity: 1, y: 0 }, out: { opacity: 0, y: -20 } };
const pageTransition = { type: 'tween', ease: 'anticipate', duration: 0.5 } as const;

// 【新增】一个全局加载组件
const FullScreenLoader: React.FC = () => (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', background: 'var(--bg-color)' }}>
        <Card>
            <Skeleton active paragraph={{ rows: 2 }} />
        </Card>
    </div>
);

const MainLayout: React.FC = () => {
    const location = useLocation();
    const [collapsed, setCollapsed] = useState(false);
    const [currentPage, setCurrentPage] = useState<string>('dashboard');
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);
    const [drawerVisible, setDrawerVisible] = useState(false);
    
    const screens = useBreakpoint();
    const isMobile = !screens.lg;
    const navigate = useNavigate();
    const queryClient = useQueryClient();
    
    const { logout } = useAuthStore();
    const user = useCurrentUser();

    useEffect(() => {
        const path = location.pathname.substring(1).split('/')[0] || 'dashboard';
        setCurrentPage(path);
    }, [location]);

    useEffect(() => { if (isMobile) { setCollapsed(true); } }, [isMobile]);

    const handleMenuClick = ({ key }: { key: string }) => {
        navigate(`/${key}`);
        if (isMobile) { setDrawerVisible(false); }
    };
    
    const handleLogout = () => {
        logout();
        queryClient.clear();
        navigate('/login');
    };

    const menuItems = useMemo(() => {
        const baseItems = [
          { key: 'dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
          { key: 'transactions', icon: <SwapOutlined />, label: '交易流水' },
          { key: 'accounts', icon: <WalletOutlined />, label: '账户管理' },
          { key: 'budgets', icon: <TrophyOutlined />, label: '预算规划' },
          { key: 'categories', icon: <AppstoreAddOutlined />, label: '分类管理' },
          { key: 'loans', icon: <CreditCardOutlined />, label: '借贷管理' },
          { key: 'settings', icon: <SettingOutlined />, label: '数据中心' },
        ];

        if (user?.is_admin) {
            baseItems.push({ key: 'admin', icon: <CrownOutlined />, label: '管理后台' });
        }
        return baseItems;
    }, [user]);

    const menuContent = (
      <>
        <div style={{ height: '32px', margin: '24px 16px', textAlign: 'center', lineHeight: '32px', color: 'white', overflow: 'hidden', fontWeight: 'bold' }}>
            {(isMobile || !collapsed) ? '极简记账本' : '记'}
        </div>
        <Menu theme="dark" mode="inline" selectedKeys={[currentPage]} onClick={handleMenuClick} items={menuItems} style={{ background: 'transparent', borderRight: 0 }}/>
      </>
    );

    const userMenu = {
        items: [
            {
                key: 'logout',
                icon: <LogoutOutlined />,
                label: '退出登录',
                onClick: handleLogout,
            },
        ],
    };

    return (
        <Layout style={{ minHeight: '100vh' }}>
            {isMobile ? (
                <Drawer placement="left" onClose={() => setDrawerVisible(false)} open={drawerVisible} width={220} styles={{ body: { padding: 0, background: 'linear-gradient(180deg, #2a4c6b 0%, #1a354f 100%)' } }}>{menuContent}</Drawer>
            ) : (
                <Sider trigger={null} collapsible collapsed={collapsed} style={{ position: 'fixed', top: 0, left: 0, zIndex: 100, height: '100vh', background: 'linear-gradient(180deg, #2a4c6b 0%, #1a354f 100%)', boxShadow: '2px 0 15px rgba(0,21,41,0.35)' }} width={220}>{menuContent}</Sider>
            )}

            <Layout style={{ marginLeft: isMobile ? 0 : (collapsed ? 80 : 220), transition: 'margin-left 0.2s ease-in-out', background: 'var(--bg-color)' }}>
                <Header style={{ padding: '0 24px', background: 'rgba(255, 255, 255, 0.6)', backdropFilter: 'blur(10px)', WebkitBackdropFilter: 'blur(10px)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--border-color)', position: 'sticky', top: 0, zIndex: 10, width: '100%' }}>
                    <Button type="text" icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} onClick={() => isMobile ? setDrawerVisible(true) : setCollapsed(!collapsed)} style={{ fontSize: '16px', width: 64, height: 64 }}/>
                    <Dropdown menu={userMenu}>
                        <Button type="text" icon={<UserOutlined />}>
                            {user?.username}
                        </Button>
                    </Dropdown>
                </Header>
                <Content style={{ padding: '24px 16px', minHeight: 'calc(100vh - 64px)' }}>
                    <AnimatePresence mode="wait">
                        <motion.div key={location.pathname} initial="initial" animate="in" exit="out" variants={pageVariants} transition={pageTransition}>
                            <Suspense fallback={<Card><Skeleton active paragraph={{ rows: 5 }} /></Card>}>
                                <Routes>
                                    <Route path="/dashboard" element={<DashboardPage />} />
                                    <Route path="/transactions" element={<TransactionPage />} />
                                    <Route path="/accounts" element={<AccountsPage />} />
                                    <Route path="/budgets" element={<BudgetPage />} />
                                    <Route path="/categories" element={<CategoryPage />} />
                                    <Route path="/loans" element={<LoanPage />} />
                                    <Route path="/settings" element={<SettingsPage />} />
                                    {user?.is_admin && <Route path="/admin" element={<AdminPage />} />}
                                    <Route path="/" element={<DashboardPage />} />
                                </Routes>
                            </Suspense>
                        </motion.div>
                    </AnimatePresence>
                </Content>
            </Layout>
            <FloatButton icon={<PlusOutlined />} type="primary" tooltip="记一笔" style={{ right: isMobile ? 20 : 40 }} onClick={() => setIsAddModalOpen(true)}/>
            <AddTransactionModalWrapper open={isAddModalOpen} onClose={() => setIsAddModalOpen(false)} />
        </Layout>
    );
}

const App: React.FC = () => {
    const mustChangePassword = useMustChangePassword();
    const { login, logout, refreshToken } = useAuthStore();
    const [isAppLoading, setIsAppLoading] = useState(true);

    const [queryClient] = useState(() => new QueryClient({
        defaultOptions: {
            queries: {
              staleTime: 1000 * 60 * 5,
              refetchOnWindowFocus: true,
              retry: 1,
            },
        },
    }));

    // 【关键修改】应用启动时的认证检查逻辑
    useEffect(() => {
        const initializeAuth = async () => {
            if (refreshToken) {
                try {
                    const { data } = await apiClient.post('/auth/refresh', { refresh_token: refreshToken });
                    const newAccessToken = data.access_token;
                    const user = useAuthStore.getState().user;
                    if (user) {
                       login(newAccessToken, refreshToken, user, false);
                    } else {
                        logout();
                    }
                } catch (error) {
                    logout();
                }
            }
            setIsAppLoading(false);
        };

        initializeAuth();
    }, [login, logout, refreshToken]); 

    if (isAppLoading) {
        return <FullScreenLoader />;
    }

    return (
        <QueryClientProvider client={queryClient}>
            <ConfigProvider locale={zhCN} theme={{ token: { colorPrimary: '#2f54eb', borderRadius: 8, fontFamily: 'Inter, sans-serif' }, components: { Menu: { darkItemSelectedBg: '#2f54eb', itemHoverBg: 'rgba(255, 255, 255, 0.1)' } } }}>
                <AntApp>
                    <Routes>
                        <Route path="/login" element={<LoginPage />} />
                        <Route path="/*" element={<ProtectedRoute><MainLayout /></ProtectedRoute>} />
                    </Routes>
                    <ForcePasswordChangeModal open={mustChangePassword} />
                </AntApp>
            </ConfigProvider>
            {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
        </QueryClientProvider>
    );
};

export default App;