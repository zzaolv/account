// src/App.tsx
import React, { useState, useEffect } from 'react';
import {
  Layout,
  Menu,
  Button,
  Grid,
  ConfigProvider,
  App as AntApp,
  FloatButton,
  message,
} from 'antd';
import {
  DashboardOutlined,
  SwapOutlined,
  CreditCardOutlined,
  TrophyOutlined,
  SettingOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  PlusOutlined,
  AppstoreAddOutlined,
  WalletOutlined,
} from '@ant-design/icons';
import zhCN from 'antd/locale/zh_CN';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';

import { motion, AnimatePresence } from 'framer-motion';

// 导入所有页面组件
import DashboardPage from './pages/DashboardPage';
import TransactionPage from './pages/TransactionPage';
import LoanPage from './pages/LoanPage';
import BudgetPage from './pages/BudgetsPage';
import SettingsPage from './pages/SettingsPage';
import CategoryPage from './pages/CategoryPage';
import AccountsPage from './pages/AccountsPage';

import AddTransactionModal from './components/AddTransactionModal';

dayjs.locale('zh-cn');

const { Header, Sider, Content } = Layout;
const { useBreakpoint } = Grid;

type PageKey = 'dashboard' | 'transactions' | 'loans' | 'budgets' | 'categories' | 'accounts' | 'settings';

const pageVariants = {
  initial: { opacity: 0, y: 20 },
  in: { opacity: 1, y: 0 },
  out: { opacity: 0, y: -20 },
} as const;

const pageTransition = {
  type: 'tween',
  ease: 'anticipate',
  duration: 0.5,
} as const;


const MainApp = () => {
  const [collapsed, setCollapsed] = useState(false);
  const [currentPage, setCurrentPage] = useState<PageKey>('dashboard');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);

  const screens = useBreakpoint();
  const isMobile = !screens.lg;

  // 【修复】删除这行未使用的代码
  // const { token: { colorBgContainer, borderRadiusLG } } = theme.useToken();

  useEffect(() => { setCollapsed(isMobile); }, [isMobile]);

  const handleMenuClick = ({ key }: { key: string }) => { setCurrentPage(key as PageKey); };
  
  const renderPage = () => {
    const pageKey = `${currentPage}-${refreshKey}`;
    switch (currentPage) {
      case 'dashboard': return <DashboardPage key={pageKey} />;
      case 'transactions': return <TransactionPage key={pageKey} />;
      case 'accounts': return <AccountsPage key={pageKey} />;
      case 'budgets': return <BudgetPage key={pageKey} />;
      case 'categories': return <CategoryPage key={pageKey} />;
      case 'loans': return <LoanPage key={pageKey} />;
      case 'settings': return <SettingsPage key={pageKey} />;
      default: return <DashboardPage key={pageKey} />;
    }
  };
  
  const handleAddSuccess = () => {
    setIsModalOpen(false);
    message.success('添加成功！');
    setRefreshKey(prevKey => prevKey + 1);
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider 
        trigger={null} 
        collapsible 
        collapsed={collapsed} 
        breakpoint="lg"
        onBreakpoint={(broken) => { setCollapsed(broken); }}
        style={{ 
            position: 'fixed', top: 0, left: 0, zIndex: 100, height: '100vh',
            background: 'linear-gradient(180deg, #2a4c6b 0%, #1a354f 100%)',
            boxShadow: '2px 0 15px rgba(0,21,41,0.35)'
        }}
        width={220}
      >
        <motion.div 
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.2 }}
            style={{ height: '32px', margin: '24px 16px', textAlign: 'center', lineHeight: '32px', color: 'white', overflow: 'hidden', fontWeight: 'bold' }}>
            {collapsed ? '记' : '极简记账本'}
        </motion.div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[currentPage]}
          onClick={handleMenuClick}
          style={{ background: 'transparent', borderRight: 0 }}
          items={[
            { key: 'dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
            { key: 'transactions', icon: <SwapOutlined />, label: '交易流水' },
            { key: 'accounts', icon: <WalletOutlined />, label: '账户管理' },
            { key: 'budgets', icon: <TrophyOutlined />, label: '预算规划' },
            { key: 'categories', icon: <AppstoreAddOutlined />, label: '分类管理' },
            { key: 'loans', icon: <CreditCardOutlined />, label: '借贷管理' },
            { key: 'settings', icon: <SettingOutlined />, label: '数据中心' },
          ]}
        />
      </Sider>
      <Layout style={{ marginLeft: collapsed ? 80 : 220, transition: 'margin-left 0.2s ease-in-out', background: 'var(--bg-color)' }}>
        <Header style={{ 
            padding: '0 24px', 
            background: 'rgba(255, 255, 255, 0.6)', 
            backdropFilter: 'blur(10px)', 
            WebkitBackdropFilter: 'blur(10px)',
            display: 'flex', 
            alignItems: 'center', 
            borderBottom: '1px solid var(--border-color)', 
            position: 'sticky', 
            top: 0, 
            zIndex: 10, 
            width: '100%' 
        }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
            style={{ fontSize: '16px', width: 64, height: 64 }}
          />
        </Header>
        <Content style={{ padding: '24px 16px', minHeight: 'calc(100vh - 64px)' }}>
            <AnimatePresence mode="wait">
                <motion.div
                    key={currentPage}
                    initial="initial"
                    animate="in"
                    exit="out"
                    variants={pageVariants}
                    transition={pageTransition}
                >
                    {renderPage()}
                </motion.div>
            </AnimatePresence>
        </Content>
      </Layout>
      <FloatButton 
        icon={<PlusOutlined />} 
        type="primary" 
        tooltip="记一笔"
        style={{ right: 40 }}
        onClick={() => setIsModalOpen(true)}
      />
      <AddTransactionModal 
        open={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSuccess={handleAddSuccess}
      />
    </Layout>
  );
};

const App: React.FC = () => (
    <ConfigProvider 
        locale={zhCN} 
        theme={{
            token: {
                colorPrimary: '#2f54eb',
                borderRadius: 12,
                fontFamily: `'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen', 'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue', sans-serif`,
                colorBgLayout: '#f0f2f5',
                colorBgContainer: '#ffffff',
                colorText: 'rgba(0, 0, 0, 0.88)',
                colorTextSecondary: 'rgba(0, 0, 0, 0.65)',
                colorTextTertiary: 'rgba(0, 0, 0, 0.45)',
            },
            components: {
                Menu: {
                    darkItemSelectedBg: '#2f54eb',
                    itemHoverBg: 'rgba(255, 255, 255, 0.1)',
                },
                Card: {
                    headerBg: 'transparent',
                    paddingLG: 20,
                },
            }
        }}
    >
        <AntApp>
            <MainApp />
        </AntApp>
    </ConfigProvider>
);

export default App;