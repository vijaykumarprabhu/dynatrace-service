package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseDynatraceConfigYAML(t *testing.T) {
	tests := []struct {
		name       string
		yamlString string
		want       *DynatraceConfig
		wantErr    bool
	}{
		{
			name:       "empty string",
			yamlString: "",
			want:       NewDynatraceConfigWithDefaults(),
			wantErr:    false,
		},
		{
			name: "valid yaml no dashboard",
			yamlString: `
spec_version: '0.1.0'
dtCreds: dyna`,
			want: &DynatraceConfig{
				SpecVersion: "0.1.0",
				DtCreds:     "dyna",
			},
			wantErr: false,
		},
		{
			name: "valid yaml with dashboard",
			yamlString: `
spec_version: '0.1.0'
dtCreds: dyna
dashboard: dash`,
			want: &DynatraceConfig{
				SpecVersion: "0.1.0",
				DtCreds:     "dyna",
				Dashboard:   "dash",
			},
			wantErr: false,
		},
		{
			name: "invalid yaml",
			yamlString: `
spec_version: '0.1.0'
dtCreds: dyna
dashboard: ****`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "yaml with special characters",
			yamlString: `
spec_version: '0.1.0'
dtCreds: dyna
dashboard: '****'`,
			want: &DynatraceConfig{
				SpecVersion: "0.1.0",
				DtCreds:     "dyna",
				Dashboard:   "****",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDynatraceConfigYAML(tt.yamlString)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.EqualValues(t, tt.want, got)
		})
	}
}
