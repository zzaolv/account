// src/pages/BudgetsPage.tsx
import React, { useState, useEffect, useCallback } from 'react';
import { Button, Card, message, Modal, Form, InputNumber, Radio, Select, Space, Popconfirm, Empty, Tooltip, Progress, Row, Col, Spin, Typography, DatePicker } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { getBudgets, createOrUpdateBudget, deleteBudget, getCategories } from '../services/api';
import type { Budget, Category } from '../types';
import axios from 'axios';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

const { Title, Text } = Typography;

// 预算卡片组件
const BudgetCard: React.FC<{ budget: Budget; onEdit: (budget: Budget) => void; onDelete: (id: number) => void; }> = ({ budget, onEdit, onDelete }) => {
    const periodText = budget.period === 'monthly' 
        ? `${budget.year}年 ${budget.month}月` 
        : `${budget.year}年`;
    const title = budget.category_name || `全局预算`;
    return (
        <Card title={title} extra={<Text type="secondary">{periodText}</Text>} actions={[ <Tooltip title="编辑" key="edit"><EditOutlined onClick={() => onEdit(budget)} /></Tooltip>, <Popconfirm key="delete" title="确定删除吗？" onConfirm={() => onDelete(budget.id)} okText="确定" cancelText="取消"> <Tooltip title="删除"><DeleteOutlined /></Tooltip> </Popconfirm> ]} >
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

const BudgetPage: React.FC = () => {
    const [budgets, setBudgets] = useState<Budget[]>([]);
    const [categories, setCategories] = useState<Category[]>([]);
    const [loading, setLoading] = useState(true);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [editingBudget, setEditingBudget] = useState<Budget | null>(null);
    const [form] = Form.useForm();
    const [filterDate, setFilterDate] = useState<Dayjs>(dayjs());

    const fetchData = useCallback(async () => {
        setLoading(true);
        try {
            const params = { year: filterDate.year(), month: filterDate.month() + 1 };
            const [budgetRes, categoryRes] = await Promise.all([getBudgets(params), getCategories()]);
            setBudgets(budgetRes.data || []);
            setCategories(categoryRes.data.filter(c => c.type === 'expense') || []);
        } catch (error) { message.error('获取数据失败'); } finally { setLoading(false); }
    }, [filterDate]);

    useEffect(() => { fetchData(); }, [fetchData]);

    // 【修正】补全所有函数实现
    const handleCancel = () => {
        setIsModalOpen(false);
        setEditingBudget(null);
        form.resetFields();
    };

    const handleFormSubmit = async (values: any) => {
        const postData = { ...values, category_id: values.category_id === 'global' ? null : values.category_id, amount: parseFloat(values.amount) };
        try {
            await createOrUpdateBudget(postData);
            message.success('预算保存成功！');
            handleCancel();
            await fetchData();
        } catch (error) {
            if (axios.isAxiosError(error) && error.response) { message.error(error.response.data.error || '操作失败');
            } else { message.error('操作失败'); }
        }
    };
    
    const openModal = (budget: Budget | null) => {
        setEditingBudget(budget);
        if (budget) { form.setFieldsValue({ ...budget, category_id: budget.category_id || 'global' });
        } else { form.resetFields(); }
        setIsModalOpen(true);
    };
    
    const handleDelete = async (id: number) => {
        try {
            await deleteBudget(id);
            message.success('删除成功！');
            await fetchData();
        } catch (error) { message.error('删除失败'); }
    };

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
             <Card> <Row justify="space-between" align="middle"> <Col> <Space> <Title level={2} style={{ margin: 0 }}>预算规划</Title> <DatePicker picker="month" value={filterDate} onChange={(date) => setFilterDate(date || dayjs())} /> </Space> </Col> <Col> <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal(null)}> 新增/修改预算 </Button> </Col> </Row> </Card>
            <Spin spinning={loading}>
                {budgets.length > 0 ? ( <Row gutter={[16, 16]}> {budgets.map(budget => ( <Col key={budget.id} xs={24} sm={12} md={8} lg={6}> <BudgetCard budget={budget} onEdit={openModal} onDelete={handleDelete} /> </Col> ))} </Row> ) : ( <Card><Empty description="暂无预算定义，请先新增。" /></Card> )}
            </Spin>
            <Modal title={editingBudget ? '编辑预算' : '新增预算'} open={isModalOpen} onOk={form.submit} onCancel={handleCancel} destroyOnClose>
                <Form form={form} layout="vertical" onFinish={handleFormSubmit} initialValues={{ period: 'monthly', category_id: 'global' }}>
                    <Form.Item name="period" label="周期" rules={[{ required: true }]}><Radio.Group disabled={!!editingBudget}><Radio.Button value="monthly">月度</Radio.Button><Radio.Button value="yearly">年度</Radio.Button></Radio.Group></Form.Item>
                    <Form.Item name="category_id" label="预算分类" rules={[{ required: true }]}><Select disabled={!!editingBudget}><Select.Option value="global">全局预算（所有支出）</Select.Option>{categories.map(cat => (<Select.Option key={cat.id} value={cat.id}>{cat.name}</Select.Option>))}</Select></Form.Item>
                    <Form.Item name="amount" label="预算金额" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} /></Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

export default BudgetPage;