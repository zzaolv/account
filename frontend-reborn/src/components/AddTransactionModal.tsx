// src/components/AddTransactionModal.tsx
import React, { useState, useEffect } from 'react';
import { Modal, Form, Input, InputNumber, DatePicker, Radio, Select, message } from 'antd';
import { getCategories, getLoans, addTransaction } from '../services/api';
// ✨ 现在导入的是正确的类型定义
import type { Category, LoanResponse, CreateTransactionRequest } from '../types'; 
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
  const [loans, setLoans] = useState<LoanResponse[]>([]); // ✨ 使用正确的 LoanResponse 类型

  useEffect(() => {
    if (open) {
      getCategories().then(res => setCategories(res.data || [])).catch(() => message.error('获取分类失败'));
      // ✨ 过滤 active 状态的借贷
      getLoans().then(res => setLoans(res.data.filter(l => l.status === 'active') || [])).catch(() => message.error('获取借贷列表失败'));
      form.setFieldsValue({
        transaction_date: dayjs(),
        type: 'expense'
      });
      setTransactionType('expense');
    }
  }, [open, form]);

  const handleOk = () => {
    form.validateFields()
      .then(values => {
        const postData: CreateTransactionRequest = { // ✨ 使用正确的请求类型
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
            message.error(err.response?.data?.error || '添加失败');
          });
      })
      .catch(info => {
        console.log('Validate Failed:', info);
      });
  };

  const onTypeChange = (e: any) => {
    const newType = e.target.value;
    setTransactionType(newType);
    form.setFieldsValue({ category_id: undefined, related_loan_id: undefined });
  };

  const filteredCategories = categories.filter(c => c.type === transactionType);

  return (
    <Modal
      title="记一笔"
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Form.Item name="type" label="类型" rules={[{ required: true }]}>
          <Radio.Group onChange={onTypeChange}>
            <Radio.Button value="expense">支出</Radio.Button>
            <Radio.Button value="income">收入</Radio.Button>
            <Radio.Button value="repayment">还款</Radio.Button>
          </Radio.Group>
        </Form.Item>

        <Form.Item name="amount" label="金额" rules={[{ required: true, message: '请输入金额' }]}>
          <InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} />
        </Form.Item>

        <Form.Item name="transaction_date" label="日期" rules={[{ required: true, message: '请选择日期' }]}>
          <DatePicker style={{ width: '100%' }} />
        </Form.Item>

        {transactionType === 'repayment' ? (
          <Form.Item name="related_loan_id" label="关联借款" rules={[{ required: true, message: '请选择关联的借款' }]}>
            <Select placeholder="选择要偿还的借款">
              {/* ✨ loan.description 现在可能是 null，需要处理 */}
              {loans.map(loan => (
                <Select.Option key={loan.id} value={loan.id}>{loan.description || `贷款 #${loan.id}`}</Select.Option>
              ))}
            </Select>
          </Form.Item>
        ) : (
          // Select 的 value 现在是字符串类型的 ID，这之前就是这样工作的，现在类型也匹配了
          <Form.Item name="category_id" label="分类" rules={[{ required: true, message: '请选择分类' }]}>
            <Select placeholder="选择分类">
              {filteredCategories.map(cat => (
                <Select.Option key={cat.id} value={cat.id}>{cat.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
        )}

        <Form.Item name="description" label="备注">
          <Input.TextArea rows={2} placeholder="选填，最多100字" maxLength={100} />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default AddTransactionModal;