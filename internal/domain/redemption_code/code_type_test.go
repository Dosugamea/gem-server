package redemption_code

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCodeType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CodeType
		wantErr bool
	}{
		{
			name:    "正常系: promotion",
			input:   "promotion",
			want:    CodeTypePromotion,
			wantErr: false,
		},
		{
			name:    "正常系: gift",
			input:   "gift",
			want:    CodeTypeGift,
			wantErr: false,
		},
		{
			name:    "正常系: event",
			input:   "event",
			want:    CodeTypeEvent,
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
			got, err := NewCodeType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCodeType_String(t *testing.T) {
	tests := []struct {
		name string
		ct   CodeType
		want string
	}{
		{
			name: "正常系: promotion",
			ct:   CodeTypePromotion,
			want: "promotion",
		},
		{
			name: "正常系: gift",
			ct:   CodeTypeGift,
			want: "gift",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ct.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCodeType_Valid(t *testing.T) {
	tests := []struct {
		name string
		ct   CodeType
		want bool
	}{
		{
			name: "正常系: promotion",
			ct:   CodeTypePromotion,
			want: true,
		},
		{
			name: "正常系: gift",
			ct:   CodeTypeGift,
			want: true,
		},
		{
			name: "正常系: event",
			ct:   CodeTypeEvent,
			want: true,
		},
		{
			name: "異常系: 無効な値",
			ct:   CodeType("invalid"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ct.Valid()
			assert.Equal(t, tt.want, got)
		})
	}
}
