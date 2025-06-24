// src/pages/TransactionPage.tsx
import React, { useState, useEffect, useCallback } from 'react';
// 【修复】移除未使用的 AnimatePresence
import { Card, Table, Tag, DatePicker, message, Space, Popconfirm, Button, Typography } from 'antd';
import { getTransactions, deleteTransaction } from '../services/api';
import type { GetTransactionsResponse, Transaction } from '../types';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { DeleteOutlined } from '@ant-design/icons';
import { motion } from 'framer-motion';

const { Title } = Typography;

const TransactionPage: React.FC = () => {
    const [data, setData] = useState<GetTransactionsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [dateFilter, setDateFilter] = useState<{ year?: number; month?: number }>({
        year: dayjs().year(),
        month: dayjs().month() + 1,
    });
    const [pickerValue, setPickerValue] = useState<dayjs.Dayjs | null>(dayjs());

    const fetchData = useCallback(async () => {
        setLoading(true);
        try {
            const res = await getTransactions(dateFilter);
            setData(res.data);
        } catch (error) {
            message.error('获取交易流水失败');
        } finally {
            setLoading(false);
        }
    }, [dateFilter]);

    useEffect(() => {
        fetchData();
    }, [fetchData]);

    const handleDateChange = (date: dayjs.Dayjs | null) => {
        setPickerValue(date);
        setDateFilter(date ? { year: date.year(), month: date.month() + 1 } : {});
    };
    
    const handleDelete = async (id: number) => {
        try {
            await deleteTransaction(id);
            message.success('删除成功！');
            fetchData();
        } catch (error) {
            message.error('删除失败');
        }
    }

    const columns: ColumnsType<Transaction> = [
        { title: '日期', dataIndex: 'transaction_date', key: 'transaction_date', sorter: (a, b) => dayjs(a.transaction_date).unix() - dayjs(b.transaction_date).unix(), responsive: ['md'] },
        { title: '类型', dataIndex: 'type', key: 'type', filters: [ { text: '收入', value: 'income' }, { text: '支出', value: 'expense' }, { text: '还款', value: 'repayment' }, { text: '转账', value: 'transfer' }, ], onFilter: (value, record) => record.type === value, render: (type) => { let color = 'blue'; if (type === 'expense') color = 'red'; if (type === 'income') color = 'green'; if (type === 'repayment') color = 'orange'; return <Tag color={color}>{type.toUpperCase()}</Tag>; } },
        { title: '金额', dataIndex: 'amount', key: 'amount', render: (amount, record) => (<span style={{ color: record.type === 'income' ? '#52c41a' : (record.type === 'expense' || record.type === 'repayment' ? '#f5222d' : '#1677ff'), fontWeight: 500 }}> {record.type === 'income' ? '+' : '-'} ¥{amount.toFixed(2)} </span>), sorter: (a, b) => a.amount - b.amount },
        { title: '分类', dataIndex: 'category_name', key: 'category_name', responsive: ['sm'] },
        { title: '描述', dataIndex: 'description', key: 'description' },
        { title: '操作', key: 'action', align: 'center', width: 80, render: (_, record) => (<Popconfirm title="确定删除这条记录吗?" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消"> <Button type="text" danger icon={<DeleteOutlined />} /> </Popconfirm>) }
    ];
    
    const MotionRow = (props: any) => (
        <motion.tr
            {...props}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.3 }}
        />
    );

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card>
                <Space>
                    <Title level={4} style={{ margin: 0 }}>交易流水</Title>
                    <DatePicker onChange={handleDateChange} picker="month" value={pickerValue} allowClear/>
                </Space>
            </Card>

            <Card>
                {/* 【修复问题三】添加 scroll 属性，让表格在内容溢出时内部滚动 */}
                <Table
                    columns={columns}
                    dataSource={data?.transactions}
                    rowKey="id"
                    loading={loading}
                    pagination={{ pageSize: 15, showTotal: (total) => `共 ${total} 条` }}
                    components={{
                       body: {
                         row: MotionRow,
                       },
                    }}
                    scroll={{ x: 'max-content' }}
                    summary={() => data && data.transactions.length > 0 ? (
                        <Table.Summary.Row>
                            <Table.Summary.Cell index={0} colSpan={2}><b>总计</b></Table.Summary.Cell>
                            <Table.Summary.Cell index={2}>
                                <Space direction="vertical" size="small">
                                    <span style={{ color: '#52c41a' }}>总收入: ¥{data?.summary.total_income.toFixed(2)}</span>
                                    <span style={{ color: '#f5222d' }}>总支出: ¥{data?.summary.total_expense.toFixed(2)}</span>
                                    <b>净结余: ¥{data?.summary.net_balance.toFixed(2)}</b>
                                </Space>
                            </Table.Summary.Cell>
                        </Table.Summary.Row>
                    ) : null}
                />
            </Card>
        </Space>
    );
};

export default TransactionPage;