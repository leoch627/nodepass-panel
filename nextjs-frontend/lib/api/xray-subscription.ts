import { post } from './client';

export const getSubscriptionToken = () => post('/v/sub/token');
export const getSubscriptionLinks = () => post('/v/sub/links');
export const resetSubscriptionToken = () => post('/v/sub/reset');
