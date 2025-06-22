// src/components/IconPicker.tsx
import React from 'react';
// 导入所有 lucide-react 图标
import * as Icons from 'lucide-react';

// 将导入的模块转换为图标组件的映射
const iconComponentMap = Icons as unknown as { [key: string]: Icons.LucideIcon };

// 定义我们在应用中使用的有效图标的白名单
// 这与后端 seedCategories 函数中的图标列表保持一致
const validIconKeys = [
    'Landmark', 'TrendingUp', 'Briefcase', 'Home', 'Utensils', 'Car', 'ShoppingBag', 'Zap', 'Film', 'HeartPulse',
    'ReceiptText', 'Percent', 'Archive', 'Scale', 'TrendingDown', 'Wallet', 'PiggyBank'
];

// 构建最终的图标名称到组件的映射
export const iconMap: { [key:string]: Icons.LucideIcon } = {};
validIconKeys.forEach(key => {
    // 确保图标真的存在于 lucide-react 中
    if (iconComponentMap[key]) {
        iconMap[key] = iconComponentMap[key];
    }
});

// 导出所有可用图标的名称列表，用于下拉选择器
export const availableIcons = Object.keys(iconMap);

// 一个根据名称渲染图标的组件
const IconDisplay = ({ name, ...props }: { name: string, [key: string]: any }) => {
    const IconComponent = iconMap[name];
    // 如果提供的名称在我们的映射中找不到，就渲染一个默认的 Archive 图标，以避免程序崩溃
    return IconComponent ? <IconComponent {...props} /> : <Icons.Archive {...props} />;
};

export default IconDisplay;