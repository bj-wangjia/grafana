package api

import (
	"fmt"
	m "github.com/grafana/grafana/pkg/models"
)

func (hs *HTTPServer) GetExperiments(c *m.ReqContext) Response {
	limit := c.ParamsInt("limit")
	start := c.ParamsInt("start")
	if limit == 0 {
		limit = 100
	}
	query := m.GetExperimentsQuery{
		Limit: limit,
		Start: start,
	}
	if err := hs.Bus.Dispatch(&query); err != nil {
		return Error(500, "Error while get experiments", err)
	}

	return JSON(200, query.Result)
}

func (hs *HTTPServer) AddExperiment(c *m.ReqContext, cmd m.AddExperimentCommand) Response {
	hs.log.Info(fmt.Sprintf("%+v", cmd))
	cmd.Author = c.Login

	if err := hs.Bus.Dispatch(&cmd); err != nil {
		return Error(500, "Failed to add experiment", err)
	}

	return JSON(200, cmd.Result)
}

func (hs *HTTPServer) UpdateExperiment(c *m.ReqContext, cmd m.UpdateExperimentCommand) Response {
	cmd.Id = c.QueryInt("id")
	cmd.Author = c.Login
	if err := hs.Bus.Dispatch(&cmd); err != nil {
		return Error(500, "Failed to update experiment", err)
	}
	return Success("experiment update")
}

func (hs *HTTPServer) DeleteExperiment(c *m.ReqContext) Response {
	cmd := m.DeleteExperimentCommand{}
	cmd.Id = c.QueryInt64("id")
	if err := hs.Bus.Dispatch(&cmd); err != nil {
		return Error(500, fmt.Sprintf("Failed to delete experiment, id[%d]", cmd.Id), err)
	}
	return Success("experiment deleted")
}
