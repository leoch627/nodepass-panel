'use client';

import { useState, useEffect, useCallback } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { User, Database, Clock, ArrowRightLeft, KeyRound } from 'lucide-react';
import { toast } from 'sonner';
import { getUserPackageInfo } from '@/lib/api/user';
import { useAuth } from '@/lib/hooks/use-auth';
import { useTranslation } from '@/lib/i18n';

export default function ProfilePage() {
  const { username, isAdmin } = useAuth();
  const { t } = useTranslation();
  const [packageInfo, setPackageInfo] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const loadData = useCallback(async () => {
    setLoading(true);
    const res = await getUserPackageInfo();
    if (res.code === 0) {
      setPackageInfo(res.data);
    } else {
      toast.error(res.msg || t('profile.loadFailed'));
    }
    setLoading(false);
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-muted-foreground">{t('common.loading')}</p>
      </div>
    );
  }

  const gostUsed = packageInfo ? (packageInfo.inFlow || 0) + (packageInfo.outFlow || 0) : 0;
  const gostTotal = packageInfo?.flow ? packageInfo.flow * 1024 * 1024 * 1024 : 0;
  const gostPercent = gostTotal > 0 ? Math.min((gostUsed / gostTotal) * 100, 100) : 0;
  const xrayUsed = packageInfo ? (packageInfo.vInFlow || 0) + (packageInfo.vOutFlow || 0) : 0;
  const xrayTotal = packageInfo?.vFlow ? packageInfo.vFlow * 1024 * 1024 * 1024 : 0;
  const xrayPercent = xrayTotal > 0 ? Math.min((xrayUsed / xrayTotal) * 100, 100) : 0;
  const isExpired = packageInfo?.expTime && new Date(packageInfo.expTime) < new Date();
  const isGostOver = gostTotal > 0 && gostUsed >= gostTotal;
  const isXrayOver = xrayTotal > 0 && xrayUsed >= xrayTotal;
  const isOverFlow = isGostOver || isXrayOver;

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">{t('profile.title')}</h2>

      <div className="grid gap-4 md:grid-cols-2">
        {/* User Info Card */}
        <Card>
          <CardHeader className="flex flex-row items-center gap-3 pb-2">
            <User className="h-5 w-5 text-primary" />
            <CardTitle className="text-lg">{t('profile.userInfo')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">{t('profile.username')}</span>
              <span className="font-medium">{username}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">{t('profile.role')}</span>
              <Badge variant={isAdmin ? 'default' : 'secondary'}>
                {isAdmin ? t('profile.admin') : t('profile.normalUser')}
              </Badge>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">{t('profile.accountStatus')}</span>
              {isExpired ? (
                <Badge variant="destructive">{t('profile.expired')}</Badge>
              ) : isOverFlow ? (
                <Badge variant="destructive">{t('profile.overTraffic')}</Badge>
              ) : (
                <Badge variant="default">{t('profile.normal')}</Badge>
              )}
            </div>
          </CardContent>
        </Card>

        {/* GOST Flow Usage Card */}
        <Card>
          <CardHeader className="flex flex-row items-center gap-3 pb-2">
            <Database className="h-5 w-5 text-primary" />
            <CardTitle className="text-lg">{t('profile.gostTrafficUsage')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">{t('profile.usedTraffic')}</span>
              <span className="font-medium">{formatBytes(gostUsed)}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">{t('profile.totalTraffic')}</span>
              <span className="font-medium">{packageInfo?.flow ? `${packageInfo.flow} GB` : t('common.unlimited')}</span>
            </div>
            {gostTotal > 0 && (
              <div className="space-y-1">
                <div className="flex justify-between text-xs text-muted-foreground">
                  <span>{t('profile.usageProgress')}</span>
                  <span>{gostPercent.toFixed(1)}%</span>
                </div>
                <div className="w-full bg-muted rounded-full h-2">
                  <div
                    className={`h-2 rounded-full transition-all ${gostPercent > 90 ? 'bg-destructive' : gostPercent > 70 ? 'bg-yellow-500' : 'bg-primary'}`}
                    style={{ width: `${gostPercent}%` }}
                  />
                </div>
              </div>
            )}
            <div className="flex justify-between items-center text-sm">
              <span className="text-muted-foreground">{t('profile.upload')}</span>
              <span>{formatBytes(packageInfo?.inFlow || 0)}</span>
            </div>
            <div className="flex justify-between items-center text-sm">
              <span className="text-muted-foreground">{t('profile.download')}</span>
              <span>{formatBytes(packageInfo?.outFlow || 0)}</span>
            </div>
          </CardContent>
        </Card>

        {/* Xray Flow Usage Card */}
        <Card>
          <CardHeader className="flex flex-row items-center gap-3 pb-2">
            <Database className="h-5 w-5 text-primary" />
            <CardTitle className="text-lg">{t('profile.xrayTrafficUsage')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">{t('profile.usedTraffic')}</span>
              <span className="font-medium">{formatBytes(xrayUsed)}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">{t('profile.totalTraffic')}</span>
              <span className="font-medium">{packageInfo?.vFlow ? `${packageInfo.vFlow} GB` : t('common.unlimited')}</span>
            </div>
            {xrayTotal > 0 && (
              <div className="space-y-1">
                <div className="flex justify-between text-xs text-muted-foreground">
                  <span>{t('profile.usageProgress')}</span>
                  <span>{xrayPercent.toFixed(1)}%</span>
                </div>
                <div className="w-full bg-muted rounded-full h-2">
                  <div
                    className={`h-2 rounded-full transition-all ${xrayPercent > 90 ? 'bg-destructive' : xrayPercent > 70 ? 'bg-yellow-500' : 'bg-primary'}`}
                    style={{ width: `${xrayPercent}%` }}
                  />
                </div>
              </div>
            )}
            <div className="flex justify-between items-center text-sm">
              <span className="text-muted-foreground">{t('profile.upload')}</span>
              <span>{formatBytes(packageInfo?.vInFlow || 0)}</span>
            </div>
            <div className="flex justify-between items-center text-sm">
              <span className="text-muted-foreground">{t('profile.download')}</span>
              <span>{formatBytes(packageInfo?.vOutFlow || 0)}</span>
            </div>
          </CardContent>
        </Card>

        {/* Package Info Card */}
        <Card>
          <CardHeader className="flex flex-row items-center gap-3 pb-2">
            <ArrowRightLeft className="h-5 w-5 text-primary" />
            <CardTitle className="text-lg">{t('profile.packageInfo')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">{t('profile.forwardLimit')}</span>
              <span className="font-medium">{packageInfo?.num || t('common.unlimited')}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">{t('profile.expireTime')}</span>
              <span className="font-medium">
                {packageInfo?.expTime ? new Date(packageInfo.expTime).toLocaleString() : t('common.neverExpire')}
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Quick Actions Card */}
        <Card>
          <CardHeader className="flex flex-row items-center gap-3 pb-2">
            <Clock className="h-5 w-5 text-primary" />
            <CardTitle className="text-lg">{t('profile.quickActions')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <Button variant="outline" className="w-full justify-start" asChild>
              <a href="/change-password">
                <KeyRound className="mr-2 h-4 w-4" />
                {t('profile.changePassword')}
              </a>
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
