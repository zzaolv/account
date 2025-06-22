// src/pages/LoanPage.tsx
import React, { useState, useEffect, useCallback } from 'react';
import { Button, Card, Table, Tag, message, Modal, Form, Input, InputNumber, DatePicker, Space, Popconfirm } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { getLoans, createLoan, updateLoan, deleteLoan, updateLoanStatus } from '../services/api';
import type { LoanResponse, UpdateLoanRequest } from '../types';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import axios from 'axios';

const LoanPage: React.FC = () => {
    const [loans, setLoans] = useState<LoanResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [editingLoan, setEditingLoan] = useState<LoanResponse | null>(null);
    const [form] = Form.useForm();

    const fetchLoans = useCallback(async () => {
        setLoading(true);
        try {
            const res = await getLoans();
            setLoans(res.data || []);
        } catch (error) {
            message.error('获取借贷列表失败');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchLoans();
    }, [fetchLoans]);

    const handleCancel = () => {
        setIsModalOpen(false);
        setEditingLoan(null);
        form.resetFields();
    };

    const handleFormSubmit = async (values: any) => {
        const postData: UpdateLoanRequest = {
            description: values.description,
            principal: parseFloat(values.principal),
            interest_rate: parseFloat(values.interest_rate) / 100,
            loan_date: values.loan_date.format('YYYY-MM-DD'),
            repayment_date: values.repayment_date ? values.repayment_date.format('YYYY-MM-DD') : null,
        };

        try {
            if (editingLoan) {
                await updateLoan(editingLoan.id, postData);
                message.success('借贷记录更新成功！');
            } else {
                await createLoan(postData);
                message.success('借贷记录添加成功！');
            }
            handleCancel();
            await fetchLoans();
        } catch (error) {
            if (axios.isAxiosError(error) && error.response) {
                message.error(error.response.data.error || '操作失败');
            } else {
                message.error('操作失败');
            }
        }
    };
    
    const openModal = (loan: LoanResponse | null) => {
        setEditingLoan(loan);
        if (loan) {
            form.setFieldsValue({
                ...loan,
                loan_date: dayjs(loan.loan_date),
                repayment_date: loan.repayment_date ? dayjs(loan.repayment_date) : null,
                interest_rate: loan.interest_rate * 100,
            });
        } else {
            form.resetFields();
        }
        setIsModalOpen(true);
    };

    const handleUpdateStatus = async (id: number, status: 'paid' | 'active') => {
        try {
            await updateLoanStatus(id, status);
            message.success('状态更新成功！');
            await fetchLoans();
        } catch (error) {
            message.error('更新失败');
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await deleteLoan(id);
            message.success('删除成功！');
            await fetchLoans();
        } catch (error: unknown) {
            if (axios.isAxiosError(error) && error.response) {
                message.error(error.response.data.error || '删除失败');
            } else {
                message.error('删除失败，发生未知错误');
            }
        }
    };

    const columns: ColumnsType<LoanResponse> = [
        { title: '描述', dataIndex: 'description', key: 'description' },
        { title: '本金', dataIndex: 'principal', key: 'principal', render: (amount: number) => `¥${amount.toFixed(2)}` },
        { title: '待还余额', dataIndex: 'outstanding_balance', key: 'outstanding_balance', render: (amount: number) => `¥${amount.toFixed(2)}` },
        { title: '借款日期', dataIndex: 'loan_date', key: 'loan_date' },
        { title: '计划还款日', dataIndex: 'repayment_date', key: 'repayment_date', render: (date: string | null) => date || '-' },
        { title: '状态', dataIndex: 'status', key: 'status', render: (status: string) => <Tag color={status === 'active' ? 'processing' : 'success'}>{status === 'active' ? '活动中' : '已还清'}</Tag> },
        {
            title: '操作',
            key: 'action',
            render: (_, record: LoanResponse) => ( // 【修正】添加类型
                <Space>
                    <Button type="link" icon={<EditOutlined />} onClick={() => openModal(record)}>编辑</Button>
                    {record.status === 'active' && (
                        // 【修正】确保 onConfirm 正确绑定
                        <Popconfirm title="确定标记为已还清吗?" onConfirm={() => handleUpdateStatus(record.id, 'paid')} okText="确定" cancelText="取消">
                            <Button type="link" icon={<CheckCircleOutlined />}>还清</Button>
                        </Popconfirm>
                    )}

                    <Popconfirm title="确定删除这笔借款吗?" description="有关联还款记录的无法删除" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
                        <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
                    </Popconfirm>
                </Space>
            ),
        },
    ];

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card title="借贷管理" extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => openModal(null)}>新增借贷</Button>} />
            <Card><Table columns={columns} dataSource={loans} rowKey="id" loading={loading} /></Card>
            <Modal title={editingLoan ? "编辑借贷" : "新增借贷"} open={isModalOpen} onOk={form.submit} onCancel={handleCancel} destroyOnClose>
                <Form form={form} layout="vertical" onFinish={handleFormSubmit}>
                    <Form.Item name="description" label="描述" rules={[{ required: true }]}><Input /></Form.Item>
                    <Form.Item name="principal" label="本金" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} prefix="¥" min={0} precision={2} /></Form.Item>
                    <Form.Item name="interest_rate" label="年利率(%)" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} suffix="%" min={-1} /></Form.Item>
                    <Form.Item name="loan_date" label="借款日期" rules={[{ required: true }]}><DatePicker style={{ width: '100%' }} /></Form.Item>
                    <Form.Item
                        name="repayment_date"
                        label="计划还款日期 (选填)"
                        dependencies={['loan_date']} // 依赖于 loan_date 字段
                        rules={[
                            // validator 是一个返回 Promise 的函数
                            ({ getFieldValue }) => ({
                                validator(_, value) {
                                    const loanDate = getFieldValue('loan_date');
                                    // 如果没有设置还款日期或借款日期，则通过校验
                                    if (!value || !loanDate) {
                                        return Promise.resolve();
                                    }
                                    // 如果还款日期在借款日期之前，则校验失败
                                    if (value.isBefore(loanDate)) {
                                        return Promise.reject(new Error('还款日期不能早于借款日期！'));
                                    }
                                    return Promise.resolve();
                                },
                            }),
                        ]}
                    >
                        <DatePicker style={{ width: '100%' }} />
                    </Form.Item>
                    {/* ============================================================ */}
                </Form>
            </Modal>
        </Space>
    );
};

export default LoanPage;