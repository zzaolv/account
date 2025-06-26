// src/pages/CategoryPage.tsx
import React, { useState } from 'react';
import { Button, Card, Table, Tag, Modal, Form, Input, Radio, Space, Popconfirm, Select, Tooltip, Typography, App } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ShareAltOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getCategories, createCategory, updateCategory, deleteCategory } from '../services/api';
import type { Category, CreateCategoryRequest } from '../types';
import type { ColumnsType } from 'antd/es/table';
import IconDisplay, { availableIcons } from '../components/IconPicker';
import { customAlphabet } from 'nanoid';
import axios from 'axios';
import { motion } from 'framer-motion';
import dayjs from 'dayjs';

const nanoid = customAlphabet('abcdefghijklmnopqrstuvwxyz0123456789', 10);
const { Title } = Typography;

const MotionRow = (props: any) => (
    <motion.tr {...props} initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3 }} />
);

const CategoryPageContent: React.FC = () => {
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [editingCategory, setEditingCategory] = useState<Category | null>(null);
    const [form] = Form.useForm();
    const queryClient = useQueryClient();
    const { message } = App.useApp();

    const { data: categories = [], isLoading } = useQuery<Category[], Error>({
        queryKey: ['categories'],
        queryFn: () => getCategories().then(res => res.data),
    });

    const handleMutationSuccess = (successMsg: string) => {
        message.success(successMsg);
        // 【关键修改】使所有可能相关的查询失效
        queryClient.invalidateQueries({ queryKey: ['categories'] });
        queryClient.invalidateQueries({ queryKey: ['budgets'] }); // 预算下拉框需要最新分类
        queryClient.invalidateQueries({ queryKey: ['transactions'] }); // 交易详情中可能显示分类名
        queryClient.invalidateQueries({ queryKey: ['analyticsCharts'] }); // 分析图表依赖分类
        setIsModalOpen(false);
        form.resetFields();
    };
    
    const handleMutationError = (error: unknown) => {
        message.error(axios.isAxiosError(error) ? error.response?.data.error : '操作失败');
    };

    const createMutation = useMutation({
        mutationFn: (data: CreateCategoryRequest) => createCategory(data),
        onSuccess: () => handleMutationSuccess('私有分类添加成功！'),
        onError: handleMutationError,
    });

    const updateMutation = useMutation({
        mutationFn: (vars: {id: string, data: {name: string, icon: string}}) => updateCategory(vars.id, vars.data),
        onSuccess: () => handleMutationSuccess('私有分类更新成功！'),
        onError: handleMutationError,
    });
    
    const deleteMutation = useMutation<void, Error, string>({
        mutationFn: async (id) => { await deleteCategory(id); },
        onSuccess: () => handleMutationSuccess('私有分类删除成功！'),
        onError: handleMutationError,
    });

    const handleCancel = () => { setIsModalOpen(false); setEditingCategory(null); form.resetFields(); };

    const handleFormSubmit = async (values: any) => {
        if (editingCategory) {
            updateMutation.mutate({ id: editingCategory.id, data: { name: values.name, icon: values.icon } });
        } else {
            createMutation.mutate({ id: `user_${nanoid()}`, ...values });
        }
    };

    const openModal = (category: Category | null) => {
        setEditingCategory(category);
        form.setFieldsValue(category || { type: 'expense', icon: 'Archive' });
        setIsModalOpen(true);
    };
    
    const columns: ColumnsType<Category> = [
        { 
            title: '分类名称', 
            dataIndex: 'name', 
            key: 'name', 
            render: (name: string, record: Category) => (
                <Space>
                    <IconDisplay name={record.icon} size={20} />
                    <span>{name}</span>
                    {record.is_shared && <Tooltip title="共享分类"><ShareAltOutlined style={{color: '#1677ff'}}/></Tooltip>}
                </Space>
            ) 
        },
        { 
            title: '类型', 
            dataIndex: 'type', 
            key: 'type', 
            filters: [{ text: '支出', value: 'expense' }, { text: '收入', value: 'income' }, { text: '内部', value: 'internal' }], 
            onFilter: (value, record) => record.type === value, 
            render: (type: string) => { 
                let color; 
                if (type === 'income') color = 'success'; 
                else if (type === 'expense') color = 'error'; 
                else color = 'processing'; 
                return <Tag color={color}>{type.toUpperCase()}</Tag> 
            } 
        },
        { 
            title: '创建时间', 
            dataIndex: 'created_at', 
            key: 'created_at', 
            responsive: ['md'], 
            render: (text: string) => dayjs(text).isValid() ? dayjs(text).format('YYYY-MM-DD HH:mm') : 'N/A',
            sorter: (a, b) => dayjs(a.created_at).unix() - dayjs(b.created_at).unix(), 
        },
        { 
            title: '操作', 
            key: 'action', 
            width: 120, 
            align: 'center', 
            render: (_, record: Category) => {
                const canEdit = !record.is_shared || (record.is_shared && record.is_editable);
                const canDelete = !record.is_shared;

                return (
                    <Space>
                        <Tooltip title={canEdit ? "编辑" : "此分类不可编辑"}>
                            <Button type="text" icon={<EditOutlined />} onClick={() => openModal(record)} disabled={!canEdit} />
                        </Tooltip>
                        <Tooltip title={canDelete ? "删除" : "共享分类不可删除"}>
                            <Popconfirm title="确定删除吗？" description="此操作不可恢复" onConfirm={() => deleteMutation.mutate(record.id)} disabled={!canDelete}>
                                <Button type="text" icon={<DeleteOutlined />} danger disabled={!canDelete} />
                            </Popconfirm>
                        </Tooltip>
                    </Space>
                );
            }, 
        },
    ];

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card title={<Title level={4} style={{ margin: 0 }}>分类管理</Title>} extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => openModal(null)}>新增私有分类</Button>} />
            <Card>
                <Table columns={columns} dataSource={categories} rowKey="id" loading={isLoading} pagination={{ pageSize: 15, showSizeChanger: true, showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items` }} components={{ body: { row: MotionRow } }} scroll={{ x: 'max-content' }} />
            </Card>
            
            <Modal title={editingCategory ? '编辑私有分类' : '新增私有分类'} open={isModalOpen} onOk={form.submit} onCancel={handleCancel} confirmLoading={createMutation.isPending || updateMutation.isPending} destroyOnHidden>
                <Form form={form} layout="vertical" onFinish={handleFormSubmit} initialValues={{ type: 'expense' }}>
                    <Form.Item name="name" label="分类名称" rules={[{ required: true, message: '请输入分类名称' }]}><Input placeholder="例如：早餐、地铁" /></Form.Item>
                    <Form.Item name="type" label="类型" rules={[{ required: true }]}><Radio.Group disabled={!!editingCategory}><Radio.Button value="expense">支出</Radio.Button><Radio.Button value="income">收入</Radio.Button></Radio.Group></Form.Item>
                    <Form.Item name="icon" label="图标" rules={[{ required: true, message: '请选择一个图标'}]}><Select showSearch optionFilterProp="children" placeholder="选择一个图标">{availableIcons.map(iconName => (<Select.Option key={iconName} value={iconName}><Space><IconDisplay name={iconName} /> {iconName}</Space></Select.Option>))}</Select></Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

const CategoryPage: React.FC = () => (
    <App>
        <CategoryPageContent />
    </App>
);

export default CategoryPage;