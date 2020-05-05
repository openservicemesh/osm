package namespace

// FakeNamespaceController is a fake namespace.Controller used for testing
type FakeNamespaceController struct {
	monitoredNamespaces []string
	Controller
}

// NewFakeNamespaceController creates a fake namespace.Controler object for testing
func NewFakeNamespaceController(monitoredNamespaces []string) FakeNamespaceController {
	return FakeNamespaceController{
		monitoredNamespaces: monitoredNamespaces,
	}
}

// IsMonitoredNamespace returns if the namespace is monitored
func (f FakeNamespaceController) IsMonitoredNamespace(namespace string) bool {
	log.Debug().Msgf("Monitored namespaces = %v", f.monitoredNamespaces)
	for _, ns := range f.monitoredNamespaces {
		if ns == namespace {
			return true
		}
	}
	return false
}
