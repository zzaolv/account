// src/pages/DashboardPage.tsx
import React, { useState, useEffect, useCallback } from 'react';
import { Space, message, Row, Col, Card, Empty, Spin, DatePicker, Radio, Statistic, Tooltip, Typography, Progress, List, Tag, Divider, Skeleton } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined } from '@ant-design/icons';
import { Line, Pie } from '@ant-design/charts';
import { getDashboardCards, getAnalyticsCharts, getDashboardWidgets } from '../services/api';
import type { DashboardCard, AnalyticsChartsResponse, DashboardWidgetsResponse, DashboardBudgetSummary, DashboardLoanInfo, ChartDataPoint } from '../types';
import IconDisplay from '../components/IconPicker';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { getSteppedColor } from '../utils/colorUtils'; // 【新增】导入阶梯颜色工具

const { Title, Text } = Typography;

// --- 子组件 ---
const StatCard: React.FC<{ item: DashboardCard }> = ({ item }) => {
    // 【修改】使用阶梯颜色
    const color = getSteppedColor(item.title === '总支出' ? -item.value : item.value);

    let percentageChange: number | null = null;
    if (item.title !== '总存款' && item.title !== '总借款') {
         if (item.prev_value !== 0) {
            percentageChange = ((item.value - item.prev_value) / Math.abs(item.prev_value)) * 100;
        } else if (item.value !== 0) {
            percentageChange = 100;
        }
    }

    return ( 
        <Card bordered={false} style={{ boxShadow: '0 2px 8px rgba(0, 0, 0, 0.09)' }}> 
            <Statistic 
                title={
                    <span style={{fontSize: 14}}>
                        {item.title} 
                        {item.title === '总存款' && item.meta?.account_count !== undefined && (
                            <Text type="secondary"> ({item.meta.account_count}个账户)</Text>
                        )}
                    </span>
                } 
                value={item.value} 
                precision={2} 
                prefix="¥" 
                valueStyle={{ color, fontSize: 24, fontWeight: 500 }} 
                suffix={ 
                    percentageChange !== null ? ( 
                        <Tooltip title={`与上期 (¥${item.prev_value.toFixed(2)}) 比较`}> 
                            <span style={{ color: percentageChange >= 0 ? '#52c41a' : '#f5222d', fontSize: 14, marginLeft: 8 }}> 
                                {percentageChange >= 0 ? <ArrowUpOutlined /> : <ArrowDownOutlined />} {Math.abs(percentageChange).toFixed(1)}% 
                            </span> 
                        </Tooltip> 
                    ) : null 
                } 
            /> 
            <div style={{ position: 'absolute', right: 20, top: '50%', transform: 'translateY(-50%)', fontSize: 32, color: 'rgba(0,0,0,.1)' }}> 
                <IconDisplay name={item.icon} /> 
            </div> 
        </Card> 
    );
};
const BudgetProgressCard: React.FC<{ budget: DashboardBudgetSummary }> = ({ budget }) => { 
    const title = budget.period === 'monthly' ? "本月总预算" : "本年总预算";
    return ( <Card title={title} size="small"> {budget.is_set ? ( <> <Tooltip title={`进度: ${(budget.progress * 100).toFixed(1)}%`}> <Progress percent={Math.round(budget.progress * 100)} strokeColor={budget.progress > 1 ? '#ff4d4f' : '#1677ff'} status={budget.progress > 1 ? 'exception' : 'normal'} /> </Tooltip> <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px', color: 'rgba(0, 0, 0, 0.45)', marginTop: '8px' }}> <span>已用: ¥{budget.spent.toFixed(2)}</span> <span>预算: ¥{budget.amount.toFixed(2)}</span> </div> </> ) : ( <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="未设置预算" /> )} </Card> );
};
const LoanWidget: React.FC<{ loan: DashboardLoanInfo }> = ({ loan }) => {
    let timeProgress = 0;
    let timeStatus: 'normal' | 'success' | 'exception' = 'normal';
    let daysRemainingText = '未设置计划还款日';
    if (loan.repayment_date) {
        const today = dayjs(); const startDate = dayjs(loan.loan_date); const endDate = dayjs(loan.repayment_date);
        if (endDate.isAfter(startDate)) {
            const totalDays = endDate.diff(startDate, 'day'); const elapsedDays = today.diff(startDate, 'day');
            timeProgress = Math.min(Math.max(0, (elapsedDays / totalDays) * 100), 100);
            if (today.isAfter(endDate)) {
                timeStatus = 'exception'; daysRemainingText = `已逾期 ${today.diff(endDate, 'day')} 天`;
            } else {
                timeStatus = 'normal'; daysRemainingText = `剩余 ${endDate.diff(today, 'day')} 天`;
            }
        }
    }
    return ( <List.Item> <Space direction="vertical" style={{width: '100%'}}> <Space style={{width: '100%', justifyContent: 'space-between'}}> <Text strong>{loan.description || `贷款 #${loan.id}`}</Text> <Tag color="orange">待还: ¥{loan.outstanding_balance.toFixed(2)}</Tag> </Space> <Tooltip title={`已还款 ${ (loan.repayment_amount_progress * 100).toFixed(1) }%`}> <Progress percent={Math.round(loan.repayment_amount_progress * 100)} size="small" /> </Tooltip> {loan.repayment_date && ( <Tooltip title={daysRemainingText}> <Progress percent={timeProgress} size="small" status={timeStatus} format={() => daysRemainingText} /> </Tooltip> )} </Space> </List.Item> );
};

// --- 主页面组件 ---
const DashboardPage: React.FC = () => {
    const [cards, setCards] = useState<DashboardCard[]>([]);
    const [charts, setCharts] = useState<AnalyticsChartsResponse | null>(null);
    const [widgets, setWidgets] = useState<DashboardWidgetsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [pickerType, setPickerType] = useState<'month' | 'year' | 'all'>('month');
    const [filter, setFilter] = useState<{ year?: number, month?: number }>({
        year: dayjs().year(),
        month: dayjs().month() + 1,
    });

    const fetchData = useCallback(async () => {
        setLoading(true);
        try {
            const [cardsRes, chartsRes, widgetsRes] = await Promise.all([
                getDashboardCards(filter),
                getAnalyticsCharts(filter),
                getDashboardWidgets(filter)
            ]);
            // 【修改】后端返回的卡片现在可能包含总存款，需要正确排序
            const cardOrder = ['总收入', '总支出', '净结余', '总存款', '总借款'];
            const sortedCards = (cardsRes.data || []).sort((a, b) => {
                return cardOrder.indexOf(a.title) - cardOrder.indexOf(b.title);
            });

            setCards(sortedCards || []);
            setCharts(chartsRes.data || null);
            setWidgets(widgetsRes.data || null);
        } catch (error) {
            message.error('获取仪表盘数据失败');
        } finally {
            setLoading(false);
        }
    }, [filter]);

    useEffect(() => { fetchData(); }, [fetchData]);

    const handlePickerTypeChange = (e: any) => {
        const newType = e.target.value;
        setPickerType(newType);
        if (newType === 'all') {
            setFilter({});
        } else {
            const now = dayjs();
            const newFilter = newType === 'month'
                ? { year: now.year(), month: now.month() + 1 }
                : { year: now.year() };
            setFilter(newFilter);
        }
    }

    const handleDateChange = (date: Dayjs | null) => {
        if (!date) {
            setPickerType('all');
            setFilter({});
            return;
        }
        if (pickerType === 'month') { setFilter({ year: date.year(), month: date.month() + 1 }); }
        else if (pickerType === 'year') { setFilter({ year: date.year() }); }
    };
    
    // ======================= 折线图配置 =======================
    const lineChartConfig = {
        data: charts?.expense_trend || [],
        xField: 'name',
        yField: 'value',
        smooth: true,
        height: 250,
        area: { style: { fill: 'l(270) 0:#ffffff 1:#1677ff' } },
        line: { style: { stroke: '#1677ff', lineWidth: 2 } },
        tooltip: false as const,
    };

    // ======================= 饼图配置 =======================
    const pieChartConfig = {
        data: charts?.category_expense || [],
        angleField: 'value',
        colorField: 'name',
        radius: 0.8,
        innerRadius: 0.6,
        height: 250,
        legend: { layout: 'horizontal', position: 'top' } as const,
        label: {
            type: 'outer',
            content: (datum: any) => `${datum.name}: ¥${(datum.value || 0).toFixed(2)}`,
        },
        tooltip: false as const,
        statistic: {
            title: { content: '总支出', style: { fontSize: '14px' } },
            content: {
                style: { fontSize: '20px', fontWeight: 'bold' },
                formatter: (_: any, data: ChartDataPoint[] | undefined) => {
                    const total = data?.reduce((sum, d) => sum + (d?.value || 0), 0) || 0;
                    return `¥ ${total.toFixed(2)}`;
                }
            }
        }
    };
    
    const getFilterTitle = () => {
        if (pickerType === 'month' && filter.year && filter.month) return `${filter.year}年${filter.month}月`;
        if (pickerType === 'year' && filter.year) return `${filter.year}年`;
        return '全部时间';
    }

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card>
                <Space wrap>
                    <span>筛选周期:</span>
                    <Radio.Group value={pickerType} onChange={handlePickerTypeChange}>
                        <Radio.Button value="all">全部时间</Radio.Button>
                        <Radio.Button value="month">按月</Radio.Button>
                        <Radio.Button value="year">按年</Radio.Button>
                    </Radio.Group>
                    {pickerType !== 'all' &&
                        <DatePicker picker={pickerType} onChange={handleDateChange} defaultValue={dayjs()} allowClear={false} />
                    }
                </Space>
            </Card>

            <Row gutter={[16, 16]}>
                {loading ? Array(4).fill(0).map((_, index) => (
                    <Col key={index} xs={24} sm={12} xl={6}>
                        <Card bordered={false} style={{ boxShadow: '0 2px 8px rgba(0, 0, 0, 0.09)' }}><Skeleton active paragraph={{ rows: 2 }} /></Card>
                    </Col>
                )) : cards.map((item, index) => (
                    <Col key={index} xs={24} sm={12} xl={6}>
                        <StatCard item={item} />
                    </Col>
                ))}
            </Row>

            <Spin spinning={loading}>
                <Row gutter={[16, 16]}>
                    <Col xs={24} lg={12}><Title level={5}>预算总览</Title>
                        {widgets?.budgets && widgets.budgets.length > 0 ?
                            <Row gutter={[16, 16]}>{widgets.budgets.map(b => <Col xs={24} sm={12} key={b.period}><BudgetProgressCard budget={b} /></Col>)}</Row>
                            : <Card><Empty description="未设置预算" /></Card>
                        }
                    </Col>
                    <Col xs={24} lg={12}><Title level={5}>在贷情况</Title>
                        <Card>
                            {widgets?.loans && widgets.loans.length > 0 ? <List itemLayout="horizontal" dataSource={widgets.loans} renderItem={(item) => <LoanWidget loan={item} />} /> : <Empty description="恭喜！暂无在贷记录"/>}
                        </Card>
                    </Col>
                </Row>
            </Spin>

            <Divider orientation="left" plain><Title level={5} style={{color: '#8c8c8c'}}>{getFilterTitle()} 数据图表</Title></Divider>

            <Spin spinning={loading}>
                <Row gutter={[16, 16]}>
                    <Col xs={24} lg={14}>
                        <Card title="支出趋势">
                            {lineChartConfig.data.length > 0 ? <Line {...lineChartConfig} /> : <Empty description="当前时段无支出趋势" />}
                        </Card>
                    </Col>
                    <Col xs={24} lg={10}>
                        <Card title="支出分类">
                            {pieChartConfig.data.length > 0 ? <Pie {...pieChartConfig} /> : <Empty description="当前时段无支出数据" />}
                        </Card>
                    </Col>
                </Row>
            </Spin>
        </Space>
    );
};

export default DashboardPage;