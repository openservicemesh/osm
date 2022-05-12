package injector

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/constants"
)

func (wh *mutatingWebhook) isMetricsEnabled(namespace string) (enabled bool, err error) {
	ns := wh.kubeController.GetNamespace(namespace)
	if ns == nil {
		log.Error().Err(errNamespaceNotFound).Msgf("Error retrieving namespace %s", namespace)
		return false, errNamespaceNotFound
	}

	metrics, ok := ns.Annotations[constants.MetricsAnnotation]
	if !ok {
		return false, nil
	}

	log.Trace().Msgf("Metrics annotation: '%s:%s'", constants.MetricsAnnotation, metrics)
	metrics = strings.ToLower(metrics)
	if metrics != "" {
		switch metrics {
		case "enabled", "yes", "true":
			enabled = true
		case "disabled", "no", "false":
			enabled = false
		default:
			err = errors.Errorf("Invalid value specified for annotation %q: %s", constants.MetricsAnnotation, metrics)
		}
	}
	return
}
