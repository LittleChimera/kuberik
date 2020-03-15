package controller

import (
	"github.com/kuberik/kuberik/pkg/controller/movie"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, movie.Add)
}
