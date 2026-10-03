package app

import (
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestVaultRecoveryReasonPreservesAccountServiceGuidance(t *testing.T) {
	const guidance = "synthetic key recovery instructions"
	failure := &rayleabot.ActionError{
		Code: "plugin.vault_locked", Message: guidance,
		Details: map[string]any{"reason": "native_key_unavailable"},
	}
	if got := friendlyError(failure); got != guidance {
		t.Fatalf("recovery guidance was replaced: %q", got)
	}
	failure.Details = nil
	if got := friendlyError(failure); got == guidance || got == "" {
		t.Fatalf("ordinary lock lost its standard guidance: %q", got)
	}
}
