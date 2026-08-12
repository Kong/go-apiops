package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ValidateMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{name: "empty mode uses the library default", mode: "", wantErr: false},
		{name: "passthrough-listener is valid", mode: "passthrough-listener", wantErr: false},
		{name: "conversion-listener is valid", mode: "conversion-listener", wantErr: false},
		{name: "conversion-only is valid", mode: "conversion-only", wantErr: false},
		{name: "listener is valid", mode: "listener", wantErr: false},
		{name: "conversion is accepted as a deprecated alias", mode: "conversion", wantErr: false},
		{name: "unknown mode is rejected", mode: "bogus", wantErr: true},
		{name: "mode is case-sensitive", mode: "Conversion-Listener", wantErr: true},
		{name: "typo'd mode is rejected", mode: "conversion-listner", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMode(tt.mode)
			if tt.wantErr {
				assert.Error(t, err, "mode %q should be rejected", tt.mode)
				assert.Contains(t, err.Error(), "invalid mode")
			} else {
				assert.NoError(t, err, "mode %q should be accepted", tt.mode)
			}
		})
	}
}

func Test_ValidateMode_ErrorMessageOmitsDeprecatedAlias(t *testing.T) {
	err := validateMode("bogus")
	if err == nil {
		t.Fatal("expected an error for an invalid mode")
	}
	assert.NotContains(t, err.Error(), "'conversion'",
		"the deprecated alias should not be advertised as a valid choice in the error message")
}
