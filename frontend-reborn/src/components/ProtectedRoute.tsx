// src/components/ProtectedRoute.tsx
import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useIsAuthenticated } from '../stores/authStore';

interface ProtectedRouteProps {
  children: React.ReactElement;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
  const isAuthenticated = useIsAuthenticated();
  const location = useLocation();

  if (!isAuthenticated) {
    // 如果用户未认证，重定向到登录页
    // 保存用户尝试访问的页面路径，以便登录后可以重定向回来
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // 如果已认证，则渲染子组件
  return children;
};

export default ProtectedRoute;