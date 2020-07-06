package models

import (
	"errors"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"time"
)

var (
	ErrExperimentNotFound = errors.New("experiment not found")
	ErrExperimentDelete   = errors.New("experiment not found")
)

const (
	StatusInit   = 0
	StatusActive = 1
	StatusPause  = 2
	StatusDelete = 3
)

type Experiment struct {
	Id    int64            `json:"id"`
	Name  string           `json:"name"`
	Value *simplejson.Json `json:"value"`

	Status    int64     `json:"status"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
}

type AddExperimentCommand struct {
	Name  string           `json:"name" binding:"required"`
	Value *simplejson.Json `json:"value" binding:"required"`

	Author string `json:"-"`

	Result Experiment `json:"-"`
}

type UpdateExperimentCommand struct {
	Id     int
	Name   string           `json:"name" binding:"required"`
	Status int64            `json:"status" binding:"required"`
	Value  *simplejson.Json `json:"value" binding:"required"`

	Author string `json:"-"`

	Result Experiment `json:"-"`
}

type GetExperimentsQuery struct {
	Limit int
	Start int

	Result []*Experiment
}

type DeleteExperimentCommand struct {
	Id int64
}
