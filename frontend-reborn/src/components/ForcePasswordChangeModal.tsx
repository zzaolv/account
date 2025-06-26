// src/components/ForcePasswordChangeModal.tsx
import React, { useState, useEffect } from 'react';
import { Modal, Form, Input, Button, App } from 'antd';
import { updatePassword } from '../services/api';
import { useAuthStore } from '../stores/authStore';
import axios from 'axios';

interface Props {
  open: boolean;
}

const ForcePasswordChangeModal: React.FC<Props> = ({ open }) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const setPasswordChanged = useAuthStore((state) => state.setPasswordChanged);
  const { message: staticMessage } = App.useApp();
  const { logout } = useAuthStore();

  // 【关键修改】使用 useEffect 监听 open 状态
  useEffect(() => {
    if (open) {
      // 当模态框打开时，清空所有表单字段
      form.resetFields();
    }
  }, [open, form]);

  const handleOk = async () => {
    try {
      setLoading(true);
      const values = await form.validateFields();
      await updatePassword({ new_password: values.newPassword });
      setPasswordChanged();
      staticMessage.success('密码修改成功，请使用新密码重新登录。');
      setTimeout(() => {
        logout(); // 使用从 store 获取的 logout 函数
        window.location.href = '/login';
      }, 2000);
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        staticMessage.error(error.response.data.error || '修改失败');
      } else {
        staticMessage.error('发生未知错误');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title="首次登录 - 请修改初始密码"
      open={open}
      closable={false}
      maskClosable={false}
      destroyOnClose // 【关键修改】在 Modal 关闭时销ryOnClose 确保状态不被缓存
      footer={[
        <Button key="submit" type="primary" loading={loading} onClick={handleOk}>
          确认修改
        </Button>,
      ]}
    >
      <p>为了您的账户安全，请设置一个新密码。</p>
      <Form form={form} layout="vertical">
        <Form.Item
          name="newPassword"
          label="新密码"
          rules={[{ required: true, message: '请输入新密码' }, { min: 6, message: '密码至少6位' }]}
        >
          <Input.Password />
        </Form.Item>
        <Form.Item
          name="confirmPassword"
          label="确认新密码"
          dependencies={['newPassword']}
          hasFeedback
          rules={[
            { required: true, message: '请再次输入新密码' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('newPassword') === value) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error('两次输入的密码不一致!'));
              },
            }),
          ]}
        >
          <Input.Password />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ForcePasswordChangeModal;