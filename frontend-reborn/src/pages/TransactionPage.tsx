// src/pages/TransactionPage.tsx
import React, { useState, useEffect, useCallback } from 'react';
import { Card, Table, Tag, DatePicker, message, Space, Popconfirm, Button } from 'antd';
import { getTransactions, deleteTransaction } from '../services/api';
import type { GetTransactionsResponse, Transaction } from '../types';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { DeleteOutlined } from '@ant-design/icons';

const { RangePicker } = DatePicker;

const TransactionPage: React.FC = () => {
    const [data, setData] = useState<GetTransactionsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [dateFilter, setDateFilter] = useState<{ year?: number, month?: number }>({
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
        if (date) {
            setDateFilter({ year: date.year(), month: date.month() + 1 });
        } else {
            setDateFilter({});
        }
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
        {
            title: '日期',
            dataIndex: 'transaction_date',
            key: 'transaction_date',
            sorter: (a, b) => dayjs(a.transaction_date).unix() - dayjs(b.transaction_date).unix(),
        },
        {
            title: '类型',
            dataIndex: 'type',
            key: 'type',
            filters: [
                { text: '收入', value: 'income' },
                { text: '支出', value: 'expense' },
                { text: '还款', value: 'repayment' },
            ],
            onFilter: (value, record) => record.type === value,
            render: (type) => {
                let color = 'blue';
                if (type === 'expense') color = 'red';
                if (type === 'income') color = 'green';
                return <Tag color={color}>{type.toUpperCase()}</Tag>;
            }
        },
        {
            title: '金额',
            dataIndex: 'amount',
            key: 'amount',
            render: (amount, record) => (
                <span style={{ color: record.type === 'income' ? 'green' : 'red' }}>
                    {record.type === 'income' ? '+' : '-'} ¥{amount.toFixed(2)}
                </span>
            ),
            sorter: (a, b) => a.amount - b.amount,
        },
        {
            title: '分类',
            dataIndex: 'category_name',
            key: 'category_name',
        },
        {
            title: '描述',
            dataIndex: 'description',
            key: 'description',
        },
        {
            title: '操作',
            key: 'action',
            render: (_, record) => (
                <Popconfirm title="确定删除这条记录吗?" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
                    <Button type="link" danger icon={<DeleteOutlined />} />
                </Popconfirm>
            )
        }
    ];

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card>
                <Space>
                    <span>选择月份:</span>
                    <DatePicker onChange={handleDateChange} picker="month" value={pickerValue} />
                </Space>
            </Card>

            <Card>
                <Table
                    columns={columns}
                    dataSource={data?.transactions}
                    rowKey="id"
                    loading={loading}
                    pagination={{ pageSize: 15, showTotal: (total) => `共 ${total} 条` }}
                    summary={() => (
                        <Table.Summary.Row>
                            <Table.Summary.Cell index={0} colSpan={2}><b>总计</b></Table.Summary.Cell>
                            <Table.Summary.Cell index={2}>
                                <Space direction="vertical">
                                    <span style={{ color: 'green' }}>总收入: ¥{data?.summary.total_income.toFixed(2)}</span>
                                    <span style={{ color: 'red' }}>总支出: ¥{data?.summary.total_expense.toFixed(2)}</span>
                                    <b>净结余: ¥{data?.summary.net_balance.toFixed(2)}</b>
                                </Space>
                            </Table.Summary.Cell>
                        </Table.Summary.Row>
                    )}
                />
            </Card>
        </Space>
    );
};

export default TransactionPage;