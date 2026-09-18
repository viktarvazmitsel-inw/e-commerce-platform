<?php

use App\Http\Controllers\HealthController as HealthController;
use App\Http\Controllers\TestWebhookController;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;

Route::get('/order/health', [HealthController::class, 'check']);

Route::get('/order/test', function (Request $request) {
    return response()->json([
        'status' => 'success',
        'service' => 'order-service',
        'message' => 'Order service API is running'
    ]);
});

Route::post('/order/stripe/webhook', [TestWebhookController::class, 'handle']);
