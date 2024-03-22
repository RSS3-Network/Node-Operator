package k8s

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

func MergeAnnotations(prev, current map[string]string) map[string]string {

	for ck, cv := range prev {
		if strings.Contains(ck, "kubernetes.io/") {
			if current == nil {
				current = make(map[string]string)
			}
			current[ck] = cv
		}
	}
	return current
}

func MergeFinalizers(src client.Object, finalizer string) []string {
	if !controllerutil.ContainsFinalizer(src, finalizer) {
		srcF := src.GetFinalizers()
		srcF = append(srcF, finalizer)
		src.SetFinalizers(srcF)
	}
	return src.GetFinalizers()
}
