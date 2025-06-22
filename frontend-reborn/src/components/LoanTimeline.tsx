// 文件路径: src/components/LoanTimeline.tsx
import { Progress, Tooltip, Typography } from 'antd';
import dayjs from 'dayjs';
// [关键修复] 导入正确的类型！我们需要的是描述一整笔贷款的 DashboardLoanInfo，而不是单次事件的 LoanProgressInfo。
import type { DashboardLoanInfo } from '../types';

const { Text } = Typography;

// [关键修复] 将组件的 props 类型从 LoanProgressInfo 修改为 DashboardLoanInfo。
const LoanTimeline = ({ loan }: { loan: DashboardLoanInfo }) => {
  // 现在，loan 对象上的所有属性（repayment_date, loan_date, description）都是类型安全的，错误将全部消失。
  if (!loan.repayment_date) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <Text style={{ width: '120px' }} ellipsis={{ tooltip: loan.description }}>{loan.description}</Text>
        <Progress percent={0} showInfo={false} />
        <Text type="secondary" style={{ fontSize: '12px', width: '100px', textAlign: 'right' }}>未设还款日</Text>
      </div>
    );
  }

  const today = dayjs();
  const loanDate = dayjs(loan.loan_date);
  const repaymentDate = dayjs(loan.repayment_date);

  if (!loanDate.isValid() || !repaymentDate.isValid()) return null;

  const isOverdue = today.isAfter(repaymentDate, 'day');
  
  const totalDays = repaymentDate.diff(loanDate, 'day');
  const elapsedDays = today.diff(loanDate, 'day');
  let percent = 0;
  if (totalDays > 0) {
      percent = Math.min(Math.max((elapsedDays / totalDays) * 100, 0), 100);
  } else if (today.isSame(loanDate, 'day') || today.isAfter(loanDate)) {
      percent = 100;
  }
  
  const remainingDays = repaymentDate.diff(today, 'day');
  const remainingText = isOverdue ? `已逾期 ${Math.abs(remainingDays)} 天` : `剩余 ${remainingDays} 天`;

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <Text style={{ width: '120px' }} ellipsis={{ tooltip: loan.description }}>{loan.description}</Text>
        <Tooltip title={`进度: ${percent.toFixed(0)}%`}>
            <Progress percent={percent} status={isOverdue ? "exception" : "active"} showInfo={false} />
        </Tooltip>
        <Text type="secondary" style={{ fontSize: '12px', width: '100px', textAlign: 'right' }}>{remainingText}</Text>
    </div>
  );
};

export default LoanTimeline;
