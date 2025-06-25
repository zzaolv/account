// src/pages/LoanPage.tsx
import React, { useState, useEffect, useCallback } from 'react';
// 【新增】导入 Grid, Dropdown, Menu
import { Button, Card, Table, Tag, message, Modal, Form, Input, InputNumber, DatePicker, Space, Popconfirm, Select, Typography, Grid, Dropdown, Menu } from 'antd';
// 【新增】导入 MoreOutlined 图标
import { PlusOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined, UndoOutlined, MoreOutlined } from '@ant-design/icons';
import { getLoans, createLoan, updateLoan, deleteLoan, updateLoanStatus, getAccounts, settleLoan } from '../services/api';
import type { LoanResponse, UpdateLoanRequest, Account, SettleLoanRequest } from '../types';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import axios from 'axios';
import { motion } from 'framer-motion';

const { Text, Title } = Typography;
const { useBreakpoint } = Grid; // 【新增】正确使用 useBreakpoint

const MotionRow = (props: any) => (
    <motion.tr {...props} initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3 }} />
);

const LoanPage: React.FC = () => {
    const [loans, setLoans] = useState<LoanResponse[]>([]);
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [loading, setLoading] = useState(true);
    const [isEditModalOpen, setIsEditModalOpen] = useState(false);
    const [isSettleModalOpen, setIsSettleModalOpen] = useState(false);
    const [editingLoan, setEditingLoan] = useState<LoanResponse | null>(null);
    const [settlingLoan, setSettlingLoan] = useState<LoanResponse | null>(null);
    const [editForm] = Form.useForm();
    const [settleForm] = Form.useForm();
    
    const screens = useBreakpoint(); // 【新增】获取屏幕断点信息
    const isMobile = !screens.md; // 【新增】定义 md (768px) 以下为移动端，给借贷页面多一点空间

    const fetchData = useCallback(async () => {
        setLoading(true);
        try {
            const [loanRes, accountRes] = await Promise.all([getLoans(), getAccounts()]);
            setLoans(loanRes.data || []);
            setAccounts(accountRes.data || []);
        } catch (error) { message.error('获取数据失败'); } finally { setLoading(false); }
    }, []);

    useEffect(() => { fetchData(); }, [fetchData]);

    const handleCancel = () => { setIsEditModalOpen(false); setIsSettleModalOpen(false); setEditingLoan(null); setSettlingLoan(null); editForm.resetFields(); settleForm.resetFields(); };
    
    const openEditModal = (loan: LoanResponse | null) => {
        setEditingLoan(loan);
        if (loan) { editForm.setFieldsValue({ ...loan, loan_date: dayjs(loan.loan_date), repayment_date: loan.repayment_date ? dayjs(loan.repayment_date) : null, interest_rate: loan.interest_rate * 100 });
        } else { editForm.resetFields(); }
        setIsEditModalOpen(true);
    };

    const openSettleModal = (loan: LoanResponse) => {
        setSettlingLoan(loan);
        settleForm.setFieldsValue({ repayment_date: dayjs(), description: `还清贷款: ${loan.description || `贷款 #${loan.id}`}` });
        setIsSettleModalOpen(true);
    };

    const handleEditFormSubmit = async (values: any) => {
        const postData: UpdateLoanRequest = { ...values, interest_rate: parseFloat(values.interest_rate) / 100, loan_date: values.loan_date.format('YYYY-MM-DD'), repayment_date: values.repayment_date ? values.repayment_date.format('YYYY-MM-DD') : null };
        try {
            if (editingLoan) { await updateLoan(editingLoan.id, postData); message.success('借贷记录更新成功！');
            } else { await createLoan(postData); message.success('借贷记录添加成功！'); }
            handleCancel(); fetchData();
        } catch (error) { message.error(axios.isAxiosError(error) ? error.response?.data.error : '操作失败'); }
    };

    const handleSettleFormSubmit = async (values: any) => {
        if (!settlingLoan) return;
        const postData: SettleLoanRequest = { ...values, repayment_date: values.repayment_date.format('YYYY-MM-DD') };
        try {
            await settleLoan(settlingLoan.id, postData);
            message.success('贷款已成功还清！'); handleCancel(); fetchData();
        } catch (error) { message.error(axios.isAxiosError(error) ? error.response?.data.error : '还清操作失败'); }
    };
    
    const handleRestoreStatus = async (id: number) => {
        try {
            await updateLoanStatus(id, 'active');
            message.success('状态已恢复为 "活动中"！'); fetchData();
        } catch (error) { message.error('恢复失败'); }
    };

    const handleDelete = async (id: number) => {
        try {
            await deleteLoan(id);
            message.success('删除成功！'); fetchData();
        } catch (error: unknown) { message.error(axios.isAxiosError(error) ? error.response?.data.error : '删除失败'); }
    };

    // 【核心修改】定义借贷管理的操作菜单项
    const getActionMenuItems = (record: LoanResponse) => {
        const items = [];
        items.push(<Menu.Item key="edit" icon={<EditOutlined />} onClick={() => openEditModal(record)}>编辑</Menu.Item>);
        if (record.status === 'active') {
            items.push(<Menu.Item key="settle" icon={<CheckCircleOutlined />} onClick={() => openSettleModal(record)}>还清</Menu.Item>);
        }
        if (record.status === 'paid') {
            items.push(
                <Menu.Item key="restore" icon={<UndoOutlined />}>
                    <Popconfirm title="确定要将此贷款恢复为“活动中”吗？" onConfirm={() => handleRestoreStatus(record.id)}>
                        <span style={{color: 'inherit'}}>恢复</span>
                    </Popconfirm>
                </Menu.Item>
            );
        }
        items.push(
            <Menu.Item key="delete" danger icon={<DeleteOutlined />}>
                <Popconfirm title="确定删除这笔借款吗?" description="有关联还款记录的无法删除" onConfirm={() => handleDelete(record.id)}>
                    <span style={{color: 'inherit'}}>删除</span>
                </Popconfirm>
            </Menu.Item>
        );
        return <Menu>{items}</Menu>;
    };

    const columns: ColumnsType<LoanResponse> = [
        { title: '描述', dataIndex: 'description', key: 'description' },
        { title: '本金', dataIndex: 'principal', key: 'principal', responsive: ['md'], render: (amount: number) => `¥${amount.toFixed(2)}` },
        { title: '待还余额', dataIndex: 'outstanding_balance', key: 'outstanding_balance', render: (amount: number) => <Text type="danger" strong>¥{amount.toFixed(2)}</Text> },
        { title: '借款日期', dataIndex: 'loan_date', key: 'loan_date', responsive: ['lg'] },
        { title: '状态', dataIndex: 'status', key: 'status', responsive: ['sm'], render: (status: string) => <Tag color={status === 'active' ? 'processing' : 'success'}>{status === 'active' ? '活动中' : '已还清'}</Tag> },
        { 
            title: '操作', 
            key: 'action', 
            align: 'center', 
            width: isMobile ? 60 : 220, // 移动端宽度变小
            render: (_, record: LoanResponse) => (
                isMobile ? (
                    // 移动端渲染 Dropdown
                    <Dropdown overlay={getActionMenuItems(record)} trigger={['click']}>
                        <Button type="text" icon={<MoreOutlined />} />
                    </Dropdown>
                ) : (
                    // 桌面端保持原有布局
                    <Space>
                        <Button type="link" icon={<EditOutlined />} onClick={() => openEditModal(record)}>编辑</Button>
                        {record.status === 'active' && (<Button type="link" icon={<CheckCircleOutlined />} onClick={() => openSettleModal(record)}>还清</Button>)}
                        {record.status === 'paid' && (<Popconfirm title="确定要将此贷款恢复为“活动中”吗？" onConfirm={() => handleRestoreStatus(record.id)}><Button type="link" icon={<UndoOutlined />}>恢复</Button></Popconfirm>)}
                        <Popconfirm title="确定删除这笔借款吗?" description="有关联还款记录的无法删除" onConfirm={() => handleDelete(record.id)}><Button type="link" danger icon={<DeleteOutlined />}>删除</Button></Popconfirm>
                    </Space>
                )
            ), 
        },
    ];
    
    // 【优化】根据是否为移动端，动态生成最终的列配置
    const getFinalColumns = () => {
        if (!isMobile) return columns;
        // 在移动端，我们只保留“描述”、“待还余额”和“操作”
        return columns.filter(col => ['description', 'outstanding_balance', 'action'].includes(String(col.key)));
    };

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card title={<Title level={4} style={{ margin: 0 }}>借贷管理</Title>} extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => openEditModal(null)}>新增借贷</Button>} />
            <Card>
                <Table 
                    columns={getFinalColumns()} 
                    dataSource={loans} 
                    rowKey="id" 
                    loading={loading} 
                    components={{ body: { row: MotionRow } }} 
                    scroll={{ x: 'max-content' }}
                />
            </Card>
            <Modal title={editingLoan ? "编辑借贷" : "新增借贷"} open={isEditModalOpen} onOk={editForm.submit} onCancel={handleCancel} destroyOnHidden>
                <Form form={editForm} layout="vertical" onFinish={handleEditFormSubmit}>
                    <Form.Item name="description" label="描述" rules={[{ required: true }]}><Input /></Form.Item>
                    <Form.Item name="principal" label="本金" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} prefix="¥" min={0} precision={2} /></Form.Item>
                    <Form.Item name="interest_rate" label="年利率(%)" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} suffix="%" min={0} /></Form.Item>
                    <Form.Item name="loan_date" label="借款日期" rules={[{ required: true }]}><DatePicker style={{ width: '100%' }} /></Form.Item>
                    <Form.Item name="repayment_date" label="计划还款日期 (选填)" dependencies={['loan_date']} rules={[({ getFieldValue }) => ({ validator(_, value) { if (!value || !getFieldValue('loan_date') || !value.isBefore(getFieldValue('loan_date'))) { return Promise.resolve(); } return Promise.reject(new Error('还款日期不能早于借款日期！')); }, }), ]}><DatePicker style={{ width: '100%' }} /></Form.Item>
                </Form>
            </Modal>
            <Modal title="确认还清贷款" open={isSettleModalOpen} onOk={settleForm.submit} onCancel={handleCancel} destroyOnHidden>
                <Form form={settleForm} layout="vertical" onFinish={handleSettleFormSubmit}>
                    <p>您正在为贷款“<Text strong>{settlingLoan?.description || `贷款 #${settlingLoan?.id}`}</Text>”进行结算。</p>
                    <p>待还清金额: <Text type="danger" strong>¥{settlingLoan?.outstanding_balance.toFixed(2)}</Text></p>
                    <Form.Item name="from_account_id" label="扣款账户" rules={[{ required: true, message: '请选择扣款账户' }]}><Select placeholder="选择资金来源账户">{accounts.map(acc => <Select.Option key={acc.id} value={acc.id}>{`${acc.name} (余额: ¥${acc.balance.toFixed(2)})`}</Select.Option>)}</Select></Form.Item>
                    <Form.Item name="repayment_date" label="还款日期" rules={[{ required: true }]}><DatePicker style={{ width: '100%' }} /></Form.Item>
                    <Form.Item name="description" label="备注"><Input.TextArea rows={2} /></Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

export default LoanPage;