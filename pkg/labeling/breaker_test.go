package labeling

import (
	"reflect"
	"testing"
	"time"

	kv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestGetRetryAt(t *testing.T) {

	t0, _ := time.Parse(time.RFC3339, "1978-12-04T22:11:00+00:00")
	type args struct {
		pod *kv1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
			want:    time.Time{},
			wantErr: true,
		},
		{
			name: "nil",
			args: args{
				pod: &kv1.Pod{},
			},
			want:    time.Time{},
			wantErr: true,
		},
		{
			name: "timeOk",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{AnnotationBreakAtKey: string("1978-12-04T22:11:00+00:00")},
					},
				},
			},
			want:    t0,
			wantErr: false,
		},
		{
			name: "Albert Einstein",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{AnnotationBreakAtKey: string("Time is what the clock says.")},
					},
				},
			},
			want:    time.Time{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBreakAt(tt.args.pod)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRetryAt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRetryAt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRetryCount(t *testing.T) {
	type args struct {
		pod *kv1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "nil",
			args: args{
				pod: &kv1.Pod{},
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "ok",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{AnnotationRetryCountKey: string("1204")},
					},
				},
			},
			want:    1204,
			wantErr: false,
		},
		{
			name: "roman",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{AnnotationRetryCountKey: string("MCCIV")},
					},
				},
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRetryCount(tt.args.pod)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRetryCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetRetryCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectorWithBreakerName(t *testing.T) {
	type args struct {
		inputSelector labels.Selector
		breakerName   string
	}
	tests := []struct {
		name    string
		args    args
		want    labels.Selector
		wantErr bool
	}{
		{
			name: "fine",
			args: args{
				inputSelector: labels.Set{"app": "test1"}.AsSelectorPreValidated(),
				breakerName:   "b1",
			},
			wantErr: false,
			want:    labels.Set{"app": "test1", LabelBreakerNameKey: "b1"}.AsSelectorPreValidated(),
		},
		{
			name: "fine nil",
			args: args{
				breakerName: "b1",
			},
			wantErr: false,
			want:    labels.Set{LabelBreakerNameKey: "b1"}.AsSelectorPreValidated(),
		},
		{
			name: "fine everything",
			args: args{
				inputSelector: labels.Everything(),
				breakerName:   "b1",
			},
			wantErr: false,
			want:    labels.Set{LabelBreakerNameKey: "b1"}.AsSelectorPreValidated(),
		},
		{
			name: "Error fornat",
			args: args{
				inputSelector: labels.Everything(),
				breakerName:   "b1!@#!!",
			},
			wantErr: true,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectorWithBreakerName(tt.args.inputSelector, tt.args.breakerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectorWithBreakerName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SelectorWithBreakerName() = %v, want %v", got, tt.want)
			}
		})
	}
}
