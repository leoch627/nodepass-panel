'use client';

import { useEffect, useState } from 'react';


interface AuthState {
  isAuthenticated: boolean;
  isAdmin: boolean;
  username: string;
  gostEnabled: boolean;
  vEnabled: boolean;
  loading: boolean;
}

export function useAuth(): AuthState {
  const [state, setState] = useState<AuthState>({
    isAuthenticated: false,
    isAdmin: false,
    username: '',
    gostEnabled: true,
    vEnabled: true,
    loading: true,
  });

  useEffect(() => {
    const token = localStorage.getItem('token');
    const name = localStorage.getItem('name') || '';
    const roleId = parseInt(localStorage.getItem('role_id') || '1', 10);
    const isAdmin = roleId === 0;

    setState({
      isAuthenticated: !!token,
      isAdmin,
      username: name,
      gostEnabled: isAdmin || localStorage.getItem('gost_enabled') !== '0',
      vEnabled: isAdmin || localStorage.getItem('v_enabled') !== '0',
      loading: false,
    });
  }, []);

  return state;
}

export function useRequireAuth() {
  const auth = useAuth();

  useEffect(() => {
    if (!auth.loading && !auth.isAuthenticated) {
      window.location.href = '/';
    }
  }, [auth.loading, auth.isAuthenticated]);

  return auth;
}

export function logout() {
  localStorage.removeItem('token');
  localStorage.removeItem('role_id');
  localStorage.removeItem('name');
  localStorage.removeItem('admin');
  localStorage.removeItem('gost_enabled');
  localStorage.removeItem('v_enabled');
  window.location.href = '/';
}
