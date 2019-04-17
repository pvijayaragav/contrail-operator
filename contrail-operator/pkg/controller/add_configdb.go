package controller

import (
	"github.com/operators/contrail-operator/pkg/controller/configdb"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, configdb.Add)
}
