// src/pages/DashboardPage.tsx
import React, { useState, useMemo } from 'react';
import { Space, Row, Col, Card, Empty, DatePicker, Radio, Statistic, Tooltip, Typography, Progress, List, Tag, Divider, Skeleton } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined } from '@ant-design/icons';
import { Line, Pie } from '@ant-design/charts';
import { useQuery } from '@tanstack/react-query';
import { getDashboardCards, getAnalyticsCharts, getDashboardWidgets } from '../services/api';
import type { DashboardCard, AnalyticsChartsResponse, DashboardWidgetsResponse, DashboardBudgetSummary, DashboardLoanInfo, ChartDataPoint } from '../types';
import IconDisplay from '../components/IconPicker';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { getSteppedColor } from '../utils/colorUtils';
import { motion } from 'framer-motion';
import weekOfYear from 'dayjs/plugin/weekOfYear';

dayjs.extend(weekOfYear);

const { Title, Text } = Typography;
const { WeekPicker } = DatePicker;

// --- 辅助函数部分保持不变 ---
const fillMissingDaysForWeek = (data: ChartDataPoint[], weekStart: Dayjs): ChartDataPoint[] => {
    if (!data) return [];
    const dataMap = new Map(data.map(item => [parseInt(item.name, 10), item.value]));
    const result: ChartDataPoint[] = [];
    for (let i = 0; i < 7; i++) {
        const currentDay = weekStart.add(i, 'day');
        const dayOfMonth = currentDay.date();
        const key = currentDay.format('MM-DD');
        result.push({ name: key, value: dataMap.get(dayOfMonth) || 0 });
    }
    return result;
}
const fillMissingDays = (data: ChartDataPoint[], year: number, month: number): ChartDataPoint[] => {
    if (!data) return [];
    const dateMap = new Map(data.map(item => [parseInt(item.name, 10), item.value]));
    const daysInMonth = dayjs(`${year}-${month}`).daysInMonth();
    const result: ChartDataPoint[] = [];
    for (let i = 1; i <= daysInMonth; i++) {
        result.push({ name: `${i}`, value: dateMap.get(i) || 0 });
    }
    return result;
};
const fillMissingMonths = (data: ChartDataPoint[], year: number): ChartDataPoint[] => {
    if (!data) return [];
    const dateMap = new Map(data.map(item => [item.name, item.value]));
    const result: ChartDataPoint[] = [];
    for (let i = 1; i <= 12; i++) {
        const monthStr = i.toString().padStart(2, '0');
        const key = `${year}-${monthStr}`;
        result.push({ name: key, value: dateMap.get(key) || 0 });
    }
    return result;
};
// --- 其他组件部分保持不变 ---
const MotionCol = motion(Col);
const containerVariants = { hidden: { opacity: 0 }, visible: { opacity: 1, transition: { staggerChildren: 0.1 } } } as const;
const itemVariants = { hidden: { y: 20, opacity: 0 }, visible: { y: 0, opacity: 1, transition: { type: 'spring', stiffness: 100 } } } as const;
const StatCard: React.FC<{ item: DashboardCard }> = ({ item }) => { const color = getSteppedColor(item.title === '总支出' ? -item.value : item.value); let percentageChange: number | null = null; if (item.title !== '总存款' && item.title !== '总借款') { if (item.prev_value !== 0) { percentageChange = ((item.value - item.prev_value) / Math.abs(item.prev_value)) * 100; } else if (item.value !== 0) { percentageChange = 100; } } return ( <motion.div whileHover={{ scale: 1.03 }} transition={{ type: 'spring', stiffness: 300 }} style={{height: '100%'}}> <Card style={{ boxShadow: 'var(--card-shadow)', height: '100%' }}> <Statistic title={<span style={{fontSize: 14}}>{item.title} {item.title === '总存款' && item.meta?.account_count !== undefined && (<Text type="secondary"> ({item.meta.account_count}个账户)</Text>)}</span>} value={item.value} precision={2} prefix="¥" valueStyle={{ color, fontSize: 24, fontWeight: 500 }} suffix={ percentageChange !== null ? ( <Tooltip title={`与上期 (¥${item.prev_value.toFixed(2)}) 比较`}> <span style={{ color: percentageChange >= 0 ? '#52c41a' : '#f5222d', fontSize: 14, marginLeft: 8 }}> {percentageChange >= 0 ? <ArrowUpOutlined /> : <ArrowDownOutlined />} {Math.abs(percentageChange).toFixed(1)}% </span> </Tooltip> ) : null } /> <div style={{ position: 'absolute', right: 20, top: '50%', transform: 'translateY(-50%)', fontSize: 32, color: 'rgba(0,0,0,.08)' }}> <IconDisplay name={item.icon} /> </div> </Card> </motion.div> ); };
const BudgetProgressCard: React.FC<{ budget: DashboardBudgetSummary }> = ({ budget }) => { const title = budget.period === 'monthly' ? "本月总预算" : "本年总预算"; return ( <Card title={title} size="small" style={{height: '100%'}}> {budget.is_set ? ( <> <Tooltip title={`进度: ${(budget.progress * 100).toFixed(1)}%`}> <Progress percent={Math.round(budget.progress * 100)} strokeColor={budget.progress > 1 ? '#ff4d4f' : '#2f54eb'} status={budget.progress > 1 ? 'exception' : 'active'} /> </Tooltip> <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px', color: 'rgba(0, 0, 0, 0.45)', marginTop: '8px' }}> <span>已用: ¥{budget.spent.toFixed(2)}</span> <span>预算: ¥{budget.amount.toFixed(2)}</span> </div> </> ) : ( <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="未设置预算" /> )} </Card> ); };
const LoanWidget: React.FC<{ loan: DashboardLoanInfo }> = ({ loan }) => { let timeProgress = 0; let timeStatus: 'normal' | 'success' | 'exception' = 'normal'; let daysRemainingText = '未设置计划还款日'; if (loan.repayment_date) { const today = dayjs(); const startDate = dayjs(loan.loan_date); const endDate = dayjs(loan.repayment_date); if (endDate.isAfter(startDate)) { const totalDays = endDate.diff(startDate, 'day'); const elapsedDays = today.diff(startDate, 'day'); timeProgress = Math.min(Math.max(0, (elapsedDays / totalDays) * 100), 100); if (today.isAfter(endDate)) { timeStatus = 'exception'; daysRemainingText = `已逾期 ${today.diff(endDate, 'day')} 天`; } else { timeStatus = 'normal'; daysRemainingText = `剩余 ${endDate.diff(today, 'day')} 天`; } } } return ( <List.Item> <Space direction="vertical" style={{width: '100%'}}> <Space style={{width: '100%', justifyContent: 'space-between'}}> <Text strong>{loan.description || `贷款 #${loan.id}`}</Text> <Tag color="orange">待还: ¥{loan.outstanding_balance.toFixed(2)}</Tag> </Space> <Tooltip title={`已还款 ${ (loan.repayment_amount_progress * 100).toFixed(1) }%`}> <Progress percent={Math.round(loan.repayment_amount_progress * 100)} size="small" /> </Tooltip> {loan.repayment_date && ( <Tooltip title={daysRemainingText}> <Progress percent={timeProgress} size="small" status={timeStatus} format={() => daysRemainingText} /> </Tooltip> )} </Space> </List.Item> ); };


const DashboardPage: React.FC = () => {
    const [pickerType, setPickerType] = useState<'week' | 'month' | 'year' | 'all'>('month');
    const [datePickerValue, setDatePickerValue] = useState<Dayjs | null>(dayjs());

    const apiFilter = useMemo(() => {
        if (!datePickerValue) return {};
        if (pickerType === 'all') return {};
        return { year: datePickerValue.year(), month: datePickerValue.month() + 1 };
    }, [datePickerValue, pickerType]);

    const { data: cardsData, isLoading: isLoadingCards } = useQuery({ queryKey: ['dashboardCards', apiFilter], queryFn: () => getDashboardCards(apiFilter).then(res => { const cardOrder = ['总收入', '总支出', '净结余', '总存款']; return (res.data || []).sort((a, b) => cardOrder.indexOf(a.title) - cardOrder.indexOf(b.title)); }), });
    const { data: charts, isLoading: isLoadingCharts } = useQuery<AnalyticsChartsResponse, Error>({ queryKey: ['analyticsCharts', apiFilter], queryFn: () => getAnalyticsCharts(apiFilter).then(res => res.data) });
    const { data: widgets, isLoading: isLoadingWidgets } = useQuery<DashboardWidgetsResponse, Error>({ queryKey: ['dashboardWidgets', apiFilter], queryFn: () => getDashboardWidgets(apiFilter).then(res => res.data) });

    const processedChartData = useMemo(() => {
        if (!charts) return { expenseTrend: [], categoryExpense: [] };
        const rawTrendData = charts.expense_trend || [];
        let trendData: ChartDataPoint[] = [];
        if (pickerType === 'week' && datePickerValue) {
            const weekStart = datePickerValue.startOf('week');
            trendData = fillMissingDaysForWeek(rawTrendData, weekStart);
        } else if (pickerType === 'month' && datePickerValue) { 
            trendData = fillMissingDays(rawTrendData, datePickerValue.year(), datePickerValue.month() + 1); 
        } else if (pickerType === 'year' && datePickerValue) { 
            trendData = fillMissingMonths(rawTrendData, datePickerValue.year()); 
        } else {
            trendData = rawTrendData;
        }

        return { expenseTrend: trendData, categoryExpense: charts.category_expense || [] };
    }, [charts, pickerType, datePickerValue]);

    const handlePickerTypeChange = (e: any) => {
        const newType = e.target.value;
        setPickerType(newType);
        if (newType === 'all') { 
            setDatePickerValue(null); 
        } else {
            setDatePickerValue(dayjs());
        }
    }
    
    const handleDateChange = (date: Dayjs | null) => { 
        setDatePickerValue(date); 
    };
    
    const lineChartConfig = { 
        data: processedChartData.expenseTrend, 
        xField: 'name', 
        yField: 'value', 
        smooth: true, 
        height: 250, 
        area: { style: { fill: 'l(270) 0:#ffffff 1:#bae0ff' } }, 
        line: { style: { stroke: '#2f54eb', lineWidth: 2 } }, 
        tooltip: {
            title: (d: ChartDataPoint) => d.name,
            items: [{ channel: 'y', name: '支出', valueFormatter: (d: number) => `¥ ${d.toFixed(2)}` }]
        },
    };

    const pieChartConfig = { 
        data: processedChartData.categoryExpense, 
        angleField: 'value', 
        colorField: 'name', 
        
        // 【核心修改】通过 padding 来控制图表大小
        padding: 'auto', // 或者可以尝试一个具体的数值，如 40
        appendPadding: 30, // 在外围增加一些额外的边距

        radius: 0.5, // 保持环形图的半径
        innerRadius: 0.7, 
        height: 250, 
        legend: { position: 'top', layout: 'horizontal' } as const, 
        label: false,
        interactions: [{ type: 'element-active' }], 
        statistic: { 
            title: { content: '总支出' }, 
            content: { formatter: (_: any, data?: ChartDataPoint[]) => `¥${(data?.reduce((s, d) => s + (d?.value || 0), 0) || 0).toFixed(2)}` } 
        },
        tooltip: {
            items: [
                (item: ChartDataPoint) => {
                    if (item.name && item.value) {
                        return {
                            name: item.name,
                            value: `¥ ${Number(item.value).toFixed(2)}`
                        };
                    }
                    return null;
                },
            ]
        },
    };


    const getFilterTitle = () => { 
        if (pickerType === 'week' && datePickerValue) return `${datePickerValue.year()}年 第${datePickerValue.week()}周`;
        if (pickerType === 'month' && datePickerValue) return `${datePickerValue.year()}年${datePickerValue.month() + 1}月`; 
        if (pickerType === 'year' && datePickerValue) return `${datePickerValue.year()}年`; 
        return '全部时间'; 
    }

    const renderDatePicker = () => {
        if (pickerType === 'week') {
            return <WeekPicker onChange={handleDateChange} value={datePickerValue} allowClear={false} />;
        }
        if (pickerType === 'month' || pickerType === 'year') {
            return <DatePicker picker={pickerType} onChange={handleDateChange} value={datePickerValue} allowClear={false} />;
        }
        return null;
    }

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Card>
                <Space wrap>
                    <span>筛选周期:</span>
                    <Radio.Group value={pickerType} onChange={handlePickerTypeChange}>
                        <Radio.Button value="all">全部</Radio.Button>
                        <Radio.Button value="week">按周</Radio.Button>
                        <Radio.Button value="month">按月</Radio.Button>
                        <Radio.Button value="year">按年</Radio.Button>
                    </Radio.Group>
                    {renderDatePicker()}
                </Space>
            </Card>
            
            <motion.div variants={containerVariants} initial="hidden" animate="visible">
                <Row gutter={[24, 24]}>{isLoadingCards ? Array(4).fill(0).map((_, i) => <Col key={i} xs={24} sm={12} xl={6}><Card><Skeleton active paragraph={{ rows: 2 }} /></Card></Col>) : cardsData?.map((item) => (<MotionCol key={item.title} variants={itemVariants} xs={24} sm={12} xl={6}><StatCard item={item} /></MotionCol>))} </Row>
            </motion.div>

            <Row gutter={[24, 24]}>
                <Col xs={24} lg={12}><Title level={5}>预算总览</Title>{isLoadingWidgets ? <Card><Skeleton active /></Card> : (widgets?.budgets && widgets.budgets.length > 0 ? <Row gutter={[16, 16]}>{widgets.budgets.map(b => <Col xs={24} sm={12} key={b.period}><BudgetProgressCard budget={b} /></Col>)}</Row> : <Card><Empty description="未设置预算" /></Card>)}</Col>
                <Col xs={24} lg={12}><Title level={5}>在贷情况</Title>{isLoadingWidgets ? <Card><Skeleton active /></Card> : (<Card styles={{body: {padding: '1px 24px'}}}>{widgets?.loans && widgets.loans.length > 0 ? <List itemLayout="horizontal" dataSource={widgets.loans} renderItem={(item) => <LoanWidget loan={item} />} /> : <Empty description="恭喜！暂无在贷记录"/>}</Card>)}</Col>
            </Row>
            
            <Divider orientation="left" plain><Title level={5} style={{color: '#8c8c8c'}}>{getFilterTitle()} 数据图表</Title></Divider>
            
            <Row gutter={[24, 24]}>
                <Col xs={24} lg={14}>
                    <Card styles={{body: { minHeight: 298 }}}>
                        <Title level={5}>支出趋势</Title>
                        {isLoadingCharts ? <Skeleton active /> : (processedChartData.expenseTrend.length > 0 && processedChartData.expenseTrend.some(p => p.value > 0) ? <Line {...lineChartConfig} /> : <Empty description="当前时段无支出趋势" />)}
                    </Card>
                </Col>
                <Col xs={24} lg={10}>
                    <Card styles={{body: { minHeight: 298 }}}>
                        <Title level={5}>支出分类</Title>
                        {isLoadingCharts ? <Skeleton active /> : (processedChartData.categoryExpense.length > 0 ? <Pie {...pieChartConfig} key={JSON.stringify(processedChartData.categoryExpense)} /> : <Empty description="当前时段无支出数据" />)}
                    </Card>
                </Col>
            </Row>
        </Space>
    );
};

export default DashboardPage;