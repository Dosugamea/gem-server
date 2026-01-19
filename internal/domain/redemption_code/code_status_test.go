package redemption_code

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCodeStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CodeStatus
		wantErr bool
	}{
		{
			name:    "正常系: active",
			input:   "active",
			want:    CodeStatusActive,
			wantErr: false,
		},
		{
			name:    "正常系: expired",
			input:   "expired",
			want:    CodeStatusExpired,
			wantErr: false,
		},
		{
			name:    "正常系: disabled",
			input:   "disabled",
			want:    CodeStatusDisabled,
			wantErr: false,
		},
		{
			name:    "異常系: 無効な値",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCodeStatus(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCodeStatus_String(t *testing.T) {
	tests := []struct {
		name string
		cs   CodeStatus
		want string
	}{
		{
			name: "正常系: active",
			cs:   CodeStatusActive,
			want: "active",
		},
		{
			name: "正常系: expired",
			cs:   CodeStatusExpired,
			want: "expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cs.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCodeStatus_Valid(t *testing.T) {
	tests := []struct {
		name string
		cs   CodeStatus
		want bool
	}{
		{
			name: "正常系: active",
			cs:   CodeStatusActive,
			want: true,
		},
		{
			name: "正常系: expired",
			cs:   CodeStatusExpired,
			want: true,
		},
		{
			name: "正常系: disabled",
			cs:   CodeStatusDisabled,
			want: true,
		},
		{
			name: "異常系: 無効な値",
			cs:   CodeStatus("invalid"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cs.Valid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCodeStatus_IsActive(t *testing.T) {
	tests := []struct {
		name string
		cs   CodeStatus
		want bool
	}{
		{
			name: "正常系: active",
			cs:   CodeStatusActive,
			want: true,
		},
		{
			name: "正常系: expired",
			cs:   CodeStatusExpired,
			want: false,
		},
		{
			name: "正常系: disabled",
			cs:   CodeStatusDisabled,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cs.IsActive()
			assert.Equal(t, tt.want, got)
		})
	}
}
