package v1alpha1

import "time"

const (
	NodeFinalizer               = "node.rss3.io/finalizer"
	lastAppliedConfigAnnotation = "operator.node.rss3.io/last-applied-configuration"
	WaitReadyTimeout            = 180 * time.Second
)
