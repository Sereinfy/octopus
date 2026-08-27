package relay

import (
	"testing"

	"github.com/looplj/axonhub/llm"
)

func TestReplaceDeveloperRoles(t *testing.T) {
	request := &llm.Request{Messages: []llm.Message{
		{Role: "system"},
		{Role: "developer"},
		{Role: "user"},
	}}

	result := replaceDeveloperRoles(request)
	if result == request {
		t.Fatal("expected a copied request")
	}
	if request.Messages[1].Role != "developer" {
		t.Fatalf("original request was mutated: %q", request.Messages[1].Role)
	}
	if result.Messages[0].Role != "system" || result.Messages[1].Role != "system" || result.Messages[2].Role != "user" {
		t.Fatalf("unexpected roles after replacement: %+v", result.Messages)
	}
}

func TestReplaceDeveloperRolesKeepsRequestWithoutDeveloperMessages(t *testing.T) {
	request := &llm.Request{Messages: []llm.Message{{Role: "user"}}}
	result := replaceDeveloperRoles(request)
	if result != request {
		t.Fatal("request without developer messages should be reused")
	}
	if result.Messages[0].Role != "user" {
		t.Fatalf("unexpected role: %q", result.Messages[0].Role)
	}
}
