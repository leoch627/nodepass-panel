import { post } from './client';

export const getNodeHealth = () => post('/monitor/node-health', {});
export const getLatencyHistory = (forwardId: number, hours: number) =>
  post('/monitor/latency-history', { forwardId, hours });
export const getForwardFlowHistory = (forwardId: number, hours: number) =>
  post('/monitor/forward-flow', { forwardId, hours });
export const getTrafficOverview = (granularity: string, hours: number) =>
  post('/monitor/traffic-overview', { granularity, hours });
export const getXrayTrafficOverview = (granularity: string, hours: number) =>
  post('/monitor/v-traffic-overview', { granularity, hours });
export const getXrayInboundFlowHistory = (inboundId: number, hours: number) =>
  post('/monitor/v-inbound-flow', { inboundId, hours });
