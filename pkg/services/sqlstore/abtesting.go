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
	b, err := cmd.Value.ToDB()
	if err != nil {
		return err
	}
	return inTransaction(func(sess *DBSession) error {
		experiment := m.Experiment{
			Name:    cmd.Name,
			Value:   string(b),
			Status:  m.StatusInit,
			Created: time.Now(),
			Updated: time.Now(),
		}
		_, err := sess.Insert(&experiment)
		cmd.Result = experiment
		return err
	})
}

func UpDateExperiment(cmd *m.UpdateExperimentCommand) error {
	b, err := cmd.Value.ToDB()
	if err != nil {
		return err
	}
	return inTransaction(func(sess *DBSession) error {
		experiment := m.Experiment{
			Name:    cmd.Name,
			Value:   string(b),
			Updated: time.Now(),
		}

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
				experiment.value`).
		//Where("dashboard_version.dashboard_id=? AND dashboard.org_id=?", query.DashboardId, query.OrgId).
		OrderBy("experiment.updated DESC").
		Limit(query.Limit, query.Start).
		Find(&query.Result)
	if err != nil {
		return err
	}

	if len(query.Result) < 1 {
		return m.ErrExperimentNotFound
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
