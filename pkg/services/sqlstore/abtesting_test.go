package sqlstore

import (
	"fmt"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/models"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestExperiment(t *testing.T) {
	Convey("Testing Experiment command & queries", t, func() {
		InitTestDB(t)

		Convey("add two experiment", func() {
			experiment1 := &models.AddExperimentCommand{
				Name:   "experiment1",
				Value:  simplejson.New(),
				Author: "wangjia",
			}
			experiment2 := &models.AddExperimentCommand{
				Name:  "experiment2",
				Value: simplejson.New(),
			}

			So(bus.Dispatch(experiment1), ShouldBeNil)
			So(bus.Dispatch(experiment2), ShouldBeNil)

			query := &models.GetExperimentsQuery{
				Limit: 10,
				Start: 0,
			}

			So(bus.Dispatch(query), ShouldBeNil)
			So(len(query.Result), ShouldEqual, 2)
			for _, q := range query.Result {
				fmt.Println(q)
			}
		})
	})
}
