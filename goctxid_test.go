package goctxid

import (
	"context"
	"testing"
)

func TestFromContext(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func() context.Context
		expectedID     string
		expectedExists bool
	}{
		{
			name: "returns correlation ID when present",
			setupContext: func() context.Context {
				return NewContext(context.Background(), "test-id-123")
			},
			expectedID:     "test-id-123",
			expectedExists: true,
		},
		{
			name: "returns empty string and false when not present",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedID:     "",
			expectedExists: false,
		},
		{
			name: "returns empty string and false for nil context value",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), ctxKey, nil)
			},
			expectedID:     "",
			expectedExists: false,
		},
		{
			name: "returns empty string and false for wrong type in context",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), ctxKey, 12345)
			},
			expectedID:     "",
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			id, exists := FromContext(ctx)

			if id != tt.expectedID {
				t.Errorf("FromContext() id = %v, want %v", id, tt.expectedID)
			}
			if exists != tt.expectedExists {
				t.Errorf("FromContext() exists = %v, want %v", exists, tt.expectedExists)
			}
		})
	}
}

func TestNewContext(t *testing.T) {
	tests := []struct {
		name       		string
		baseCtx    		context.Context
		id         		string
		expectedID     	string
		expectedExists 	bool
	}{
		{
			name:       	"creates context with correlation ID",
			baseCtx:    	context.Background(),
			id:         	"new-correlation-id",
			expectedID:     "new-correlation-id",
			expectedExists: true,
		},
		{
			name:       	"creates context with empty string ID",
			baseCtx:    	context.Background(),
			id:         	"",
			expectedID:     "",
			expectedExists: true,
		},
		{
			name:       	"overwrites existing correlation ID",
			baseCtx:    	NewContext(context.Background(), "old-id"),
			id:         	"new-id",
			expectedID:     "new-id",
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(tt.baseCtx, tt.id)
			id, exists := FromContext(ctx)

			if id != tt.expectedID {
				t.Errorf("NewContext() stored id = %v, want %v", id, tt.expectedID)
			}
			if exists != tt.expectedExists {
				t.Errorf("NewContext() exists = %v, want %v", exists, tt.expectedExists)
			}
		})
	}
}

func TestMustFromContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "returns correlation ID when present",
			ctx:      NewContext(context.Background(), "test-id-456"),
			expected: "test-id-456",
		},
		{
			name:     "returns empty string when not present",
			ctx:      context.Background(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := MustFromContext(tt.ctx)
			if id != tt.expected {
				t.Errorf("MustFromContext() = %v, want %v", id, tt.expected)
			}
		})
	}
}

func TestDefaultGenerator(t *testing.T) {
	// Test that default generator produces non-empty UUIDs
	id1 := defaultGenerator()
	if id1 == "" {
		t.Error("defaultGenerator() returned empty string")
	}

	// Test that it generates unique IDs
	id2 := defaultGenerator()
	if id1 == id2 {
		t.Error("defaultGenerator() produced duplicate IDs")
	}

	// Test UUID format (basic check for length and hyphens)
	if len(id1) != 36 {
		t.Errorf("defaultGenerator() produced ID with wrong length: got %d, want 36", len(id1))
	}
}

func TestContextKeyIsolation(t *testing.T) {
	// Ensure our context key doesn't conflict with other string keys
	ctx := context.Background()
	
	// Add a value with a regular string key
	ctx = context.WithValue(ctx, "goctxid_key", "wrong-value")
	
	// Add our correlation ID
	ctx = NewContext(ctx, "correct-value")
	
	// Our typed key should retrieve the correct value
	id, exists := FromContext(ctx)
	if !exists {
		t.Error("FromContext() should find the correlation ID")
	}
	if id != "correct-value" {
		t.Errorf("FromContext() = %v, want %v (context key collision detected)", id, "correct-value")
	}
	
	// The string key should still have its value
	if val := ctx.Value("goctxid_key"); val != "wrong-value" {
		t.Error("String key value was overwritten (context key collision)")
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Test that context operations are safe for concurrent use
	ctx := NewContext(context.Background(), "concurrent-test-id")
	
	done := make(chan bool)
	
	// Spawn multiple goroutines reading from context
	for i := 0; i < 100; i++ {
		go func() {
			id, exists := FromContext(ctx)
			if !exists || id != "concurrent-test-id" {
				t.Errorf("Concurrent FromContext() failed: got (%v, %v)", id, exists)
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

func BenchmarkNewContext(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewContext(ctx, "benchmark-id")
	}
}

func BenchmarkFromContext(b *testing.B) {
	ctx := NewContext(context.Background(), "benchmark-id")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromContext(ctx)
	}
}

func BenchmarkDefaultGenerator(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		defaultGenerator()
	}
}

