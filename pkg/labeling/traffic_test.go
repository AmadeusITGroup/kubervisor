package labeling

import (
	"testing"

	kv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetTraficLabel(t *testing.T) {
	type args struct {
		pod *kv1.Pod
		val LabelTraffic
	}
	tests := []struct {
		name      string
		args      args
		checkFunc func(pod *kv1.Pod) bool
	}{
		{
			name: "from blank",
			args: args{
				val: LabelTrafficYes,
				pod: &kv1.Pod{},
			},
			checkFunc: func(p *kv1.Pod) bool {
				if v, ok := p.Labels[LabelTrafficKey]; v != string(LabelTrafficYes) || !ok {
					return false
				}
				return true
			},
		},
		{
			name: "from No",
			args: args{
				val: LabelTrafficYes,
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{LabelBreakerNameKey: string(LabelTrafficNo)},
					},
				},
			},
			checkFunc: func(p *kv1.Pod) bool {
				if v, ok := p.Labels[LabelTrafficKey]; v != string(LabelTrafficYes) || !ok {
					return false
				}
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetTraficLabel(tt.args.pod, tt.args.val)
		})
	}
}

func TestIsPodTrafficLabelOkOrPause(t *testing.T) {
	type args struct {
		pod *kv1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   bool
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				pod: &kv1.Pod{},
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name: "from No",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{LabelTrafficKey: string(LabelTrafficNo)},
					},
				},
			},
			want:    false,
			want1:   false,
			wantErr: false,
		},
		{
			name: "from Yes",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{LabelTrafficKey: string(LabelTrafficYes)},
					},
				},
			},
			want:    true,
			want1:   false,
			wantErr: false,
		},
		{
			name: "from Pause",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{LabelTrafficKey: string(LabelTrafficPause)},
					},
				},
			},
			want:    false,
			want1:   true,
			wantErr: false,
		},
		{
			name: "from garbage",
			args: args{
				pod: &kv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{LabelTrafficKey: "gArbAgE"},
					},
				},
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := IsPodTrafficLabelOkOrPause(tt.args.pod)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsPodTrafficLabelOkOrPause() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsPodTrafficLabelOkOrPause() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("IsPodTrafficLabelOkOrPause() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
