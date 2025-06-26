// src/pages/AccountsPage.tsx
import React, { useState } from 'react';
import { Button, Card, Table, Tag, Modal, Form, Input, InputNumber, Select, Space, Popconfirm, Tooltip, Row, Col, Typography, App, Dropdown, Grid, DatePicker, Skeleton, Result, Empty, notification } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, StarFilled, StarOutlined, SwapOutlined, MoreOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getAccounts, createAccount, updateAccount, deleteAccount, setPrimaryAccount, addTransaction } from '../services/api';
import type { Account, UpdateAccountRequest, CreateAccountRequest, CreateTransactionRequest } from '../types';
import type { ColumnsType } from 'antd/es/table';
import IconDisplay, { availableIcons } from '../components/IconPicker';
import axios from 'axios';
import dayjs from 'dayjs';
import { motion } from 'framer-motion';

const { Title, Text } = Typography;
const { useBreakpoint } = Grid;

const accountTypeMap = { wechat: { name: '微信钱包' }, alipay: { name: '支付宝' }, card: { name: '储蓄卡' }, other: { name: '其他' } };
const MotionRow = (props: any) => (<motion.tr {...props} initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3 }} />);

// 金额格式化组件
const FormattedInputNumber: React.FC<any> = (props) => (
    <InputNumber
        {...props}
        formatter={(value) => `${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
        parser={(value) => (value ? value.replace(/\$\s?|(,*)/g, '') : '')}
    />
);

const AccountsPageContent: React.FC = () => {
    const [isEditModalOpen, setIsEditModalOpen] = useState(false);
    const [isTransferModalOpen, setIsTransferModalOpen] = useState(false);
    const [editingAccount, setEditingAccount] = useState<Account | null>(null);
    const [editForm] = Form.useForm();
    const [transferForm] = Form.useForm();
    
    const screens = useBreakpoint();
    const isMobile = !screens.sm;
    const queryClient = useQueryClient();
    const { message } = App.useApp();

    const { data: accounts = [], isLoading, isError, error, refetch } = useQuery<Account[], Error>({
        queryKey: ['accounts'],
        queryFn: () => getAccounts().then(res => res.data || []),
        retry: 1,
    });

    const mutationOptions = (successMsg: string) => ({
        onSuccess: () => {
            handleCancel();
            message.success(successMsg);
            queryClient.invalidateQueries({ queryKey: ['accounts'] });
            queryClient.invalidateQueries({ queryKey: ['dashboardCards'] });
            if (successMsg.includes('转账')) {
                queryClient.invalidateQueries({ queryKey: ['transactions'] });
            }
        },
        onError: (err: unknown) => {
            const errorMsg = axios.isAxiosError(err) && err.response ? err.response.data.error : '操作失败';
            notification.error({
                message: '操作失败',
                description: errorMsg,
            });
        },
    });

    const createMutation = useMutation({ mutationFn: (newData: CreateAccountRequest) => createAccount(newData), ...mutationOptions('账户创建成功！') });
    const updateMutation = useMutation({ mutationFn: (vars: { id: number; data: UpdateAccountRequest }) => updateAccount(vars.id, vars.data), ...mutationOptions('账户更新成功！') });
    const deleteMutation = useMutation<void, Error, number>({ mutationFn: async (id) => { await deleteAccount(id); }, ...mutationOptions('删除成功！') });
    const setPrimaryMutation = useMutation<void, Error, number>({ mutationFn: async (id) => { await setPrimaryAccount(id); }, ...mutationOptions('主账户设置成功！') });
    const transferMutation = useMutation({ mutationFn: (data: CreateTransactionRequest) => addTransaction(data), ...mutationOptions('转账成功！') });

    const handleCancel = () => {
        setIsEditModalOpen(false); setIsTransferModalOpen(false); setEditingAccount(null);
        editForm.resetFields(); transferForm.resetFields();
    };

    const openEditModal = (account: Account | null) => {
        setEditingAccount(account);
        editForm.setFieldsValue(account || { type: 'card', balance: 0, icon: 'Wallet' });
        setIsEditModalOpen(true);
    };

    const handleEditFormSubmit = (values: CreateAccountRequest | UpdateAccountRequest) => {
        if (editingAccount) {
            updateMutation.mutate({ id: editingAccount.id, data: values as UpdateAccountRequest });
        } else {
            createMutation.mutate(values as CreateAccountRequest);
        }
    };
    
    const handleTransferFormSubmit = (values: any) => {
        const postData: CreateTransactionRequest = { 
            type: 'transfer',
            from_account_id: values.from_account_id,
            to_account_id: values.to_account_id,
            amount: values.amount,
            transaction_date: values.date.format('YYYY-MM-DD'),
            description: values.description,
            category_id: 'transfer',
         };
        transferMutation.mutate(postData);
    };

    const getActionMenuItems = (record: Account) => {
        const isDeleting = deleteMutation.isPending && deleteMutation.variables === record.id;
        const isSettingPrimary = setPrimaryMutation.isPending && setPrimaryMutation.variables === record.id;
        const isMutating = isDeleting || isSettingPrimary;

        return {
            items: [
                { key: 'set_primary', icon: <StarOutlined />, label: '设为主账户', disabled: record.is_primary || isMutating, onClick: () => setPrimaryMutation.mutate(record.id) },
                { key: 'edit', icon: <EditOutlined />, label: '编辑', disabled: isMutating, onClick: () => openEditModal(record) },
                { key: 'delete', danger: true, icon: <DeleteOutlined />, label: ( <Popconfirm title="确定删除账户吗?" description="请先确保账户余额为0。" onConfirm={() => deleteMutation.mutate(record.id)} disabled={record.is_primary || isMutating}> <span style={{ color: (record.is_primary || isMutating) ? 'rgba(0,0,0,0.25)' : 'inherit' }}>删除</span> </Popconfirm> ) },
            ]
        };
    };
    
    const columns: ColumnsType<Account> = [
        { title: '账户名称', dataIndex: 'name', key: 'name', render: (_, record: Account) => (<Space><IconDisplay name={record.icon} /><Text strong>{record.name}</Text>{record.is_primary && <Tag icon={<StarFilled />} color="gold">主账户</Tag>}</Space>) },
        { title: '类型', dataIndex: 'type', key: 'type', responsive: ['md'], render: (type: keyof typeof accountTypeMap) => accountTypeMap[type].name },
        { title: '余额', dataIndex: 'balance', key: 'balance', align: 'right', render: (balance: number) => <Text type={balance < 0 ? 'danger' : undefined}>¥{balance.toFixed(2)}</Text> },
        { title: '操作', key: 'action', width: isMobile ? 60 : 150, align: 'center', render: (_, record: Account) => {
            const isDeleting = deleteMutation.isPending && deleteMutation.variables === record.id;
            const isSettingPrimary = setPrimaryMutation.isPending && setPrimaryMutation.variables === record.id;
            const isMutating = isDeleting || isSettingPrimary;

            if (isMobile) return <Dropdown menu={getActionMenuItems(record)} trigger={['click']} disabled={isMutating}><Button type="text" icon={<MoreOutlined />} loading={isMutating} /></Dropdown>;

            return (<Space>
                <Tooltip title={record.is_primary ? "主账户" : "设为主账户"}><Button type="text" icon={record.is_primary ? <StarFilled /> : <StarOutlined />} onClick={() => setPrimaryMutation.mutate(record.id)} disabled={record.is_primary || isMutating} loading={isSettingPrimary} /></Tooltip>
                <Tooltip title="编辑"><Button type="text" icon={<EditOutlined />} onClick={() => openEditModal(record)} disabled={isMutating} /></Tooltip>
                <Popconfirm title="确定删除账户吗?" description="请先确保账户余额为0。" onConfirm={() => deleteMutation.mutate(record.id)} disabled={record.is_primary || isMutating}><Tooltip title="删除"><Button type="text" danger icon={<DeleteOutlined />} loading={isDeleting} disabled={record.is_primary || isMutating} /></Tooltip></Popconfirm>
            </Space>);
        }},
    ];

    const renderMainContent = () => {
        if (isLoading) return <Card><Skeleton active paragraph={{ rows: 5 }} /></Card>;
        if (isError) return <Card><Result status="error" title="账户数据加载失败" subTitle={`错误: ${error.message}`} extra={<Button type="primary" onClick={() => refetch()}>点击重试</Button>} /></Card>;
        
        // 【关键修改】使用正确的 Empty 组件用法
        if (accounts.length === 0) return (
            <Card>
                <Empty description={
                    <span>
                        您还没有任何资金账户，快来添加第一个吧！
                        <br />
                        <Button type="primary" onClick={() => openEditModal(null)} style={{ marginTop: 16 }}>
                            立即新增
                        </Button>
                    </span>
                } />
            </Card>
        );

        return (
            <Card>
                <Table columns={columns} dataSource={accounts} rowKey="id" pagination={{ pageSize: 10 }} components={{ body: { row: MotionRow } }} scroll={{ x: 'max-content' }} />
            </Card>
        );
    }
    
    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card>
                <Row justify="space-between" align="middle">
                    <Col><Title level={4} style={{ margin: 0 }}>账户管理</Title></Col>
                    <Col><Space><Button icon={<SwapOutlined />} onClick={() => setIsTransferModalOpen(true)}>账户转账</Button><Button type="primary" icon={<PlusOutlined />} onClick={() => openEditModal(null)}>新增账户</Button></Space></Col>
                </Row>
            </Card>

            {renderMainContent()}

            <Modal title={editingAccount ? '编辑账户' : '新增账户'} open={isEditModalOpen} onOk={editForm.submit} onCancel={handleCancel} destroyOnClose confirmLoading={createMutation.isPending || updateMutation.isPending}>
                <Form form={editForm} layout="vertical" onFinish={handleEditFormSubmit}>
                    <Form.Item name="name" label="账户名称" rules={[{ required: true }]} validateTrigger="onBlur"><Input /></Form.Item>
                    <Form.Item name="type" label="账户类型" rules={[{ required: true }]}><Select disabled={!!editingAccount}>{Object.entries(accountTypeMap).map(([key, { name }]) => <Select.Option key={key} value={key}>{name}</Select.Option>)}</Select></Form.Item>
                    <Form.Item name="balance" label="初始余额" rules={[{ required: true }]}><FormattedInputNumber style={{ width: '100%' }} prefix="¥" precision={2} disabled={!!editingAccount} /></Form.Item>
                    <Form.Item name="icon" label="图标" rules={[{ required: true }]}><Select showSearch>{availableIcons.map((iconName: string) => (<Select.Option key={iconName} value={iconName}><Space><IconDisplay name={iconName} /> {iconName}</Space></Select.Option>))}</Select></Form.Item>
                </Form>
            </Modal>
            
            <Modal title="账户间转账" open={isTransferModalOpen} onOk={transferForm.submit} onCancel={handleCancel} destroyOnClose confirmLoading={transferMutation.isPending}>
                <Form form={transferForm} layout="vertical" onFinish={handleTransferFormSubmit} initialValues={{ date: dayjs() }}>
                    <Form.Item name="from_account_id" label="从账户" rules={[{ required: true }, ({ getFieldValue }) => ({ validator(_, value) { if (!value || getFieldValue('to_account_id') !== value) { return Promise.resolve(); } return Promise.reject(new Error('转出和转入账户不能相同!')); } })]}><Select placeholder="选择转出账户">{accounts?.map(acc => <Select.Option key={acc.id} value={acc.id}>{acc.name} (余额: {acc.balance.toFixed(2)})</Select.Option>)}</Select></Form.Item>
                    <Form.Item name="to_account_id" label="到账户" rules={[{ required: true }]}><Select placeholder="选择转入账户">{accounts?.map(acc => <Select.Option key={acc.id} value={acc.id}>{acc.name}</Select.Option>)}</Select></Form.Item>
                    <Form.Item name="amount" label="转账金额" rules={[{ required: true }]}><FormattedInputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} /></Form.Item>
                    <Form.Item name="date" label="转账日期" rules={[{ required: true }]}><DatePicker style={{ width: '100%' }} /></Form.Item>
                    <Form.Item name="description" label="备注 (可选)"><Input.TextArea rows={2} /></Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

const AccountsPage: React.FC = () => (
    <App>
        <AccountsPageContent />
    </App>
);

export default AccountsPage;