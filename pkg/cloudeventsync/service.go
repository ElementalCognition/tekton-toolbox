package cloudeventsync

import (
	"context"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
)

type Service interface {
	Sync(ctx context.Context, eventType string, cloudEvent *cloudevent.TektonCloudEventData) error
}
