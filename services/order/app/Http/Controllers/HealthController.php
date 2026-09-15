<?php

namespace App\Http\Controllers;

use Illuminate\Http\JsonResponse;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Redis;

class HealthController extends Controller
{
    public function check(): JsonResponse
    {
        $dbStatus = 'disconnected';
        $cacheStatus = 'disconnected';

        try {
            DB::connection()->getPdo();
            $dbStatus = 'connected';
        } catch (\Throwable $e) {
            $dbStatus = 'disconnected';
        }

        try {
            Redis::connection()->ping();
            $cacheStatus = 'connected';
        } catch (\Throwable $e) {
            $cacheStatus = 'disconnected';
        }

        $isHealthy = ($dbStatus === 'connected' && $cacheStatus === 'connected');

        return response()->json([
            'status' => $isHealthy ? 'ok' : 'degraded',
            'service' => 'order-service',
            'database' => $dbStatus,
            'cache' => $cacheStatus,
            'timestamp' => now()->toIso8601String(),
        ], $isHealthy ? 200 : 503);
    }
}
