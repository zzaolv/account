// frontend-reborn/src/pages/BudgetsPage.tsx

import React, { useState, useMemo } from 'react';
import { Button, Card, Modal, Form, InputNumber, Radio, Select, Space, Popconfirm, Empty, Tooltip, Progress, Row, Col, Spin, Typography, DatePicker, App } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getBudgets, createOrUpdateBudget, deleteBudget, getCategories } from '../services/api';
import type { Budget, Category, CreateOrUpdateBudgetRequest } from '../types';
import axios from 'axios';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

const { Title, Text } = Typography;

const BudgetCard: React.FC<{ budget: Budget; onEdit: (budget: Budget) => void; onDelete: (id: number) => void; isMutating: boolean; mutatingId: number | null }> = ({ budget, onEdit, onDelete, isMutating, mutatingId }) => {
    const periodText = budget.period === 'monthly' ? `${budget.year}年 ${budget.month}月` : `${budget.year}年`;
    const title = budget.category_name || `全局预算`;
    const isLoading = isMutating && mutatingId === budget.id;

    return (
        <Card 
            title={title} 
            extra={<Text type="secondary">{periodText}</Text>} 
            loading={isLoading}
            actions={[
                <Tooltip title="编辑" key="edit"><EditOutlined onClick={() => onEdit(budget)} /></Tooltip>,
                <Popconfirm key="delete" title="确定删除吗？" onConfirm={() => onDelete(budget.id)} okText="确定" cancelText="取消">
                    <Tooltip title="删除"><DeleteOutlined /></Tooltip>
                </Popconfirm>
            ]}
        >
            <Space direction="vertical" style={{ width: '100%' }}>
                <Text>预算金额</Text>
                <Title level={3} style={{ marginTop: 0 }}>¥{budget.amount.toFixed(2)}</Title>
                <Tooltip title={`已用: ¥${budget.spent.toFixed(2)} / 剩余: ¥${budget.remaining.toFixed(2)}`}>
                    <Progress percent={Math.round(budget.progress * 100)} status={budget.progress > 1 ? 'exception' : 'normal'} />
                </Tooltip>
            </Space>
        </Card>
    );
};

const BudgetPageContent: React.FC = () => {
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [editingBudget, setEditingBudget] = useState<Budget | null>(null);
    const [form] = Form.useForm();
    const [filterDate, setFilterDate] = useState<Dayjs>(dayjs());
    const queryClient = useQueryClient();
    const { message } = App.useApp();

    const apiFilter = useMemo(() => ({
        year: filterDate.year(),
        month: filterDate.month() + 1
    }), [filterDate]);

    const { data: budgets, isLoading: isLoadingBudgets } = useQuery<Budget[], Error>({
        queryKey: ['budgets', apiFilter],
        queryFn: () => getBudgets(apiFilter).then(res => res.data || [])
    });

    const { data: categories, isLoading: isLoadingCategories } = useQuery<Category[], Error>({
        queryKey: ['categories', 'expense'],
        queryFn: () => getCategories().then(res => (res.data || []).filter(c => c.type === 'expense')),
    });

    const mutationOptions = (successMsg: string) => ({
        onSuccess: () => {
            message.success(successMsg);
            queryClient.invalidateQueries({ queryKey: ['budgets', apiFilter] }); // 精确失效
            queryClient.invalidateQueries({ queryKey: ['dashboardWidgets'] });
            handleCancel();
        },
        onError: (err: unknown) => {
            const errorMsg = axios.isAxiosError(err) && err.response ? err.response.data.error : '操作失败';
            message.error(errorMsg);
        },
    });

    const saveMutation = useMutation({
        mutationFn: (data: CreateOrUpdateBudgetRequest) => createOrUpdateBudget(data),
        ...mutationOptions('预算保存成功！'),
    });

    const deleteMutation = useMutation<void, Error, number>({
        mutationFn: async (id) => { await deleteBudget(id); },
        ...mutationOptions('预算删除成功！'),
    });

    const handleCancel = () => {
        setIsModalOpen(false);
        setEditingBudget(null);
        form.resetFields();
    };

    const handleFormSubmit = (values: any) => {
        // 【核心修正】提交时附带当前选择的 year 和 month
        const postData: CreateOrUpdateBudgetRequest = {
            ...values,
            category_id: values.category_id === 'global' ? null : values.category_id,
            amount: parseFloat(values.amount),
            year: filterDate.year(),
            month: values.period === 'monthly' ? filterDate.month() + 1 : undefined
        };
        saveMutation.mutate(postData);
    };

    const openModal = (budget: Budget | null) => {
        setEditingBudget(budget);
        if (budget) {
            form.setFieldsValue({ ...budget, category_id: budget.category_id || 'global' });
        } else {
            // 新增时，默认周期为月度
            form.resetFields();
            form.setFieldsValue({ period: 'monthly' });
        }
        setIsModalOpen(true);
    };

    const isLoading = isLoadingBudgets || isLoadingCategories;

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card>
                <Row justify="space-between" align="middle">
                    <Col>
                        <Space>
                            <Title level={4} style={{ margin: 0 }}>预算规划</Title>
                            <DatePicker picker="month" value={filterDate} onChange={(date) => setFilterDate(date || dayjs())} allowClear={false} />
                        </Space>
                    </Col>
                    <Col>
                        <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal(null)}>
                            新增/修改预算
                        </Button>
                    </Col>
                </Row>
            </Card>

            <Spin spinning={isLoading}>
                {!isLoading && (!budgets || budgets.length === 0) ? (
                    <Card><Empty description="当前周期暂无预算定义，请先新增。" /></Card>
                ) : (
                    <Row gutter={[16, 16]}>
                        {budgets?.map(budget => (
                            <Col key={budget.id} xs={24} sm={12} md={8} lg={6}>
                                <BudgetCard
                                    budget={budget}
                                    onEdit={openModal}
                                    onDelete={deleteMutation.mutate}
                                    isMutating={deleteMutation.isPending}
                                    mutatingId={deleteMutation.variables ?? null}
                                />
                            </Col>
                        ))}
                    </Row>
                )}
            </Spin>

            <Modal 
                title={editingBudget ? '编辑预算' : `为 ${filterDate.format('YYYY年MM月')} 新增预算`} 
                open={isModalOpen} 
                onOk={form.submit} 
                onCancel={handleCancel} 
                destroyOnClose
                confirmLoading={saveMutation.isPending}
            >
                <Form form={form} layout="vertical" onFinish={handleFormSubmit} initialValues={{ period: 'monthly' }}>
                    {/* 用户在新增时只能为当前选定月份/年份创建预算，所以周期和时间选择器可以禁用或隐藏 */}
                    <Form.Item name="period" label="周期" rules={[{ required: true }]}>
                        <Radio.Group disabled={!!editingBudget}>
                            <Radio.Button value="monthly">月度</Radio.Button>
                            <Radio.Button value="yearly">年度</Radio.Button>
                        </Radio.Group>
                    </Form.Item>
                    <Form.Item name="category_id" label="预算分类" rules={[{ required: true }]}>
                        <Select disabled={!!editingBudget} loading={isLoadingCategories}>
                            <Select.Option value="global">全局预算（所有支出）</Select.Option>
                            {categories?.map(cat => (
                                <Select.Option key={cat.id} value={cat.id}>{cat.name}</Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                    <Form.Item name="amount" label="预算金额" rules={[{ required: true }]}>
                        <InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} />
                    </Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

const BudgetPage: React.FC = () => (
    <App>
        <BudgetPageContent />
    </App>
);

export default BudgetPage;