// src/components/AddTransactionModal.tsx
import React, { useState, useEffect } from 'react';
// 【修复】移除未使用的 message 导入
import { Modal, Form, Input, InputNumber, DatePicker, Radio, Select, App } from 'antd';
import { getCategories, getLoans, addTransaction, getAccounts } from '../services/api';
import type { Category, LoanResponse, CreateTransactionRequest, Account } from '../types'; 
import dayjs from 'dayjs';

interface Props {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const AddTransactionModal: React.FC<Props> = ({ open, onClose, onSuccess }) => {
  const [form] = Form.useForm();
  const [transactionType, setTransactionType] = useState('expense');
  const [categories, setCategories] = useState<Category[]>([]);
  const [loans, setLoans] = useState<LoanResponse[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  
  const { message: staticMessage } = App.useApp();

  useEffect(() => {
    if (open) {
      const fetchAllData = async () => {
        try {
          const [catRes, loanRes, accRes] = await Promise.all([
            getCategories(),
            getLoans(),
            getAccounts()
          ]);
          setCategories(catRes.data || []);
          setLoans(loanRes.data.filter(l => l.status === 'active') || []);
          setAccounts(accRes.data || []);
        } catch (error) {
          staticMessage.error('获取基础数据失败');
        }
      };
      
      fetchAllData();

      form.setFieldsValue({
        transaction_date: dayjs(),
        type: 'expense'
      });
      setTransactionType('expense');
    }
  }, [open, form, staticMessage]);

  const handleOk = () => {
    form.validateFields()
      .then(values => {
        const postData: CreateTransactionRequest = {
          ...values,
          transaction_date: values.transaction_date.format('YYYY-MM-DD'),
          amount: parseFloat(values.amount)
        };
        addTransaction(postData)
          .then(() => {
            onSuccess();
            form.resetFields();
          })
          .catch(err => {
            staticMessage.error(err.response?.data?.error || '添加失败');
          });
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
  
  const filteredCategories = categories.filter(c => c.type === transactionType);

  const renderConditionalFields = () => {
    switch (transactionType) {
      case 'repayment':
        return (
          <>
            <Form.Item name="related_loan_id" label="关联借款" rules={[{ required: true, message: '请选择关联的借款' }]}>
              <Select placeholder="选择要偿还的借款">
                {loans.map(loan => (
                  <Select.Option key={loan.id} value={loan.id}>{loan.description || `贷款 #${loan.id}`}</Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="from_account_id" label="扣款账户" rules={[{ required: true, message: '请选择扣款账户' }]}>
              <Select placeholder="选择资金来源账户">
                {accounts.map(acc => (
                  <Select.Option key={acc.id} value={acc.id}>{`${acc.name} (余额: ¥${acc.balance.toFixed(2)})`}</Select.Option>
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
      case 'income':
      case 'expense':
      default:
        return (
          <Form.Item name="category_id" label="分类" rules={[{ required: true, message: '请选择分类' }]}>
            <Select placeholder="选择分类">
              {filteredCategories.map(cat => (
                <Select.Option key={cat.id} value={cat.id}>{cat.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
        );
    }
  };

  return (
    <Modal
      title="记一笔"
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item name="type" label="类型" rules={[{ required: true }]}>
          <Radio.Group onChange={onTypeChange}>
            <Radio.Button value="expense">支出</Radio.Button>
            <Radio.Button value="income">收入</Radio.Button>
            <Radio.Button value="repayment">还款</Radio.Button>
            <Radio.Button value="transfer">转账</Radio.Button>
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
    </Modal>
  );
};

export default AddTransactionModal;