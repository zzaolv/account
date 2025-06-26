// src/pages/AdminPage.tsx
import React, { useState } from 'react';
import { Card, Col, Row, Statistic, Typography, Table, Space, Button, Popconfirm, Modal, Form, Input, App, Spin } from 'antd';
import { UserAddOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getSystemStats, getUsers, registerUser, deleteUser } from '../services/api';
import type { User, RegisterRequest } from '../services/api';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import axios from 'axios';

const { Title } = Typography;

const formatBytes = (bytes: number, decimals = 2) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
};

const AdminPage: React.FC = () => {
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [form] = Form.useForm();
    const queryClient = useQueryClient();
    const { message } = App.useApp();

    const { data: stats, isLoading: isLoadingStats } = useQuery({
        queryKey: ['systemStats'],
        queryFn: getSystemStats,
    });

    const { data: users, isLoading: isLoadingUsers } = useQuery({
        queryKey: ['users'],
        queryFn: getUsers,
    });
    
    const mutationOptions = (successMsg: string) => ({
        onSuccess: () => {
            message.success(successMsg);
            queryClient.invalidateQueries({ queryKey: ['users'] });
            queryClient.invalidateQueries({ queryKey: ['systemStats'] });
            setIsModalOpen(false);
            form.resetFields();
        },
        onError: (error: unknown) => {
            const errorMsg = axios.isAxiosError(error) ? error.response?.data.error : '操作失败';
            message.error(errorMsg);
        },
    });

    const registerMutation = useMutation({
        mutationFn: (data: RegisterRequest) => registerUser(data),
        ...mutationOptions('用户创建成功！'),
    });

    const deleteMutation = useMutation({
        mutationFn: (id: number) => deleteUser(id),
        ...mutationOptions('用户删除成功！'),
    });

    const handleAddUser = () => {
        form.validateFields().then(values => {
            registerMutation.mutate(values);
        });
    };

    const columns: ColumnsType<User> = [
        { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
        { title: '用户名', dataIndex: 'username', key: 'username' },
        { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: (text) => dayjs(text).format('YYYY-MM-DD HH:mm') },
        { 
            title: '操作', 
            key: 'action', 
            width: 120,
            render: (_, record) => (
                <Popconfirm
                    title={`确定要删除用户 "${record.username}" 吗？`}
                    description="此操作不可恢复，将删除该用户及其所有数据！"
                    onConfirm={() => deleteMutation.mutate(record.id)}
                    disabled={record.is_admin}
                >
                    <Button type="link" danger icon={<DeleteOutlined />} disabled={record.is_admin}>
                        删除
                    </Button>
                </Popconfirm>
            )
        },
    ];

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Title level={2}>管理后台</Title>
            
            <Spin spinning={isLoadingStats}>
                <Row gutter={[16, 16]}>
                    <Col xs={24} sm={12} md={6}>
                        <Card><Statistic title="用户总数" value={stats?.data.user_count} /></Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                        <Card><Statistic title="流水总数" value={stats?.data.transaction_count} /></Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                        <Card><Statistic title="账户总数" value={stats?.data.account_count} /></Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                        <Card><Statistic title="数据库大小" value={formatBytes(stats?.data.db_size_bytes || 0)} /></Card>
                    </Col>
                </Row>
            </Spin>

            <Card
                title="用户管理"
                extra={<Button type="primary" icon={<UserAddOutlined />} onClick={() => setIsModalOpen(true)}>新增用户</Button>}
            >
                <Spin spinning={isLoadingUsers}>
                    <Table dataSource={users?.data} columns={columns} rowKey="id" />
                </Spin>
            </Card>

            <Modal
                title="新增用户"
                open={isModalOpen}
                onCancel={() => setIsModalOpen(false)}
                onOk={handleAddUser}
                confirmLoading={registerMutation.isPending}
            >
                <Form form={form} layout="vertical">
                    <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
                        <Input />
                    </Form.Item>
                    <Form.Item name="password" label="初始密码" rules={[{ required: true, min: 6, message: '密码至少6位' }]}>
                        <Input.Password />
                    </Form.Item>
                </Form>
            </Modal>
        </Space>
    );
};

export default AdminPage;