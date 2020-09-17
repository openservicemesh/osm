package injector

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

func (wh *webhook) isMetricsEnabled(namespace string) (bool, error) {
	ns, err := wh.kubeClient.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving namespace %s", namespace)
		return false, err
	}

	enabled, err := isAnnotatedForMetrics(ns.Annotations)
	if err != nil {
		return false, err
	}

	return enabled, nil
}

func isAnnotatedForMetrics(annotations map[string]string) (enabled bool, err error) {
	metrics, ok := annotations[constants.MetricsAnnotation]
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
			err = errors.Errorf("Invalid annotion value specified for annotation %q: %s", constants.MetricsAnnotation, metrics)
		}
	}
	return
}
