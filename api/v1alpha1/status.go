package v1alpha1

type UpdateStatus string

const (
	// UpdateStatusAvailable represents the status of the Deployment reconciliation
	UpdateStatusAvailable UpdateStatus = "Available"
	// UpdateStatusDegraded represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	UpdateStatusDegraded UpdateStatus = "Degraded"
	// UpdateStatusFailed represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	UpdateStatusFailed UpdateStatus = "Failed"
	// UpdateStatusExpanding represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	UpdateStatusExpanding UpdateStatus = "Expanding"
	// UpdateStatusOperational represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	UpdateStatusOperational UpdateStatus = "Operational"
	// UpdateStatusPending represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	UpdateStatusPending UpdateStatus = "Pending"
)
