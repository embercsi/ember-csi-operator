package controller

import (
	"github.com/embercsi/ember-csi-operator/pkg/controller/embercsi"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, embercsi.Add)
}
