package sqlstore

import (
	"github.com/grafana/grafana/pkg/bus"
	m "github.com/grafana/grafana/pkg/models"
	"time"
)

func init() {
	bus.AddHandler("sql", AddExperiment)
	bus.AddHandler("sql", UpDateExperiment)
	bus.AddHandler("sql", GetExperiments)
	bus.AddHandler("sql", DeleteExperiment)
}

func AddExperiment(cmd *m.AddExperimentCommand) error {
	return inTransaction(func(sess *DBSession) error {
		experiment := m.Experiment{
			Name:      cmd.Name,
			Value:     cmd.Value,
			Status:    m.StatusInit,
			Created:   time.Now(),
			Updated:   time.Now(),
			CreatedBy: cmd.Author,
			UpdatedBy: cmd.Author,
		}
		_, err := sess.Insert(&experiment)
		cmd.Result = experiment
		return err
	})
}

func UpDateExperiment(cmd *m.UpdateExperimentCommand) error {
	return inTransaction(func(sess *DBSession) error {
		experiment := m.Experiment{
			Name:      cmd.Name,
			Status:    cmd.Status,
			Value:     cmd.Value,
			Updated:   time.Now(),
			UpdatedBy: cmd.Author,
		}
		cmd.Result = experiment

		affectedRows, err := sess.ID(cmd.Id).Where("name = ?", cmd.Name).Update(&experiment)

		if err != nil {
			return err
		}

		if affectedRows == 0 {
			return m.ErrExperimentNotFound
		}

		return nil
	})
}

func GetExperiments(query *m.GetExperimentsQuery) error {
	query.Result = make([]*m.Experiment, 0)

	err := x.Table("experiment").
		Select(`experiment.id,
				experiment.id,
				experiment.name,
				experiment.status,
				experiment.created,
				experiment.updated,
				experiment.value,
				experiment.created_by,
				experiment.updated_by`).
		//Where("dashboard_version.dashboard_id=? AND dashboard.org_id=?", query.DashboardId, query.OrgId).
		OrderBy("experiment.updated DESC").
		Limit(query.Limit, query.Start).
		Find(&query.Result)
	if err != nil {
		return err
	}

	return nil
}

func DeleteExperiment(cmd *m.DeleteExperimentCommand) error {
	return inTransaction(func(sess *DBSession) error {
		sql := "DELETE FROM experiment WHERE id=?"
		result, err := sess.Exec(sql, cmd.Id)
		if err != nil {
			return err
		}
		affectRows, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if affectRows != 1 {
			return m.ErrExperimentDelete
		}

		return nil
	})
}
