import { post, get } from './client';

export const startXray = (nodeId: number) => post('/v/node/start', { nodeId });
export const stopXray = (nodeId: number) => post('/v/node/stop', { nodeId });
export const restartXray = (nodeId: number) => post('/v/node/restart', { nodeId });
export const getXrayStatus = (nodeId: number) => post('/v/node/status', { nodeId });
export const switchXrayVersion = (nodeId: number, version: string) => post('/v/node/switch-version', { nodeId, version });
export const getXrayVersions = () => get('/v/node/versions');
