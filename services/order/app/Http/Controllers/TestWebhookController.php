<?php

namespace App\Http\Controllers;

use Illuminate\Http\JsonResponse as JsonResponse;
use Illuminate\Http\Request;
use Stripe\Event;
use Stripe\Exception\SignatureVerificationException;
use Stripe\Webhook;

class TestWebhookController extends Controller
{
    public function handle(Request $request): JsonResponse {

        $payload = $request->getContent();
        $sigHeader = $request->header('Stripe-Signature');
        $endpointSecret = config('services.stripe.webhook_secret'); 

        try {
            $event = Webhook::constructEvent($payload, $sigHeader, $endpointSecret);
        } catch (SignatureVerificationException $e) {
            return response()->json(['error' => 'Invalid signature'], 400);
        } catch (\UnexpectedValueException $e) {
            return response()->json(['error' => 'Invalid payload'], 400);
        }

        return $this->processEvent($event);

    }

    private function processEvent(Event $event): JsonResponse
    {
        switch ($event->type) {
            case 'payment_intent.succeeded':
                //
                break;

            case 'customer.subscription.deleted':
                //
                break;
        }

        // Обязательно возвращаем Stripe 200 OK
        return response()->json(['status' => 'success'], 200);
    }
}
