// src/components/AddTransactionModal.tsx
import React, { useState, useEffect, useMemo } from 'react';
import { Modal, Form, Input, InputNumber, DatePicker, Radio, Select, App, Spin } from 'antd';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getCategories, getLoans, addTransaction, getAccounts } from '../services/api';
import type { Category, LoanResponse, CreateTransactionRequest, Account } from '../types';
import dayjs from 'dayjs';
import axios from 'axios';

interface Props {
  open: boolean;
  onClose: () => void;
}

const AddTransactionModal: React.FC<Props> = ({ open, onClose }) => {
  const [form] = Form.useForm();
  const [transactionType, setTransactionType] = useState<string>('expense');
  const { message: staticMessage } = App.useApp();
  const queryClient = useQueryClient();

  const { data: categories = [], isLoading: isLoadingCategories } = useQuery<Category[]>({
    queryKey: ['categories'],
    queryFn: () => getCategories().then(res => res.data || []),
    enabled: open,
  });

  const { data: loans = [], isLoading: isLoadingLoans } = useQuery<LoanResponse[]>({
    queryKey: ['loans'],
    queryFn: () => getLoans().then(res => (res.data || []).filter(l => l.status === 'active')),
    enabled: open,
  });

  const { data: accounts = [], isLoading: isLoadingAccounts } = useQuery<Account[]>({
    queryKey: ['accounts'],
    queryFn: () => getAccounts().then(res => res.data || []),
    enabled: open,
  });

  useEffect(() => {
    if (open) {
      form.resetFields();
      form.setFieldsValue({
        type: 'expense',
        transaction_date: dayjs(),
      });
      setTransactionType('expense');
    }
  }, [open, form]);
  
  const addMutation = useMutation({
    mutationFn: addTransaction,
    onSuccess: () => {
        staticMessage.success('流水记录创建成功！');
        // 使所有相关查询失效，自动刷新数据
        queryClient.invalidateQueries({ queryKey: ['transactions'] });
        queryClient.invalidateQueries({ queryKey: ['accounts'] });
        queryClient.invalidateQueries({ queryKey: ['dashboardCards'] });
        queryClient.invalidateQueries({ queryKey: ['analyticsCharts'] });
        queryClient.invalidateQueries({ queryKey: ['dashboardWidgets'] });
        queryClient.invalidateQueries({ queryKey: ['loans'] }); // 还款会影响贷款
        queryClient.invalidateQueries({ queryKey: ['budgets'] }); // 流水会影响预算
        onClose();
    },
    onError: (err: unknown) => {
        const errorMsg = axios.isAxiosError(err) && err.response ? err.response.data.error : '添加失败';
        staticMessage.error(errorMsg);
    }
  });


  const handleOk = () => {
    form.validateFields()
      .then(values => {
        if (accounts.length === 0 && (values.type === 'income' || values.type === 'expense' || values.type === 'repayment' || values.type === 'transfer')) {
            staticMessage.error('您还没有创建任何资金账户，请先到“账户管理”页面添加账户！');
            return;
        }

        const postData: CreateTransactionRequest = {
          ...values,
          transaction_date: values.transaction_date.format('YYYY-MM-DD'),
          amount: parseFloat(values.amount)
        };
        addMutation.mutate(postData);
      })
      .catch(info => {
        console.log('Validate Failed:', info);
      });
  };

  const onTypeChange = (e: any) => {
    const newType = e.target.value;
    setTransactionType(newType);
    form.setFieldsValue({
      category_id: undefined,
      related_loan_id: undefined,
      from_account_id: undefined,
      to_account_id: undefined
    });
  };

  const filteredCategories = useMemo(() => {
    return categories.filter(c => c.type === transactionType);
  }, [categories, transactionType]);
  
  const isLoading = isLoadingCategories || isLoadingLoans || isLoadingAccounts;

  const renderConditionalFields = () => {
    if (accounts.length === 0 && (transactionType !== 'settlement')) {
        return (
            <Form.Item label={transactionType === 'transfer' ? '账户' : '付款账户'}>
                <div style={{ color: '#ff4d4f' }}>
                    您还没有任何资金账户。请先到“账户管理”页面添加。
                </div>
            </Form.Item>
        )
    }

    switch (transactionType) {
      case 'income':
        return (
          <>
            <Form.Item name="to_account_id" label="收款账户" rules={[{ required: true, message: '请选择收款账户' }]}>
              <Select placeholder="选择资金流入的账户">
                {accounts.map(acc => (
                  <Select.Option key={acc.id} value={acc.id}>{acc.name}</Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="category_id" label="收入分类" rules={[{ required: true, message: '请选择分类' }]}>
              <Select placeholder="选择收入来源分类">
                {filteredCategories.map(cat => (
                  <Select.Option key={cat.id} value={cat.id}>{cat.name}</Select.Option>
                ))}
              </Select>
            </Form.Item>
          </>
        );
      case 'expense':
        return (
          <>
            <Form.Item name="from_account_id" label="付款账户" rules={[{ required: true, message: '请选择付款账户' }]}>
              <Select placeholder="选择资金来源账户">
                {accounts.map(acc => (
                  <Select.Option key={acc.id} value={acc.id}>{`${acc.name} (余额: ¥${acc.balance.toFixed(2)})`}</Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="category_id" label="支出分类" rules={[{ required: true, message: '请选择分类' }]}>
              <Select placeholder="选择支出用途分类">
                {filteredCategories.map(cat => (
                  <Select.Option key={cat.id} value={cat.id}>{cat.name}</Select.Option>
                ))}
              </Select>
            </Form.Item>
          </>
        );
      case 'repayment':
        return (
          <>
            <Form.Item name="from_account_id" label="付款账户" rules={[{ required: true, message: '请选择扣款账户' }]}>
              <Select placeholder="选择资金来源账户">
                {accounts.map(acc => (
                  <Select.Option key={acc.id} value={acc.id}>{`${acc.name} (余额: ¥${acc.balance.toFixed(2)})`}</Select.Option>
                ))}
              </Select>
            </Form.Item>
             <Form.Item name="related_loan_id" label="关联借款" rules={[{ required: true, message: '请选择关联的借款' }]}>
              <Select placeholder="选择要偿还的借款" disabled={loans.length === 0}>
                {loans.map(loan => (
                  <Select.Option key={loan.id} value={loan.id}>{`${loan.description || `贷款 #${loan.id}`} (待还: ¥${loan.outstanding_balance.toFixed(2)})`}</Select.Option>
                ))}
              </Select>
            </Form.Item>
          </>
        );
      case 'transfer':
         return (
          <>
            <Form.Item name="from_account_id" label="从账户" rules={[{ required: true, message: '请选择转出账户' }]}>
              <Select placeholder="选择转出账户">{accounts.map(acc => <Select.Option key={acc.id} value={acc.id}>{`${acc.name} (余额: ¥${acc.balance.toFixed(2)})`}</Select.Option>)}</Select>
            </Form.Item>
            <Form.Item name="to_account_id" label="到账户" rules={[{ required: true, message: '请选择转入账户' }]}>
               <Select placeholder="选择转入账户">{accounts.map(acc => <Select.Option key={acc.id} value={acc.id}>{acc.name}</Select.Option>)}</Select>
            </Form.Item>
          </>
        );
      default:
        return null;
    }
  };

  return (
    <Modal
      title="记一笔"
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      destroyOnHidden
      confirmLoading={addMutation.isPending}
    >
      <Spin spinning={isLoading}>
        <Form form={form} layout="vertical">
          <Form.Item name="type" label="类型" rules={[{ required: true }]}>
            <Radio.Group onChange={onTypeChange}>
              <Radio.Button value="expense">支出</Radio.Button>
              <Radio.Button value="income">收入</Radio.Button>
              <Radio.Button value="transfer">转账</Radio.Button>
              <Radio.Button value="repayment">还款</Radio.Button>
            </Radio.Group>
          </Form.Item>

          <Form.Item name="amount" label="金额" rules={[{ required: true, message: '请输入金额' }]}>
            <InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} />
          </Form.Item>

          <Form.Item name="transaction_date" label="日期" rules={[{ required: true, message: '请选择日期' }]}>
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
          
          {renderConditionalFields()}

          <Form.Item name="description" label="备注">
            <Input.TextArea rows={2} placeholder="选填，最多100字" maxLength={100} />
          </Form.Item>
        </Form>
      </Spin>
    </Modal>
  );
};

const AddTransactionModalWrapper: React.FC<Props> = (props) => (
    <App>
        <AddTransactionModal {...props} />
    </App>
);

export default AddTransactionModalWrapper;