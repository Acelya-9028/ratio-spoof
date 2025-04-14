package input

import (
	"errors"
	"testing"
)

func CheckError(out error, want error, t *testing.T) {
	t.Helper()
	if out == nil && want == nil {
		return
	}
	if out != nil && want == nil {
		t.Errorf("got %v, want %v", out.Error(), "")
	}
	if out == nil && want != nil {
		t.Errorf("got %v, want %v", "", want.Error())
	}
	if out != nil && want != nil && out.Error() != want.Error() {
		t.Errorf("got %v, want %v", out.Error(), want.Error())
	}
}

func TestExtractInputInitialByteCount(T *testing.T) {
	data := []struct {
		name            string
		inSize          string
		inTotal         int
		inErrorIfHigher bool
		err             error
	}{
		{
			name:            "[Downloaded - error if higher] 50% input with 200kb limit shouldn't return error test",
			inSize:          "50%",
			inTotal:         204800,
			inErrorIfHigher: true,
		},
		{
			name:            "[Downloaded - error if higher] 150% input with 200kb limit should return error test",
			inSize:          "150%",
			inTotal:         204800,
			inErrorIfHigher: true,
			err:             errors.New("percentage must be between 0 and 100"),
		},
		{
			name:            "[Uploaded] 50% input with 200kb limit shouldn't return error test",
			inSize:          "50%",
			inTotal:         204800,
			inErrorIfHigher: false,
		},
		{
			name:            "[Uploaded] 150% input with 200kb limit shouldn't return error test",
			inSize:          "150%",
			inTotal:         204800,
			inErrorIfHigher: false,
			err:             errors.New("percentage must be between 0 and 100"),
		},
		{
			name:            "[Downloaded] -50% should return negative number error test",
			inSize:          "-50%",
			inTotal:         204800,
			inErrorIfHigher: true,
			err:             errors.New("percentage must be between 0 and 100"),
		},
		{
			name:            "[Uploaded] -50% should return negative number error test",
			inSize:          "-50%",
			inTotal:         204800,
			inErrorIfHigher: false,
			err:             errors.New("percentage must be between 0 and 100"),
		},
		{
			name:            "Invalid format should return error test",
			inSize:          "50kb",
			inTotal:         204800,
			inErrorIfHigher: true,
			err:             errors.New("initial value must be in percentage"),
		},
		{
			name:            "Invalid percentage value should return error test",
			inSize:          "abc%",
			inTotal:         204800,
			inErrorIfHigher: true,
			err:             errors.New("invalid percentage value"),
		},
	}

	for _, td := range data {
		T.Run(td.name, func(t *testing.T) {
			_, err := extractInputInitialByteCount(td.inSize, td.inTotal, td.inErrorIfHigher)
			CheckError(err, td.err, t)
		})
	}
}

func TestExtractInputByteSpeed(T *testing.T) {
	data := []struct {
		name     string
		speed    string
		expected int
		err      error
	}{
		{
			name:     "1kbps test",
			speed:    "1kbps",
			expected: 1024,
		},
		{
			name:     "1024kbps test",
			speed:    "1024kbps",
			expected: 1048576,
		},
		{
			name:     "1mbps test",
			speed:    "1mbps",
			expected: 1048576,
		},
		{
			name:  "2.5tbps test",
			speed: "2.5tbps",
			err:   errors.New("speed must be in [kbps mbps]"),
		},
		{
			name:  "-akbps test",
			speed: "-akbps",
			err:   errors.New("invalid speed number"),
		},
		{
			name:  "-10kbps test",
			speed: "-10kbps",
			err:   errors.New("speed can not be negative"),
		},
	}

	for _, td := range data {
		T.Run(td.name, func(t *testing.T) {
			got, err := extractInputByteSpeed(td.speed)
			if td.err != nil {
				if td.err.Error() != err.Error() {
					t.Errorf("got %v, want %v", err.Error(), td.err.Error())
				}
			}

			if got != td.expected {
				t.Errorf("got %v, want %v", got, td.expected)
			}
		})
	}
}
