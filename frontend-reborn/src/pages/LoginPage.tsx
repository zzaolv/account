// src/pages/LoginPage.tsx
import React, { useState } from 'react';
import { Card, Form, Input, Button, Typography, Checkbox, Row, Col, App } from 'antd';
import { UserOutlined, LockOutlined, SafetyOutlined } from '@ant-design/icons';
import { useAuthStore } from '../stores/authStore';
import { login } from '../services/api';
import type { LoginRequest } from '../services/api';
import { useNavigate } from 'react-router-dom';
import axios from 'axios';
import { motion } from 'framer-motion';

const { Title, Text, Paragraph } = Typography;

// 你可以将这个 URL 替换为你自己喜欢的、可商用的高质量图片
const backgroundImageUrl = 'https://images.unsplash.com/photo-1554224155-8d04421cd699?q=80&w=2072&auto=format&fit=crop';

const LoginPageContent: React.FC = () => {
    const navigate = useNavigate();
    const authLogin = useAuthStore((state) => state.login);
    const [loading, setLoading] = useState(false);
    const [form] = Form.useForm();
    const { modal } = App.useApp();

    const onFinish = async (values: LoginRequest) => {
        setLoading(true);
        try {
            const response = await login(values);
            const { access_token, refresh_token, username, is_admin, must_change_password } = response.data;
            
            authLogin(access_token, refresh_token || null, { username, is_admin }, must_change_password);
            
            if (!must_change_password) {
                modal.success({
                    title: `欢迎回来, ${username}!`,
                    content: '即将跳转到仪表盘...',
                    okText: '立即跳转',
                    onOk: () => navigate('/'),
                });
                setTimeout(() => {
                    if (window.location.pathname.includes('/login')) {
                         navigate('/');
                    }
                }, 2000);
            }

        } catch (error) {
            if (axios.isAxiosError(error) && error.response) {
                modal.warning({
                    title: '登录失败',
                    content: error.response.data.error || '发生未知错误，请稍后重试。',
                    okText: '知道了'
                });
            } else {
                modal.error({ title: '网络错误', content: '无法连接到服务器。' });
            }
        } finally {
            setLoading(false);
            form.setFieldsValue({ password: '' });
        }
    };

    return (
        <div style={{ 
            display: 'flex', 
            justifyContent: 'center', 
            alignItems: 'center', 
            minHeight: '100vh', 
            background: '#f0f2f5', // 一个柔和的背景色
            padding: '16px' 
        }}>
            <motion.div
                initial={{ opacity: 0, y: -20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5 }}
                style={{ width: '100%', maxWidth: '960px' }} // 调整了最大宽度
            >
                <Card style={{ 
                    width: '100%', 
                    boxShadow: '0 4px 20px rgba(0,0,0,0.1)', // 更深的阴影
                    borderRadius: '12px', 
                    overflow: 'hidden',
                    border: 'none',
                }}>
                    <Row>
                        {/* 【关键修改】左侧美化栏 */}
                        <Col 
                            xs={0} 
                            sm={12} 
                            md={10} // 在中等屏幕上稍微小一点
                            style={{
                                position: 'relative',
                                backgroundImage: `url(${backgroundImageUrl})`,
                                backgroundSize: 'cover',
                                backgroundPosition: 'center',
                                color: 'white',
                                display: 'flex',
                                flexDirection: 'column',
                                justifyContent: 'flex-end', // 内容置于底部
                                padding: '48px',
                            }}
                        >
                            {/* 半透明蒙层 */}
                            <div style={{
                                position: 'absolute',
                                top: 0, left: 0, right: 0, bottom: 0,
                                background: 'linear-gradient(to top, rgba(0, 0, 0, 0.7), rgba(0, 0, 0, 0.1))',
                            }} />
                            
                            {/* 文字内容（在蒙层之上） */}
                            <div style={{ position: 'relative', zIndex: 1 }}>
                                <Title level={2} style={{ color: 'white', fontWeight: 700, marginBottom: '16px' }}>
                                    极简记账本
                                </Title>
                                <Paragraph style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: '16px' }}>
                                    掌控您的每一笔资金流动，轻松实现财务自由。
                                </Paragraph>
                            </div>
                        </Col>

                        {/* 【关键修改】右侧登录表单 */}
                        <Col 
                            xs={24} 
                            sm={12} 
                            md={14} // 相应调整
                            style={{ 
                                padding: '48px 40px', 
                                display: 'flex', 
                                flexDirection: 'column', 
                                justifyContent: 'center',
                                minHeight: '550px', // 确保高度一致
                            }}
                        >
                            <div style={{ textAlign: 'center', marginBottom: '32px' }}>
                                <SafetyOutlined style={{ fontSize: '48px', color: 'var(--ant-primary-color)' }}/>
                                <Title level={3} style={{ marginTop: '16px' }}>账户登录</Title>
                                <Text type="secondary">欢迎回来！</Text>
                            </div>
                            <Form
                                form={form}
                                name="login"
                                initialValues={{ rememberMe: true }}
                                onFinish={onFinish}
                                size="large"
                            >
                                <Form.Item
                                    name="username"
                                    rules={[{ required: true, message: '请输入用户名!' }]}
                                >
                                    <Input prefix={<UserOutlined />} placeholder="用户名" />
                                </Form.Item>
                                <Form.Item
                                    name="password"
                                    rules={[{ required: true, message: '请输入密码!' }]}
                                >
                                    <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                                </Form.Item>
                                <Form.Item>
                                    <Form.Item name="rememberMe" valuePropName="checked" noStyle>
                                        <Checkbox>记住我</Checkbox>
                                    </Form.Item>
                                    <a style={{ float: 'right' }} href="">忘记密码?</a>
                                </Form.Item>
                                <Form.Item style={{marginTop: '24px'}}>
                                    <Button type="primary" htmlType="submit" style={{ width: '100%' }} loading={loading}>
                                        登 录
                                    </Button>
                                </Form.Item>
                            </Form>
                        </Col>
                    </Row>
                </Card>
            </motion.div>
        </div>
    );
};

const LoginPage: React.FC = () => (
    <App>
        <LoginPageContent />
    </App>
);

export default LoginPage;