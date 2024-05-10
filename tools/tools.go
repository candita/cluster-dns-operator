//go:build tools
// +build tools

// This package contains import references to packages required only for the
// build process.
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
package tools

import (
	_ "github.com/openshift/api/operator/v1/zz_generated.crd-manifests"
	_ "github.com/openshift/build-machinery-go"
)
