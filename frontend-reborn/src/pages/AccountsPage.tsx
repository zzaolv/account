// src/pages/AccountsPage.tsx
import React, { useState, useEffect, useCallback } from 'react';
import { Button, Card, Table, Tag, message, Modal, Form, Input, InputNumber, Select, Space, Popconfirm, Tooltip, Row, Col, Typography, DatePicker, App } from 'antd'; // 导入 App
import { PlusOutlined, EditOutlined, DeleteOutlined, StarFilled, StarOutlined, SwapOutlined, ThunderboltOutlined, QuestionCircleOutlined } from '@ant-design/icons';
import { getAccounts, createAccount, updateAccount, deleteAccount, setPrimaryAccount, transferFunds, executeMonthlyTransfer } from '../services/api';
import type { Account, CreateAccountRequest, UpdateAccountRequest, TransferRequest } from '../types';
import type { ColumnsType } from 'antd/es/table';
import IconDisplay, { availableIcons } from '../components/IconPicker';
import axios from 'axios';
import dayjs from 'dayjs';

const { Title, Paragraph, Text } = Typography;

const accountTypeMap = {
    wechat: { name: '微信钱包', icon: 'Wallet' },
    alipay: { name: '支付宝', icon: 'Briefcase' },
    card: { name: '储蓄卡', icon: 'CreditCard' },
    other: { name: '其他', icon: 'Archive' },
};

const AccountsPage: React.FC = () => {
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [loading, setLoading] = useState(true);
    const [isEditModalOpen, setIsEditModalOpen] = useState(false);
    const [isTransferModalOpen, setIsTransferModalOpen] = useState(false);
    const [editingAccount, setEditingAccount] = useState<Account | null>(null);
    const [editForm] = Form.useForm();
    const [transferForm] = Form.useForm();
    
    // 【修复 1/2】使用 useApp hook 来获取 modal 实例，解决静态方法警告
    const { modal } = App.useApp();

    const fetchAccounts = useCallback(async () => {
        setLoading(true);
        try {
            const res = await getAccounts();
            setAccounts(res.data || []);
        } catch (error) {
            message.error('获取账户列表失败');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchAccounts();
    }, [fetchAccounts]);
    
    // --- Modal Handlers ---
    const handleCancel = () => {
        setIsEditModalOpen(false);
        setIsTransferModalOpen(false);
        setEditingAccount(null);
        editForm.resetFields();
        transferForm.resetFields();
    };

    const openEditModal = (account: Account | null) => {
        setEditingAccount(account);
        if (account) {
            editForm.setFieldsValue(account);
        } else {
            editForm.setFieldsValue({ type: 'card', balance: 0, icon: 'CreditCard' });
        }
        setIsEditModalOpen(true);
    };

    // --- Action Handlers ---
    const handleEditFormSubmit = async (values: any) => {
        try {
            if (editingAccount) {
                const updateData: UpdateAccountRequest = { name: values.name, icon: values.icon };
                await updateAccount(editingAccount.id, updateData);
                message.success('账户更新成功！');
            } else {
                const createData: CreateAccountRequest = values;
                await createAccount(createData);
                message.success('账户添加成功！');
            }
            handleCancel();
            fetchAccounts();
        } catch (error) {
            message.error(axios.isAxiosError(error) ? error.response?.data.error : '操作失败');
        }
    };
    
    const handleTransferFormSubmit = async (values: any) => {
        const postData: TransferRequest = {
            ...values,
            date: values.date.format('YYYY-MM-DD'),
        };
        try {
            await transferFunds(postData);
            message.success('转账成功！');
            handleCancel();
            fetchAccounts();
        } catch (error) {
            message.error(axios.isAxiosError(error) ? error.response?.data.error : '转账失败');
        }
    };

    const handleSetPrimary = async (id: number) => {
        try {
            await setPrimaryAccount(id);
            message.success('主账户设置成功！');
            fetchAccounts();
        } catch (error) {
            message.error(axios.isAxiosError(error) ? error.response?.data.error : '设置失败');
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await deleteAccount(id);
            message.success('删除成功！');
            fetchAccounts();
        } catch (error) {
            message.error(axios.isAxiosError(error) ? error.response?.data.error : '删除失败');
        }
    };
    
    const handleExecuteMonthlyTransfer = async () => {
        // 【修复 2/2】使用 hook 返回的 modal 实例
        modal.confirm({
            title: '确认执行上月结算吗？',
            content: '系统将计算上一个月的净收入，并自动将差额转入/转出您的主账户。此操作会生成一条流水记录。',
            okText: '确认执行',
            cancelText: '取消',
            onOk: async () => {
                try {
                    const res = await executeMonthlyTransfer();
                    message.success(res.data.message);
                    modal.info({
                        title: '结算完成',
                        content: res.data.details,
                    });
                    fetchAccounts();
                } catch(error) {
                    message.error(axios.isAxiosError(error) ? error.response?.data.error : '结算失败');
                }
            }
        });
    };

    const columns: ColumnsType<Account> = [
        {
            title: '账户名称',
            key: 'name',
            render: (_, record) => (
                <Space>
                    <IconDisplay name={record.icon} />
                    <Text strong>{record.name}</Text>
                    {record.is_primary && <Tag icon={<StarFilled />} color="gold">主账户</Tag>}
                </Space>
            )
        },
        { title: '类型', dataIndex: 'type', key: 'type', render: (type: keyof typeof accountTypeMap) => accountTypeMap[type].name },
        { title: '余额', dataIndex: 'balance', key: 'balance', render: (balance: number) => <Text type={balance < 0 ? 'danger' : undefined}>¥{balance.toFixed(2)}</Text> },
        {
            title: '操作',
            key: 'action',
            width: 250,
            align: 'center',
            render: (_, record) => (
                <Space>
                    <Tooltip title={record.is_primary ? "已经是主账户" : "设为主账户"}>
                        <Button icon={record.is_primary ? <StarFilled /> : <StarOutlined />} onClick={() => handleSetPrimary(record.id)} disabled={record.is_primary} />
                    </Tooltip>
                    <Tooltip title="编辑">
                        <Button icon={<EditOutlined />} onClick={() => openEditModal(record)} />
                    </Tooltip>
                    <Popconfirm title="确定删除账户吗?" description="请先确保账户余额为0。" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
                        <Tooltip title="删除">
                            <Button icon={<DeleteOutlined />} danger />
                        </Tooltip>
                    </Popconfirm>
                </Space>
            ),
        },
    ];

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card>
                <Row justify="space-between" align="middle">
                    <Col>
                        <Title level={2} style={{ margin: 0 }}>账户管理</Title>
                        <Paragraph type="secondary">管理您的所有存款账户，并进行月度结算。</Paragraph>
                    </Col>
                    <Col>
                        <Space>
                            <Button icon={<SwapOutlined />} onClick={() => setIsTransferModalOpen(true)}>账户转账</Button>
                            <Button type="primary" icon={<PlusOutlined />} onClick={() => openEditModal(null)}>新增账户</Button>
                        </Space>
                    </Col>
                </Row>
            </Card>
            
             <Card 
                title="智能月度转存" 
                extra={<Button type="dashed" icon={<ThunderboltOutlined />} onClick={handleExecuteMonthlyTransfer}>手动执行上月结算</Button>}
             >
                <Paragraph>
                    此功能会在每月结束时，自动计算上个月的 <Text strong>总收入 - 总支出</Text>，并将净结余转入您的 <Tag icon={<StarFilled />} color="gold">主账户</Tag>。
                    <Tooltip title="例如，5月31日结束后，系统会计算5月份的净收入，并在6月1日将这笔钱存入主账户。如果净收入为负，则会从主账户扣除。">
                        <QuestionCircleOutlined style={{ marginLeft: 4, cursor: 'pointer' }} />
                    </Tooltip>
                </Paragraph>
             </Card>

            <Card>
                <Table columns={columns} dataSource={accounts} rowKey="id" loading={loading} pagination={{ pageSize: 10 }} />
            </Card>

            {/* 【修复】使用 destroyOnHidden 替代 destroyOnClose */}
            <Modal title={editingAccount ? '编辑账户' : '新增账户'} open={isEditModalOpen} onOk={editForm.submit} onCancel={handleCancel} destroyOnHidden>
                <Form form={editForm} layout="vertical" onFinish={handleEditFormSubmit}>
                    <Form.Item name="name" label="账户名称" rules={[{ required: true }]}><Input /></Form.Item>
                    <Form.Item name="type" label="账户类型" rules={[{ required: true }]}>
                        <Select disabled={!!editingAccount}>
                            {Object.entries(accountTypeMap).map(([key, { name }]) => <Select.Option key={key} value={key}>{name}</Select.Option>)}
                        </Select>
                    </Form.Item>
                    <Form.Item name="balance" label="初始余额" rules={[{ required: true }]}>
                        <InputNumber style={{ width: '100%' }} prefix="¥" precision={2} disabled={!!editingAccount} />
                    </Form.Item>
                    <Form.Item name="icon" label="图标" rules={[{ required: true }]}>
                        <Select showSearch>
                            {availableIcons.map(iconName => (<Select.Option key={iconName} value={iconName}><Space><IconDisplay name={iconName} /> {iconName}</Space></Select.Option>))}
                        </Select>
                    </Form.Item>
                </Form>
            </Modal>
            
            {/* 【修复】使用 destroyOnHidden 替代 destroyOnClose */}
            <Modal title="账户间转账" open={isTransferModalOpen} onOk={transferForm.submit} onCancel={handleCancel} destroyOnHidden>
                <Form form={transferForm} layout="vertical" onFinish={handleTransferFormSubmit} initialValues={{ date: dayjs() }}>
                    <Form.Item name="from_account_id" label="从账户" rules={[{ required: true }]}>
                        <Select placeholder="选择转出账户">{accounts.map(acc => <Select.Option key={acc.id} value={acc.id}>{acc.name} (余额: {acc.balance.toFixed(2)})</Select.Option>)}</Select>
                    </Form.Item>
                    <Form.Item name="to_account_id" label="到账户" rules={[{ required: true }]}>
                        <Select placeholder="选择转入账户">{accounts.map(acc => <Select.Option key={acc.id} value={acc.id}>{acc.name}</Select.Option>)}</Select>
                    </Form.Item>
                    <Form.Item name="amount" label="转账金额" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} /></Form.Item>
                    <Form.Item name="date" label="转账日期" rules={[{ required: true }]}><DatePicker style={{ width: '100%' }} /></Form.Item>
                    <Form.Item name="description" label="备注 (可选)"><Input.TextArea rows={2} /></Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

export default AccountsPage;