// src/pages/LoanPage.tsx
import React, { useState } from 'react';
import { Button, Card, Table, Tag, Modal, Form, Input, InputNumber, DatePicker, Space, Popconfirm, Select, Typography, Grid, Dropdown, App, Skeleton, Result, Empty, notification } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined, UndoOutlined, MoreOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getLoans, createLoan, updateLoan, deleteLoan, updateLoanStatus, getAccounts, settleLoan } from '../services/api';
import type { LoanResponse, UpdateLoanRequest, Account, SettleLoanRequest } from '../types';
import type { ColumnsType } from 'antd/es/table';
import type { MenuProps } from 'antd';
import dayjs from 'dayjs';
import axios from 'axios';
import { motion } from 'framer-motion';

const { Text, Title } = Typography;
const { useBreakpoint } = Grid;

const MotionRow = (props: any) => (
    <motion.tr {...props} initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3 }} />
);

const FormattedInputNumber: React.FC<any> = (props) => (
    <InputNumber {...props} formatter={(value) => `${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')} parser={(value) => (value ? value.replace(/\$\s?|(,*)/g, '') : '')} />
);

const LoanPageContent: React.FC = () => {
    const [isEditModalOpen, setIsEditModalOpen] = useState(false);
    const [isSettleModalOpen, setIsSettleModalOpen] = useState(false);
    const [editingLoan, setEditingLoan] = useState<LoanResponse | null>(null);
    const [settlingLoan, setSettlingLoan] = useState<LoanResponse | null>(null);
    const [editForm] = Form.useForm();
    const [settleForm] = Form.useForm();
    
    const screens = useBreakpoint();
    const isMobile = !screens.md;
    const queryClient = useQueryClient();
    const { message } = App.useApp();

    const { data: loans = [], isLoading: isLoadingLoans, isError: isErrorLoans, error: errorLoans, refetch: refetchLoans } = useQuery<LoanResponse[], Error>({
        queryKey: ['loans'],
        queryFn: () => getLoans().then(res => res.data || []),
        retry: 1,
    });

    const { data: accounts = [], isLoading: isLoadingAccounts } = useQuery<Account[], Error>({
        queryKey: ['accounts'],
        queryFn: () => getAccounts().then(res => res.data || []),
        retry: 1,
    });

    const handleMutationError = (err: unknown, title: string) => {
        notification.error({
            message: title,
            description: axios.isAxiosError(err) ? err.response?.data.error : '发生未知错误',
        });
    };
    
    const invalidateQueries = () => {
        queryClient.invalidateQueries({ queryKey: ['loans'] });
        queryClient.invalidateQueries({ queryKey: ['transactions'] });
        queryClient.invalidateQueries({ queryKey: ['accounts'] });
        queryClient.invalidateQueries({ queryKey: ['dashboardWidgets'] });
    };

    const createMutation = useMutation({
        mutationFn: (data: UpdateLoanRequest) => createLoan(data),
        onSuccess: () => { message.success('借贷记录添加成功！'); invalidateQueries(); handleCancel(); },
        onError: (err) => handleMutationError(err, '添加失败'),
    });

    const updateMutation = useMutation({
        mutationFn: (vars: { id: number, data: UpdateLoanRequest }) => updateLoan(vars.id, vars.data),
        onSuccess: () => { message.success('借贷记录更新成功！'); invalidateQueries(); handleCancel(); },
        onError: (err) => handleMutationError(err, '更新失败'),
    });

    const settleMutation = useMutation({
        mutationFn: (vars: { id: number, data: SettleLoanRequest }) => settleLoan(vars.id, vars.data),
        onSuccess: () => { message.success('贷款已成功还清！'); invalidateQueries(); handleCancel(); },
        onError: (err) => handleMutationError(err, '还清操作失败'),
    });

    const restoreMutation = useMutation({
        mutationFn: (id: number) => updateLoanStatus(id, 'active'),
        onSuccess: () => { message.success('状态已恢复为 "活动中"！'); invalidateQueries(); },
        onError: (err) => handleMutationError(err, '恢复失败'),
    });

    const deleteMutation = useMutation({
        mutationFn: (id: number) => deleteLoan(id),
        onSuccess: () => { message.success('删除成功！'); invalidateQueries(); },
        onError: (err) => handleMutationError(err, '删除失败'),
    });

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

    const handleEditFormSubmit = (values: any) => {
        const postData: UpdateLoanRequest = { ...values, interest_rate: parseFloat(values.interest_rate) / 100, loan_date: values.loan_date.format('YYYY-MM-DD'), repayment_date: values.repayment_date ? values.repayment_date.format('YYYY-MM-DD') : null };
        if (editingLoan) { updateMutation.mutate({ id: editingLoan.id, data: postData }); } else { createMutation.mutate(postData); }
    };

    const handleSettleFormSubmit = (values: any) => {
        if (!settlingLoan) return;
        const postData: SettleLoanRequest = { ...values, repayment_date: values.repayment_date.format('YYYY-MM-DD') };
        settleMutation.mutate({ id: settlingLoan.id, data: postData });
    };

    // 【关键修改】修复 Dropdown menu 类型
    const getActionMenuItems = (record: LoanResponse): MenuProps => {
        const isMutating = deleteMutation.isPending || restoreMutation.isPending || updateMutation.isPending;
        const items = [
            { key: 'edit', icon: <EditOutlined />, label: '编辑', disabled: isMutating, onClick: () => openEditModal(record) },
            record.status === 'active' && { key: 'settle', icon: <CheckCircleOutlined />, label: '还清', disabled: isMutating, onClick: () => openSettleModal(record) },
            record.status === 'paid' && { key: 'restore', icon: <UndoOutlined />, label: <Popconfirm title="确定要恢复为“活动中”吗？" onConfirm={() => restoreMutation.mutate(record.id)} disabled={isMutating}>恢复</Popconfirm>, disabled: isMutating },
            { key: 'delete', danger: true, icon: <DeleteOutlined />, label: <Popconfirm title="确定删除这笔借款吗?" description="有关联还款记录的无法删除" onConfirm={() => deleteMutation.mutate(record.id)} disabled={isMutating}>删除</Popconfirm>, disabled: isMutating }
        ];
        return { items: items.filter(Boolean) as MenuProps['items'] }; // 使用 filter(Boolean) 过滤掉 false
    };

    const columns: ColumnsType<LoanResponse> = [
        { title: '描述', dataIndex: 'description', key: 'description' },
        { title: '本金', dataIndex: 'principal', key: 'principal', responsive: ['md'], render: (amount: number) => `¥${amount.toFixed(2)}` },
        { title: '待还余额', dataIndex: 'outstanding_balance', key: 'outstanding_balance', render: (amount: number) => <Text type="danger" strong>¥{amount.toFixed(2)}</Text> },
        { title: '借款日期', dataIndex: 'loan_date', key: 'loan_date', responsive: ['lg'] },
        { title: '状态', dataIndex: 'status', key: 'status', responsive: ['sm'], render: (status: string) => <Tag color={status === 'active' ? 'processing' : 'success'}>{status === 'active' ? '活动中' : '已还清'}</Tag> },
        { title: '操作', key: 'action', align: 'center', width: isMobile ? 60 : 220, render: (_, record: LoanResponse) => {
            const isDeleting = deleteMutation.isPending && deleteMutation.variables === record.id;
            const isRestoring = restoreMutation.isPending && restoreMutation.variables === record.id;
            const isMutatingAny = deleteMutation.isPending || restoreMutation.isPending || updateMutation.isPending || settleMutation.isPending;

            if (isMobile) {
                return <Dropdown menu={getActionMenuItems(record)} trigger={['click']} disabled={isMutatingAny}><Button type="text" icon={<MoreOutlined />} loading={isRestoring || isDeleting} /></Dropdown>;
            }
            return (
                <Space>
                    <Button type="link" icon={<EditOutlined />} onClick={() => openEditModal(record)} disabled={isMutatingAny}>编辑</Button>
                    {record.status === 'active' && (<Button type="link" icon={<CheckCircleOutlined />} onClick={() => openSettleModal(record)} disabled={isMutatingAny}>还清</Button>)}
                    {record.status === 'paid' && (<Popconfirm title="确定要恢复为“活动中”吗？" onConfirm={() => restoreMutation.mutate(record.id)} disabled={isMutatingAny}><Button type="link" icon={<UndoOutlined />} loading={isRestoring}>恢复</Button></Popconfirm>)}
                    <Popconfirm title="确定删除这笔借款吗?" description="有关联还款记录的无法删除" onConfirm={() => deleteMutation.mutate(record.id)} disabled={isMutatingAny}><Button type="link" danger icon={<DeleteOutlined />} loading={isDeleting}>删除</Button></Popconfirm>
                </Space>
            );
        }},
    ];
    
    const getFinalColumns = () => {
        if (!isMobile) return columns;
        return columns.filter(col => ['description', 'outstanding_balance', 'action'].includes(String(col.key)));
    };

    const renderMainContent = () => {
        if (isLoadingLoans) return <Card><Skeleton active paragraph={{ rows: 5 }} /></Card>;
        if (isErrorLoans) return <Card><Result status="error" title="借贷数据加载失败" subTitle={`错误: ${errorLoans.message}`} extra={<Button type="primary" onClick={() => refetchLoans()}>点击重试</Button>} /></Card>;
        
        // 【关键修改】修复 Empty 组件用法
        if (loans.length === 0) return (
            <Card>
                <Empty description={
                    <span>
                        暂无借贷记录
                        <br/>
                        <Button type="primary" onClick={() => openEditModal(null)} style={{ marginTop: 16 }}>新增一笔</Button>
                    </span>
                } />
            </Card>
        );

        return (
            <Card>
                <Table columns={getFinalColumns()} dataSource={loans} rowKey="id" components={{ body: { row: MotionRow } }} scroll={{ x: 'max-content' }} />
            </Card>
        );
    };

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card title={<Title level={4} style={{ margin: 0 }}>借贷管理</Title>} extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => openEditModal(null)}>新增借贷</Button>} />
            {renderMainContent()}
            <Modal title={editingLoan ? "编辑借贷" : "新增借贷"} open={isEditModalOpen} onOk={editForm.submit} onCancel={handleCancel} destroyOnClose confirmLoading={createMutation.isPending || updateMutation.isPending}>
                <Form form={editForm} layout="vertical" onFinish={handleEditFormSubmit}>
                    <Form.Item name="description" label="描述" rules={[{ required: true }]}><Input /></Form.Item>
                    <Form.Item name="principal" label="本金" rules={[{ required: true }]}><FormattedInputNumber style={{ width: '100%' }} prefix="¥" min={0} precision={2} /></Form.Item>
                    <Form.Item name="interest_rate" label="年利率(%)" rules={[{ required: true }]}><FormattedInputNumber style={{ width: '100%' }} suffix="%" min={0} /></Form.Item>
                    <Form.Item name="loan_date" label="借款日期" rules={[{ required: true }]}><DatePicker style={{ width: '100%' }} /></Form.Item>
                    <Form.Item name="repayment_date" label="计划还款日期 (选填)" dependencies={['loan_date']} rules={[({ getFieldValue }) => ({ validator(_, value) { if (!value || !getFieldValue('loan_date') || !value.isBefore(getFieldValue('loan_date'))) { return Promise.resolve(); } return Promise.reject(new Error('还款日期不能早于借款日期！')); }, }), ]}><DatePicker style={{ width: '100%' }} /></Form.Item>
                </Form>
            </Modal>
            <Modal title="确认还清贷款" open={isSettleModalOpen} onOk={settleForm.submit} onCancel={handleCancel} destroyOnClose confirmLoading={settleMutation.isPending}>
                <Form form={settleForm} layout="vertical" onFinish={handleSettleFormSubmit}>
                    <p>您正在为贷款“<Text strong>{settlingLoan?.description || `贷款 #${settlingLoan?.id}`}</Text>”进行结算。</p>
                    <p>待还清金额: <Text type="danger" strong>¥{settlingLoan?.outstanding_balance.toFixed(2)}</Text></p>
                    <Form.Item name="from_account_id" label="扣款账户" rules={[{ required: true, message: '请选择扣款账户' }]}><Select placeholder="选择资金来源账户" loading={isLoadingAccounts}>{accounts.map(acc => <Select.Option key={acc.id} value={acc.id}>{`${acc.name} (余额: ¥${acc.balance.toFixed(2)})`}</Select.Option>)}</Select></Form.Item>
                    <Form.Item name="repayment_date" label="还款日期" rules={[{ required: true }]}><DatePicker style={{ width: '100%' }} /></Form.Item>
                    <Form.Item name="description" label="备注"><Input.TextArea rows={2} /></Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

const LoanPage: React.FC = () => (
    <App>
        <LoanPageContent />
    </App>
);

export default LoanPage;