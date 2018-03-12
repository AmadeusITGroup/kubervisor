package item

import (
	"context"
	"sync"
	"testing"

	activator "github.com/amadeusitgroup/podkubervisor/pkg/activate"
	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/breaker"
	"k8s.io/apimachinery/pkg/labels"
)

func TestKubervisorServiceItem_CompareWithSpec(t *testing.T) {
	type fields struct {
		name       string
		activator  activator.Activator
		breaker    breaker.Breaker
		cancelFunc context.CancelFunc
		waitGroup  sync.WaitGroup
	}
	type args struct {
		spec     *v1.KubervisorServiceSpec
		selector labels.Selector
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
	/*
		{
			name:   "similar",
			fields: fields{},
			args:   args{},
			want:   true,
		},
	*/
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &KubervisorServiceItem{
				name:       tt.fields.name,
				activator:  tt.fields.activator,
				breaker:    tt.fields.breaker,
				cancelFunc: tt.fields.cancelFunc,
				waitGroup:  tt.fields.waitGroup,
			}
			if got := b.CompareWithSpec(tt.args.spec, tt.args.selector); got != tt.want {
				t.Errorf("KubervisorServiceItem.CompareWithSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}
