// src/App.tsx
import React, { useState, useEffect } from 'react';
import {
  Layout,
  Menu,
  Button,
  theme,
  Grid,
  ConfigProvider,
  App as AntApp, // 使用 AntApp 包裹以使用 message, Modal 等静态方法
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
  AppstoreAddOutlined, // 为“分类管理”新增的图标
} from '@ant-design/icons';
import zhCN from 'antd/locale/zh_CN';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';

// 导入所有页面组件
import DashboardPage from './pages/DashboardPage';
import TransactionPage from './pages/TransactionPage';
import LoanPage from './pages/LoanPage';
import BudgetPage from './pages/BudgetsPage';
import SettingsPage from './pages/SettingsPage';
import CategoryPage from './pages/CategoryPage'; // 【新增】导入我们新的分类页面

// 导入新增交易的弹窗组件
import AddTransactionModal from './components/AddTransactionModal';

// 设置 dayjs 的全局本地化为中文
dayjs.locale('zh-cn');

const { Header, Sider, Content } = Layout;
const { useBreakpoint } = Grid;

// 【修改】定义页面键的类型，加入 'categories'
type PageKey = 'dashboard' | 'transactions' | 'loans' | 'budgets' | 'categories' | 'settings';

// 主应用的核心逻辑组件
const MainApp = () => {
  const [collapsed, setCollapsed] = useState(false);
  const [currentPage, setCurrentPage] = useState<PageKey>('dashboard');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0); // 用于触发页面刷新的状态
  
  const screens = useBreakpoint();
  const isMobile = !screens.lg;

  const { token: { colorBgContainer, borderRadiusLG } } = theme.useToken();
  
  useEffect(() => { setCollapsed(isMobile); }, [isMobile]);

  const handleMenuClick = ({ key }: { key: string }) => { setCurrentPage(key as PageKey); };
  
  // 渲染当前选中的页面
  const renderPage = () => {
    // 传递 refreshKey，以便在需要时重新加载页面数据
    switch (currentPage) {
      case 'dashboard': return <DashboardPage key={refreshKey} />;
      case 'transactions': return <TransactionPage key={refreshKey} />;
      case 'loans': return <LoanPage key={refreshKey} />;
      case 'budgets': return <BudgetPage key={refreshKey} />;
      case 'categories': return <CategoryPage key={refreshKey} />;
      case 'settings': return <SettingsPage key={refreshKey} />;
      default: return <DashboardPage key={refreshKey} />;
    }
  };
  
  // 处理“记一笔”成功后的回调
  const handleAddSuccess = () => {
    setIsModalOpen(false);
    message.success('添加成功！');
    // 【优化】不再强制刷新整个网页，而是更新一个key来触发组件的重新渲染和数据获取
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
        style={{ position: 'fixed', top: 0, left: 0, zIndex: 100, height: '100vh' }}
      >
        <div style={{ height: '32px', margin: '16px', background: 'rgba(255, 255, 255, 0.2)', borderRadius: '6px', textAlign: 'center', lineHeight: '32px', color: 'white', overflow: 'hidden', transition: 'all 0.2s', fontWeight: 'bold' }}>
            {collapsed ? '记' : '简易记账本'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[currentPage]}
          onClick={handleMenuClick}
          // 【修改】在菜单项中加入“分类管理”
          items={[
            { key: 'dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
            { key: 'transactions', icon: <SwapOutlined />, label: '交易流水' },
            { key: 'budgets', icon: <TrophyOutlined />, label: '预算规划' },
            { key: 'categories', icon: <AppstoreAddOutlined />, label: '分类管理' },
            { key: 'loans', icon: <CreditCardOutlined />, label: '借贷管理' },
            { key: 'settings', icon: <SettingOutlined />, label: '数据中心' },
          ]}
        />
      </Sider>
      <Layout style={{ marginLeft: collapsed ? 80 : 200, transition: 'margin-left 0.2s' }}>
        <Header style={{ padding: 0, background: colorBgContainer, display: 'flex', alignItems: 'center', borderBottom: '1px solid #f0f0f0', position: 'sticky', top: 0, zIndex: 1, width: '100%' }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
            style={{ fontSize: '16px', width: 64, height: 64 }}
          />
        </Header>
        <Content
          style={{
            margin: '24px 16px',
            padding: 24,
            minHeight: 'calc(100vh - 112px)',
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
          }}
        >
          {renderPage()}
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

// 根组件，提供全局配置
const App: React.FC = () => (
    <ConfigProvider locale={zhCN} theme={{ token: { colorPrimary: '#1677ff' } }}>
        <AntApp>
            <MainApp />
        </AntApp>
    </ConfigProvider>
);

export default App;