import { post } from './client';

export const createXrayCert = (data: any) => post('/v/cert/create', data);
export const getXrayCertList = (nodeId?: number) => post('/v/cert/list', nodeId ? { nodeId } : {});
export const deleteXrayCert = (id: number) => post('/v/cert/delete', { id });
export const issueXrayCert = (id: number) => post('/v/cert/issue', { id });
export const renewXrayCert = (id: number) => post('/v/cert/renew', { id });
