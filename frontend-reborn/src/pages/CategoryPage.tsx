// src/pages/CategoryPage.tsx
import React, { useState, useEffect, useCallback } from 'react';
import { Button, Card, Table, Tag, message, Modal, Form, Input, Radio, Space, Popconfirm, Select, Tooltip, Typography } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { getCategories, createCategory, updateCategory, deleteCategory } from '../services/api';
import type { Category } from '../types';
import type { ColumnsType } from 'antd/es/table';
import IconDisplay, { availableIcons } from '../components/IconPicker';
import { customAlphabet } from 'nanoid';
import axios from 'axios';
import { motion } from 'framer-motion';

const nanoid = customAlphabet('abcdefghijklmnopqrstuvwxyz0123456789', 10);
const { Title } = Typography;

const MotionRow = (props: any) => (
    <motion.tr {...props} initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3 }} />
);

const CategoryPage: React.FC = () => {
    const [categories, setCategories] = useState<Category[]>([]);
    const [loading, setLoading] = useState(true);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [editingCategory, setEditingCategory] = useState<Category | null>(null);
    const [form] = Form.useForm();

    const fetchCategories = useCallback(async () => {
        setLoading(true);
        try {
            const res = await getCategories();
            setCategories(res.data || []);
        } catch (error) {
            message.error('获取分类列表失败');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchCategories();
    }, [fetchCategories]);

    const handleCancel = () => { setIsModalOpen(false); setEditingCategory(null); form.resetFields(); };

    const handleFormSubmit = async (values: any) => {
        setIsSubmitting(true);
        try {
            if (editingCategory) {
                await updateCategory(editingCategory.id, { name: values.name, icon: values.icon });
                message.success('分类更新成功！');
            } else {
                await createCategory({ id: `user_${nanoid()}`, ...values });
                message.success('分类添加成功！');
            }
            handleCancel(); fetchCategories();
        } catch (error: unknown) {
             message.error(axios.isAxiosError(error) ? error.response?.data.error : '操作失败');
        } finally { setIsSubmitting(false); }
    };

    const openModal = (category: Category | null) => {
        setEditingCategory(category);
        form.setFieldsValue(category || { type: 'expense', icon: 'Archive' });
        setIsModalOpen(true);
    };

    const handleDelete = async (id: string) => {
        try {
            await deleteCategory(id);
            message.success('分类删除成功！'); fetchCategories();
        } catch (error: unknown) { message.error(axios.isAxiosError(error) ? error.response?.data.error : '删除失败'); }
    };
    
    const columns: ColumnsType<Category> = [
        { title: '分类名称', dataIndex: 'name', key: 'name', render: (name: string, record: Category) => (<Space><IconDisplay name={record.icon} size={20} /><span>{name}</span></Space>) },
        { title: '类型', dataIndex: 'type', key: 'type', filters: [{ text: '支出', value: 'expense' }, { text: '收入', value: 'income' }, { text: '内部', value: 'internal' }], onFilter: (value, record) => record.type === value, render: (type: string) => { let color; if (type === 'income') color = 'success'; else if (type === 'expense') color = 'error'; else color = 'processing'; return <Tag color={color}>{type.toUpperCase()}</Tag> } },
        { title: '创建时间', dataIndex: 'created_at', key: 'created_at', responsive: ['md'], render: (text: string) => new Date(text).toLocaleString(), sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(), },
        { title: '操作', key: 'action', width: 120, align: 'center', render: (_, record) => { const isPreset = !record.id.startsWith('user_'); return (<Space><Tooltip title={isPreset ? "预设分类不可编辑" : "编辑"}><Button type="text" icon={<EditOutlined />} onClick={() => openModal(record)} disabled={isPreset} /></Tooltip><Tooltip title={isPreset ? "预设分类不可删除" : "删除"}><Popconfirm title="确定删除吗？" description="此操作不可恢复" onConfirm={() => handleDelete(record.id)} disabled={isPreset}><Button type="text" icon={<DeleteOutlined />} danger disabled={isPreset} /></Popconfirm></Tooltip></Space>); }, },
    ];

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card title={<Title level={4} style={{ margin: 0 }}>分类管理</Title>} extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => openModal(null)}>新增分类</Button>} />
            <Card>
                <Table columns={columns} dataSource={categories} rowKey="id" loading={loading} pagination={{ pageSize: 10, showSizeChanger: true }} components={{ body: { row: MotionRow } }} scroll={{ x: 'max-content' }} />
            </Card>
            
            <Modal title={editingCategory ? '编辑分类' : '新增分类'} open={isModalOpen} onOk={form.submit} onCancel={handleCancel} confirmLoading={isSubmitting} destroyOnHidden>
                <Form form={form} layout="vertical" onFinish={handleFormSubmit} initialValues={{ type: 'expense' }}>
                    <Form.Item name="name" label="分类名称" rules={[{ required: true, message: '请输入分类名称' }]}><Input placeholder="例如：早餐、地铁" /></Form.Item>
                     <Form.Item name="type" label="类型" rules={[{ required: true }]}><Radio.Group disabled={!!editingCategory}><Radio.Button value="expense">支出</Radio.Button><Radio.Button value="income">收入</Radio.Button></Radio.Group></Form.Item>
                    <Form.Item name="icon" label="图标" rules={[{ required: true, message: '请选择一个图标'}]}><Select showSearch optionFilterProp="children" placeholder="选择一个图标">{availableIcons.map(iconName => (<Select.Option key={iconName} value={iconName}><Space><IconDisplay name={iconName} /> {iconName}</Space></Select.Option>))}</Select></Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

export default CategoryPage;