// src/pages/SettingsPage.tsx
import React, { useState } from 'react';
import { Card, Button, Space, Upload, message, Typography } from 'antd';
import { UploadOutlined, DownloadOutlined } from '@ant-design/icons';
import { exportData, importData } from '../services/api';

const { Title, Paragraph } = Typography;

const SettingsPage: React.FC = () => {
    const [uploading, setUploading] = useState(false);

    const handleImport = async (options: any) => {
        const { file } = options;
        setUploading(true);
        try {
            await importData(file);
            message.success(`${file.name} 文件上传成功，后端正在处理导入...`);
        } catch (error) {
            message.error('导入失败，请检查文件格式或联系管理员');
        } finally {
            setUploading(false);
        }
    };

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Title level={2}>数据中心</Title>
            
            <Card title="数据导出">
                <Paragraph>
                    将您所有的交易流水记录导出为 CSV 文件。这是一个很好的备份数据的方式。
                </Paragraph>
                <Button type="primary" icon={<DownloadOutlined />} onClick={exportData}>
                    导出为 CSV
                </Button>
            </Card>

            <Card title="数据导入 (功能开发中)">
                <Paragraph>
                    从 CSV 文件导入交易数据。请注意，这一个高风险操作，请在导入前务必备份好现有数据。
                </Paragraph>
                 <Upload customRequest={handleImport} showUploadList={false}>
                    <Button icon={<UploadOutlined />} loading={uploading}>
                        选择 CSV 文件导入
                    </Button>
                </Upload>
            </Card>
        </Space>
    );
};

export default SettingsPage;