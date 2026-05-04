package domain_test

import (
	"testing"
	"time"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateOnly_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    time.Time
		wantErr bool
	}{
		{
			name:    "valid format",
			input:   []byte(`"2023-10-25"`),
			want:    time.Date(2023, 10, 25, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "valid format no quotes",
			input:   []byte(`2023-10-25`),
			want:    time.Date(2023, 10, 25, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   []byte(`""`),
			want:    time.Time{},
			wantErr: false,
		},
		{
			name:    "null",
			input:   []byte(`null`),
			want:    time.Time{},
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   []byte(`"25-10-2023"`),
			want:    time.Time{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d domain.DateOnly
			err := d.UnmarshalJSON(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, d.Time)
			}
		})
	}
}

func TestDateOnly_MarshalJSON(t *testing.T) {
	t.Run("zero time", func(t *testing.T) {
		d := domain.DateOnly{Time: time.Time{}}
		b, err := d.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, []byte("null"), b)
	})

	t.Run("valid time", func(t *testing.T) {
		d := domain.DateOnly{Time: time.Date(2023, 10, 25, 0, 0, 0, 0, time.UTC)}
		b, err := d.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, []byte(`"2023-10-25"`), b)
	})
}

func TestDateOnly_Value(t *testing.T) {
	t.Run("zero time", func(t *testing.T) {
		d := domain.DateOnly{Time: time.Time{}}
		v, err := d.Value()
		require.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("valid time", func(t *testing.T) {
		expectedTime := time.Date(2023, 10, 25, 0, 0, 0, 0, time.UTC)
		d := domain.DateOnly{Time: expectedTime}
		v, err := d.Value()
		require.NoError(t, err)
		assert.Equal(t, expectedTime, v)
	})
}

func TestDateOnly_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    time.Time
		wantErr bool
	}{
		{
			name:    "nil",
			input:   nil,
			want:    time.Time{},
			wantErr: false,
		},
		{
			name:    "time.Time",
			input:   time.Date(2023, 10, 25, 15, 30, 0, 0, time.UTC),
			want:    time.Date(2023, 10, 25, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "string",
			input:   "2023-10-25",
			want:    time.Date(2023, 10, 25, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "[]byte",
			input:   []byte("2023-10-25"),
			want:    time.Date(2023, 10, 25, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "invalid string",
			input:   "invalid-date",
			wantErr: true,
		},
		{
			name:    "unsupported type",
			input:   12345,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d domain.DateOnly
			err := d.Scan(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, d.Time)
			}
		})
	}
}

func TestTimeOnly_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    time.Time
		wantErr bool
	}{
		{
			name:    "valid format HH:MM",
			input:   []byte(`"15:30"`),
			want:    time.Date(2000, 1, 1, 15, 30, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "valid format HH:MM:SS",
			input:   []byte(`"15:30:45"`),
			want:    time.Date(2000, 1, 1, 15, 30, 45, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   []byte(`""`),
			want:    time.Time{},
			wantErr: false,
		},
		{
			name:    "null",
			input:   []byte(`null`),
			want:    time.Time{},
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   []byte(`"15-30"`),
			want:    time.Time{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var to domain.TimeOnly
			err := to.UnmarshalJSON(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, to.Time)
			}
		})
	}
}

func TestTimeOnly_MarshalJSON(t *testing.T) {
	t.Run("zero time", func(t *testing.T) {
		to := domain.TimeOnly{Time: time.Time{}}
		b, err := to.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, []byte("null"), b)
	})

	t.Run("valid time", func(t *testing.T) {
		to := domain.TimeOnly{Time: time.Date(2000, 1, 1, 15, 30, 0, 0, time.UTC)}
		b, err := to.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, []byte(`"15:30"`), b)
	})
}

func TestTimeOnly_Value(t *testing.T) {
	t.Run("zero time", func(t *testing.T) {
		to := domain.TimeOnly{Time: time.Time{}}
		v, err := to.Value()
		require.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("valid time", func(t *testing.T) {
		expectedTime := time.Date(2000, 1, 1, 15, 30, 0, 0, time.UTC)
		to := domain.TimeOnly{Time: expectedTime}
		v, err := to.Value()
		require.NoError(t, err)
		assert.Equal(t, expectedTime, v)
	})
}

func TestTimeOnly_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    time.Time
		wantErr bool
	}{
		{
			name:    "nil",
			input:   nil,
			want:    time.Time{},
			wantErr: false,
		},
		{
			name:    "time.Time",
			input:   time.Date(2023, 10, 25, 15, 30, 45, 0, time.UTC),
			want:    time.Date(2000, 1, 1, 15, 30, 45, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "string HH:MM",
			input:   "15:30",
			want:    time.Date(2000, 1, 1, 15, 30, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "string HH:MM:SS",
			input:   "15:30:45",
			want:    time.Date(2000, 1, 1, 15, 30, 45, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "[]byte HH:MM",
			input:   []byte("15:30"),
			want:    time.Date(2000, 1, 1, 15, 30, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "[]byte HH:MM:SS",
			input:   []byte("15:30:45"),
			want:    time.Date(2000, 1, 1, 15, 30, 45, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "invalid string",
			input:   "invalid-time",
			wantErr: true,
		},
		{
			name:    "unsupported type",
			input:   12345,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var to domain.TimeOnly
			err := to.Scan(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, to.Time)
			}
		})
	}
}
