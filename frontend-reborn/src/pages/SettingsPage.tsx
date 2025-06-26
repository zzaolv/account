// src/pages/SettingsPage.tsx
import React, { useState } from 'react';
import { Card, Button, Space, Upload, message, Typography, Modal, Alert, Input } from 'antd';
import { UploadOutlined, DownloadOutlined, WarningTwoTone } from '@ant-design/icons';
import { exportData, importData } from '../services/api';
import { useAuthStore } from '../stores/authStore';
import axios from 'axios';

// 【修正】从 Typography 中正确解构 Text
const { Title, Paragraph, Text } = Typography;

const SettingsPage: React.FC = () => {
    const [exporting, setExporting] = useState(false);
    const [uploading, setUploading] = useState(false);
    
    // 用于恢复确认模态框
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [confirmText, setConfirmText] = useState('');
    const [fileToUpload, setFileToUpload] = useState<File | null>(null);

    const { logout } = useAuthStore();

    const handleExport = async () => {
        setExporting(true);
        try {
            await exportData();
            message.success('备份文件已开始下载！');
        } catch (error) {
            message.error('备份失败，请稍后重试');
        } finally {
            setExporting(false);
        }
    };

    // 文件选择后的处理，打开确认模态框
    const beforeUpload = (file: File) => {
        if (!file.name.endsWith('.db')) {
            message.error('请选择 .db 格式的数据库备份文件！');
            return Upload.LIST_IGNORE;
        }
        setFileToUpload(file);
        setIsModalVisible(true);
        return false; // 阻止自动上传
    };

    // 确认恢复操作
    const handleConfirmImport = async () => {
        if (!fileToUpload) return;
        
        setIsModalVisible(false);
        setUploading(true);

        try {
            const response = await importData(fileToUpload);
            message.success(response.data.message, 5);
            Modal.success({
                title: '恢复成功',
                content: '数据已成功从备份恢复。为确保所有数据显示正确，应用将自动退出登录，请您使用恢复后数据中的账户重新登录。',
                okText: '好的，重新登录',
                onOk: () => {
                    logout();
                    window.location.href = '/login';
                },
            });
        } catch (error) {
            if (axios.isAxiosError(error) && error.response?.data?.error) {
                message.error(`恢复失败: ${error.response.data.error}`, 5);
            } else {
                message.error('恢复过程中发生未知错误', 5);
            }
        } finally {
            setUploading(false);
            setFileToUpload(null);
            setConfirmText('');
        }
    };

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Title level={2}>数据中心</Title>
            
            <Card title="全量数据备份">
                <Paragraph>
                    将您的**所有数据**（包括账户、流水、预算、借贷、分类等）完整备份为一个 `.db` 数据库文件。
                    这是最可靠的数据备份方式。请妥善保管好您的备份文件。
                </Paragraph>
                <Button type="primary" icon={<DownloadOutlined />} onClick={handleExport} loading={exporting}>
                    备份所有数据
                </Button>
            </Card>

            <Card title="从备份中恢复">
                <Alert
                    message="极度危险操作：恢复数据将覆盖当前所有数据！"
                    description="此操作会用您上传的备份文件完全替换服务器上的现有数据库。当前的所有数据都将丢失且无法找回。请务必在操作前确认您已备份好当前数据！"
                    type="error"
                    showIcon
                    style={{ marginBottom: 16 }}
                />
                 <Upload beforeUpload={beforeUpload} showUploadList={false} accept=".db">
                    <Button icon={<UploadOutlined />} loading={uploading} danger>
                        选择备份文件以恢复
                    </Button>
                </Upload>
            </Card>

            <Modal
                title={
                    <Space>
                        <WarningTwoTone twoToneColor="#faad14" />
                        确认恢复操作
                    </Space>
                }
                open={isModalVisible}
                onOk={handleConfirmImport}
                onCancel={() => setIsModalVisible(false)}
                okText="我确认，执行恢复"
                okButtonProps={{ disabled: confirmText !== '我确认覆盖所有数据' }}
                destroyOnClose
            >
                <Paragraph>
                    这是一个不可逆的操作。恢复后，您当前的所有数据将被彻底删除。
                </Paragraph>
                <Paragraph>
                    请在下面的输入框中输入“<Text strong>我确认覆盖所有数据</Text>”以确认您了解风险并继续。
                </Paragraph>
                <Input 
                    placeholder="我确认覆盖所有数据"
                    value={confirmText}
                    onChange={(e) => setConfirmText(e.target.value)}
                />
            </Modal>
        </Space>
    );
};

export default SettingsPage;