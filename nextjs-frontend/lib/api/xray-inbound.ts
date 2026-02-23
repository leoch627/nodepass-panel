import { post } from './client';

export const createXrayInbound = (data: any) => post('/v/inbound/create', data);
export const getXrayInboundList = (nodeId?: number) => post('/v/inbound/list', nodeId ? { nodeId } : {});
export const updateXrayInbound = (data: any) => post('/v/inbound/update', data);
export const deleteXrayInbound = (id: number) => post('/v/inbound/delete', { id });
export const enableXrayInbound = (id: number) => post('/v/inbound/enable', { id });
export const disableXrayInbound = (id: number) => post('/v/inbound/disable', { id });
export const genXrayKey = () => post<{ privateKey: string; publicKey: string }>('/v/inbound/genkey', {});
