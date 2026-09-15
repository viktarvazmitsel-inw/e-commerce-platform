<?php

use Illuminate\Support\Facades\Route;
use Illuminate\Support\Facades\Storage;

Route::get('/', function () {
    return view('welcome');
});

Route::get('/test-minio', function () {
    $filename = 'test-connection-' . time() . '.txt';
    $content = 'MinIO and Laravel are successfully connected!';

    try {
        Storage::disk('s3')->put($filename, $content);

        if (!Storage::disk('s3')->exists($filename)) {
            return response()->json(['status' => 'error', 'message' => 'File uploaded, but could not be found.'], 500);
        }

        $url = Storage::cloud()->url($filename);

        Storage::cloud()->delete($filename);

        return response()->json([
            'status' => 'success',
            'message' => 'Connection works perfectly!',
            'generated_url' => $url
        ]);

    } catch (\Throwable $e) {
        return response()->json([
            'status' => 'failed',
            'message' => $e->getMessage(),
            'previous' => $e->getPrevious() ? $e->getPrevious()->getMessage() : 'None'
        ], 500);
    }
});
