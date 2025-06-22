// src/pages/CategoryPage.tsx
import React, { useState, useEffect, useCallback } from 'react';
import { Button, Card, Table, Tag, message, Modal, Form, Input, Radio, Space, Popconfirm, Select, Tooltip } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { getCategories, createCategory, updateCategory, deleteCategory } from '../services/api';
import type { Category } from '../types';
import type { ColumnsType } from 'antd/es/table';
import IconDisplay, { availableIcons } from '../components/IconPicker';
import { customAlphabet } from 'nanoid';
import axios from 'axios';

// 使用 nanoid 生成一个10位的、由小写字母和数字组成的唯一ID
const nanoid = customAlphabet('abcdefghijklmnopqrstuvwxyz0123456789', 10);

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

    const handleCancel = () => {
        setIsModalOpen(false);
        setEditingCategory(null);
        form.resetFields();
    };

    const handleFormSubmit = async (values: any) => {
        setIsSubmitting(true);
        try {
            if (editingCategory) {
                // 更新模式
                await updateCategory(editingCategory.id, { name: values.name, icon: values.icon });
                message.success('分类更新成功！');
            } else {
                // 创建模式
                const newCategory: Omit<Category, 'created_at'> = {
                    id: `user_${nanoid()}`, // 为用户自定义分类加上前缀以作区分
                    name: values.name,
                    type: values.type,
                    icon: values.icon,
                };
                await createCategory(newCategory);
                message.success('分类添加成功！');
            }
            handleCancel();
            await fetchCategories();
        } catch (error: unknown) {
             if (axios.isAxiosError(error) && error.response) {
                message.error(error.response.data.error || '操作失败');
            } else {
                message.error('操作失败，发生未知错误');
            }
        } finally {
            setIsSubmitting(false);
        }
    };

    const openModal = (category: Category | null) => {
        setEditingCategory(category);
        if (category) {
            form.setFieldsValue(category);
        } else {
            // 设置新增时的默认值
            form.setFieldsValue({ type: 'expense', icon: 'Archive' });
        }
        setIsModalOpen(true);
    };

    const handleDelete = async (id: string) => {
        try {
            await deleteCategory(id);
            message.success('分类删除成功！');
            await fetchCategories();
        } catch (error: unknown) {
            // 【安全处理】捕获后端返回的冲突错误
            if (axios.isAxiosError(error) && error.response) {
                message.error(error.response.data.error || '删除失败');
            } else {
                message.error('删除失败，发生未知错误');
            }
        }
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
                </Space>
            )
        },
        {
            title: '类型',
            dataIndex: 'type',
            key: 'type',
            filters: [
                { text: '支出', value: 'expense' },
                { text: '收入', value: 'income' },
            ],
            onFilter: (value, record) => record.type === value,
            render: (type: 'income' | 'expense') => (
                <Tag color={type === 'income' ? 'success' : 'error'}>
                    {type === 'income' ? '收入' : '支出'}
                </Tag>
            )
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            render: (text: string) => new Date(text).toLocaleString(),
            sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
        },
        {
            title: '操作',
            key: 'action',
            width: 120,
            align: 'center',
            render: (_, record) => {
                // 【保护预设分类】预设分类的ID不带 "user_" 前缀
                const isPreset = !record.id.startsWith('user_');
                return (
                    <Space>
                        <Tooltip title={isPreset ? "预设分类不可编辑" : "编辑"}>
                             <Button icon={<EditOutlined />} onClick={() => openModal(record)} disabled={isPreset} />
                        </Tooltip>
                        <Tooltip title={isPreset ? "预设分类不可删除" : "删除"}>
                            <Popconfirm 
                                title="确定删除吗？"
                                description="此操作不可恢复"
                                onConfirm={() => handleDelete(record.id)} 
                                disabled={isPreset}
                                okText="确定"
                                cancelText="取消"
                            >
                                <Button icon={<DeleteOutlined />} danger disabled={isPreset} />
                            </Popconfirm>
                        </Tooltip>
                    </Space>
                );
            },
        },
    ];

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card
                title="分类管理"
                extra={
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal(null)}>
                        新增分类
                    </Button>
                }
            />
            <Card>
                <Table
                    columns={columns}
                    dataSource={categories}
                    rowKey="id"
                    loading={loading}
                    pagination={{ pageSize: 10, showSizeChanger: true }}
                />
            </Card>
            
            <Modal
                title={editingCategory ? '编辑分类' : '新增分类'}
                open={isModalOpen} 
                onOk={form.submit} 
                onCancel={handleCancel} 
                confirmLoading={isSubmitting}
                destroyOnClose
            >
                <Form form={form} layout="vertical" onFinish={handleFormSubmit} initialValues={{ type: 'expense' }}>
                    <Form.Item name="name" label="分类名称" rules={[{ required: true, message: '请输入分类名称' }]}>
                        <Input placeholder="例如：早餐、地铁" />
                    </Form.Item>
                     <Form.Item name="type" label="类型" rules={[{ required: true }]}>
                        <Radio.Group disabled={!!editingCategory}>
                            <Radio.Button value="expense">支出</Radio.Button>
                            <Radio.Button value="income">收入</Radio.Button>
                        </Radio.Group>
                    </Form.Item>
                    <Form.Item name="icon" label="图标" rules={[{ required: true, message: '请选择一个图标'}]}>
                        <Select showSearch optionFilterProp="children" placeholder="选择一个图标">
                            {availableIcons.map(iconName => (
                                <Select.Option key={iconName} value={iconName}>
                                    <Space>
                                        <IconDisplay name={iconName} /> {iconName}
                                    </Space>
                                </Select.Option>
                            ))}
                        </Select>
                    </Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

export default CategoryPage;