package servutil_test

import (
	"metrics-service/internal/servutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepareUrl(t *testing.T) {
	type args struct {
		URL string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{name: "positive test #1", args: args{"/update/counter/someMetric/527"}, want: []string{"counter", "someMetric", "527"}},
		{name: "positive test #2", args: args{"/update/gauge/Sys/100.1"}, want: []string{"gauge", "Sys", "100.1"}},
		{name: "positive test #3", args: args{"/counter/someMetric/527"}, want: []string{"counter", "someMetric", "527"}},
		{name: "positive test #4", args: args{"////counter/someMetric/527"}, want: []string{"counter", "someMetric", "527"}},
		{name: "positive test #5", args: args{"///update/counter/someMetric/527"}, want: []string{"counter", "someMetric", "527"}},	
		{name: "positive test #6", args: args{"update/counter/someMetric/527"}, want: []string{"counter", "someMetric", "527"}},
		{name: "positive test #7", args: args{"/update/counter/someMetric/527/foo"}, want: []string{"counter", "someMetric", "527", "foo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := servutil.PrepareURL(tt.args.URL); !assert.Equal(t, got, tt.want) {
				t.Errorf("PrepareUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

