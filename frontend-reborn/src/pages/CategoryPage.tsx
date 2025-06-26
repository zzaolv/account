// src/pages/CategoryPage.tsx
import React, { useState } from 'react';
import { Button, Card, Table, Tag, Modal, Form, Input, Radio, Space, Popconfirm, Select, Tooltip, Typography, App, Result, Skeleton, Empty, notification } from 'antd';
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

    const { data: categories = [], isLoading, isError, error, refetch } = useQuery<Category[], Error>({
        queryKey: ['categories'],
        queryFn: () => getCategories().then(res => res.data || []),
        retry: 1,
    });

    const handleMutationError = (error: unknown, title: string) => {
        notification.error({
            message: title,
            description: axios.isAxiosError(error) ? error.response?.data.error : '发生未知错误，请稍后重试。',
        });
    };
    
    const createMutation = useMutation({
        mutationFn: (data: CreateCategoryRequest) => createCategory(data),
        onMutate: async (newCategoryData) => {
            await queryClient.cancelQueries({ queryKey: ['categories'] });
            const previousCategories = queryClient.getQueryData<Category[]>(['categories']) || [];
            const optimisticCategory: Category = {
                ...newCategoryData,
                created_at: new Date().toISOString(),
                is_shared: false,
                is_editable: true,
            };
            queryClient.setQueryData<Category[]>(['categories'], (old) => [...(old || []), optimisticCategory]);
            handleCancel();
            message.success('分类已添加！');
            return { previousCategories };
        },
        onError: (err, _newCategoryData, context) => {
            if (context?.previousCategories) {
                queryClient.setQueryData(['categories'], context.previousCategories);
            }
            handleMutationError(err, '添加分类失败');
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: ['categories'] });
            queryClient.invalidateQueries({ queryKey: ['budgets'] });
        },
    });

    const updateMutation = useMutation({
        mutationFn: (vars: {id: string, data: {name: string, icon: string}}) => updateCategory(vars.id, vars.data),
        onSuccess: () => {
            message.success('分类更新成功！');
            queryClient.invalidateQueries({ queryKey: ['categories'] });
            queryClient.invalidateQueries({ queryKey: ['budgets'] });
            handleCancel();
        },
        onError: (err) => handleMutationError(err, '更新分类失败'),
    });
    
    const deleteMutation = useMutation<void, Error, string>({
        mutationFn: async (id) => { await deleteCategory(id); },
        onSuccess: () => {
            message.success('分类删除成功！');
            queryClient.invalidateQueries({ queryKey: ['categories'] });
            queryClient.invalidateQueries({ queryKey: ['budgets'] });
        },
        onError: (err) => handleMutationError(err, '删除分类失败'),
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
        { title: '分类名称', dataIndex: 'name', key: 'name', render: (name: string, record: Category) => (<Space><IconDisplay name={record.icon} size={20} /><span>{name}</span>{record.is_shared && <Tooltip title="共享分类"><ShareAltOutlined style={{color: '#1677ff'}}/></Tooltip>}</Space>) },
        { title: '类型', dataIndex: 'type', key: 'type', filters: [{ text: '支出', value: 'expense' }, { text: '收入', value: 'income' }, { text: '内部', value: 'internal' }], onFilter: (value, record) => record.type === value, render: (type: string) => { let color; if (type === 'income') color = 'success'; else if (type === 'expense') color = 'error'; else color = 'processing'; return <Tag color={color}>{type.toUpperCase()}</Tag> } },
        { title: '创建时间', dataIndex: 'created_at', key: 'created_at', responsive: ['md'], render: (text: string) => dayjs(text).isValid() ? dayjs(text).format('YYYY-MM-DD HH:mm') : 'N/A', sorter: (a, b) => dayjs(a.created_at).unix() - dayjs(b.created_at).unix(), },
        { title: '操作', key: 'action', width: 120, align: 'center', render: (_, record: Category) => {
            const canEdit = !record.is_shared || (record.is_shared && record.is_editable);
            const canDelete = !record.is_shared;
            const isDeleting = deleteMutation.isPending && deleteMutation.variables === record.id;
            const isMutating = updateMutation.isPending && updateMutation.variables?.id === record.id;

            return (
                <Space>
                    <Tooltip title={canEdit ? "编辑" : "此分类不可编辑"}>
                        <Button type="text" icon={<EditOutlined />} onClick={() => openModal(record)} disabled={!canEdit || isDeleting} loading={isMutating} />
                    </Tooltip>
                    <Tooltip title={canDelete ? "删除" : "共享分类不可删除"}>
                        <Popconfirm title="确定删除吗？" description="此操作不可恢复" onConfirm={() => deleteMutation.mutate(record.id)} disabled={!canDelete || isDeleting}>
                            <Button type="text" icon={<DeleteOutlined />} danger disabled={!canDelete} loading={isDeleting} />
                        </Popconfirm>
                    </Tooltip>
                </Space>
            );
        }},
    ];
    
    const renderMainContent = () => {
        if (isLoading) {
            return <Card><Skeleton active paragraph={{ rows: 5 }} /></Card>;
        }
        if (isError) {
            return (
                <Card>
                    <Result
                        status="error"
                        title="分类数据加载失败"
                        subTitle={`错误: ${error.message}`}
                        extra={<Button type="primary" onClick={() => refetch()}>点击重试</Button>}
                    />
                </Card>
            );
        }
        
        // 【关键修改】使用正确的 Empty 组件用法
        if (categories.length === 0) {
            return (
                <Card>
                    <Empty description={
                        <span>
                            您还没有任何私有分类，快来创建一个吧！
                            <br />
                            <Button type="primary" onClick={() => openModal(null)} style={{ marginTop: 16 }}>
                                立即新增
                            </Button>
                        </span>
                    } />
                </Card>
            );
        }

        return (
            <Card>
                <Table columns={columns} dataSource={categories} rowKey="id" pagination={{ pageSize: 15, showSizeChanger: true, showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items` }} components={{ body: { row: MotionRow } }} scroll={{ x: 'max-content' }} />
            </Card>
        );
    }

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card title={<Title level={4} style={{ margin: 0 }}>分类管理</Title>} extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => openModal(null)}>新增私有分类</Button>} />
            
            {renderMainContent()}
            
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