package controller

const nodeFinalizer = "node.rss3.io/finalizer"

const (
	// typeAvailable represents the status of the Deployment reconciliation
	typeAvailable = "Available"
	// typeDegraded represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	typeDegraded = "Degraded"
)
