// src/utils/colorUtils.ts

/**
 * 根据数值大小返回不同的阶梯颜色。
 * 规则：
 * - 负数：越小（绝对值越大），红色越深。
 * - 0: 灰色。
 * - 正数：越大，绿色/蓝色越深。
 * @param value 要计算颜色的数值
 * @returns CSS 颜色字符串
 */
export const getSteppedColor = (value: number): string => {
    if (value < 0) {
        if (value < -10000) return '#a8071a'; // 深红
        if (value < -1000) return '#cf1322';  // 红
        return '#f5222d';                     // 亮红
    }
    if (value === 0) {
        return '#8c8c8c'; // 灰色
    }
    // 正数
    if (value > 50000) return '#096dd9'; // 深蓝
    if (value > 10000) return '#1677ff'; // 主题蓝
    if (value > 1000) return '#237804';  // 深绿
    return '#52c41a';                     // 亮绿
};