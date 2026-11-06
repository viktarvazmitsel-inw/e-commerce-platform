package abroker

import (
	"context"
	"testing"
)

func TestValidateRouter(t *testing.T) {
	handler := func(context.Context, []byte, int) error { return nil }

	tests := []struct {
		name    string
		router  *BrokerRouter
		wantErr bool
	}{
		{
			name: "keys match declared services",
			router: &BrokerRouter{
				Services: []string{"notification", "order"},
				HandlerMap: map[string]MessageHandler{
					"notification":            handler,
					"notification.email.sent": handler,
					"order.created":           handler,
				},
			},
		},
		{
			name: "key matches no declared service",
			router: &BrokerRouter{
				Services: []string{"notification"},
				HandlerMap: map[string]MessageHandler{
					"user.created": handler,
				},
			},
			wantErr: true,
		},
		{
			name: "key shares a prefix without the dot boundary",
			router: &BrokerRouter{
				Services: []string{"notification"},
				HandlerMap: map[string]MessageHandler{
					"notificationextra.sent": handler,
				},
			},
			wantErr: true,
		},
		{
			name:    "nil router",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRouter(test.router)
			if test.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
