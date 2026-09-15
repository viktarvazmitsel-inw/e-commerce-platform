<?php

namespace Tests\Feature;

use Tests\TestCase;

class HealthTest extends TestCase
{
    public function test_health_check_endpoint_returns_success_and_valid_structure(): void
    {
        $response = $this->getJson('/api/health');

        $response->assertStatus(200)
            ->assertJsonStructure([
                'status',
                'service',
                'database',
                'cache',
                'timestamp',
            ])
            ->assertJson([
                'status' => 'ok',
                'service' => 'order-service',
                'database' => 'connected',
                'cache' => 'connected',
            ]);
    }
}
