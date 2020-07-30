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

// ListMonitoredNamespaces returns the namespaces monitored by the mesh
func (f FakeNamespaceController) ListMonitoredNamespaces() ([]string, error) {
	return f.monitoredNamespaces, nil
}

// GetAnnouncementsChannel returns the channel on which namespace makes announcements
func (f FakeNamespaceController) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}
