// src/pages/TransactionPage.tsx
import React, { useState } from 'react';
import { Card, Table, Tag, DatePicker, Space, Popconfirm, Button, Typography, Result, Skeleton, Empty, App } from 'antd'; // 【修改】导入 App
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTransactions, deleteTransaction } from '../services/api';
import type { GetTransactionsResponse, Transaction } from '../types';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { DeleteOutlined, ArrowRightOutlined } from '@ant-design/icons';
import { motion } from 'framer-motion';
import axios from 'axios';

const { Title, Text } = Typography;
const MotionRow = (props: any) => (<motion.tr {...props} initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3 }} />);

// 【新增】将页面主要内容包裹起来
const TransactionPageContent: React.FC = () => {
    const [dateFilter, setDateFilter] = useState<{ year?: number; month?: number }>({
        year: dayjs().year(),
        month: dayjs().month() + 1,
    });
    const [pickerValue, setPickerValue] = useState<dayjs.Dayjs | null>(dayjs());
    const queryClient = useQueryClient();
    const { message } = App.useApp(); // 【关键修改】使用 hook 获取 message 实例

    const { data, isLoading, isError, error, refetch } = useQuery<GetTransactionsResponse, Error>({
        queryKey: ['transactions', dateFilter],
        queryFn: () => getTransactions(dateFilter).then(res => res.data),
        retry: 1,
    });

    const deleteMutation = useMutation({
        mutationFn: deleteTransaction,
        onSuccess: () => {
            message.success('删除成功！'); // 【关键修改】使用 hook 返回的 message
            queryClient.invalidateQueries({ queryKey: ['transactions'] });
            queryClient.invalidateQueries({ queryKey: ['accounts'] });
            queryClient.invalidateQueries({ queryKey: ['dashboardCards'] });
            queryClient.invalidateQueries({ queryKey: ['analyticsCharts'] });
            queryClient.invalidateQueries({ queryKey: ['dashboardWidgets'] });
            queryClient.invalidateQueries({ queryKey: ['loans'] });
        },
        onError: (err: unknown) => {
            const errorMsg = axios.isAxiosError(err) && err.response ? err.response.data.error : '删除失败';
            message.error(errorMsg); // 【关键修改】使用 hook 返回的 message
        },
    });

    const handleDateChange = (date: dayjs.Dayjs | null) => {
        setPickerValue(date);
        setDateFilter(date ? { year: date.year(), month: date.month() + 1 } : {});
    };

    const columns: ColumnsType<Transaction> = [
        { title: '日期', dataIndex: 'transaction_date', key: 'transaction_date', sorter: (a, b) => dayjs(a.transaction_date).unix() - dayjs(b.transaction_date).unix(), responsive: ['md'], width: 120 },
        { title: '类型', dataIndex: 'type', key: 'type', width: 100, filters: [ { text: '收入', value: 'income' }, { text: '支出', value: 'expense' }, { text: '还款', value: 'repayment' }, { text: '转账', value: 'transfer' }, ], onFilter: (value, record) => record.type === value, render: (type) => { let color = 'blue'; if (type === 'expense') color = 'red'; if (type === 'income') color = 'green'; if (type === 'repayment') color = 'orange'; return <Tag color={color}>{type.toUpperCase()}</Tag>; } },
        { title: '金额', dataIndex: 'amount', key: 'amount', width: 150, align: 'right', render: (amount, record) => (<Text type={record.type === 'income' ? 'success' : (record.type === 'transfer' ? undefined : 'danger')} strong> {record.type === 'income' ? '+' : (record.type === 'transfer' ? '' : '-')} ¥{amount.toFixed(2)} </Text>), sorter: (a, b) => a.amount - b.amount },
        { title: '账户', key: 'account', render: (_, record) => { if (record.type === 'transfer') { return <Space><Tag>{record.from_account_name}</Tag> <ArrowRightOutlined /> <Tag>{record.to_account_name}</Tag></Space>; } if (record.from_account_name) return <Tag>{record.from_account_name}</Tag>; if (record.to_account_name) return <Tag>{record.to_account_name}</Tag>; return <Text type="secondary">无</Text>; } },
        { title: '分类', dataIndex: 'category_name', key: 'category_name', responsive: ['sm'] },
        { title: '描述', dataIndex: 'description', key: 'description' },
        { title: '操作', key: 'action', align: 'center', width: 80, fixed: 'right', render: (_, record) => (<Popconfirm title="确定删除这条记录吗?" description="相关账户的余额将会恢复。" onConfirm={() => deleteMutation.mutate(record.id)} okText="确定" cancelText="取消" disabled={deleteMutation.isPending}> <Button type="text" danger icon={<DeleteOutlined />} loading={deleteMutation.isPending && deleteMutation.variables === record.id} /> </Popconfirm>) }
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
                        title="数据加载失败"
                        subTitle={`错误: ${error.message}`}
                        extra={<Button type="primary" onClick={() => refetch()}>点击重试</Button>}
                    />
                </Card>
            );
        }
        if (!data || data.transactions.length === 0) {
            return (
                <Card>
                    <Empty description="当前时段没有流水记录哦~" />
                </Card>
            );
        }
        return (
            <Card>
                <Table
                    columns={columns}
                    dataSource={data?.transactions}
                    rowKey="id"
                    pagination={{ pageSize: 15, showTotal: (total) => `共 ${total} 条` }}
                    components={{ body: { row: MotionRow } }}
                    scroll={{ x: 'max-content' }}
                    summary={() => data && data.transactions.length > 0 ? (
                        <Table.Summary fixed>
                            <Table.Summary.Row>
                                <Table.Summary.Cell index={0} colSpan={2}><Text strong>期间总计</Text></Table.Summary.Cell>
                                <Table.Summary.Cell index={2} align="right"><Text type="success" strong>¥{data?.summary.total_income.toFixed(2)}</Text></Table.Summary.Cell>
                                <Table.Summary.Cell index={3} colSpan={3} ><Text type="danger" strong>¥{data?.summary.total_expense.toFixed(2)}</Text></Table.Summary.Cell>
                                <Table.Summary.Cell index={6}><Text strong>净: ¥{data?.summary.net_balance.toFixed(2)}</Text></Table.Summary.Cell>
                            </Table.Summary.Row>
                        </Table.Summary>
                    ) : null}
                />
            </Card>
        );
    };

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card>
                <Space>
                    <Title level={4} style={{ margin: 0 }}>交易流水</Title>
                    <DatePicker onChange={handleDateChange} picker="month" value={pickerValue} allowClear/>
                </Space>
            </Card>
            {renderMainContent()}
        </Space>
    );
};

// 【新增】导出被 App 包裹的组件
const TransactionPage: React.FC = () => (
    <App>
        <TransactionPageContent />
    </App>
);

export default TransactionPage;