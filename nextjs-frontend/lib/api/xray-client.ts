import { post } from './client';

export const createXrayClient = (data: any) => post('/v/client/create', data);
export const getXrayClientList = (params?: { inboundId?: number; userId?: number }) => post('/v/client/list', params || {});
export const updateXrayClient = (data: any) => post('/v/client/update', data);
export const deleteXrayClient = (id: number) => post('/v/client/delete', { id });
export const resetXrayClientTraffic = (id: number) => post('/v/client/reset-traffic', { id });
export const getXrayClientLink = (id: number) => post('/v/client/link', { id });
