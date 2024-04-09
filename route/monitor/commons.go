package monitor

import (
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
	"github.com/spiker/spiker-server/route/shared"
)

func checkMeasurementAccess(c *shared.Context, measurementId int) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	service := shared.CreateService(S.MeasurementService{}, c).(*S.MeasurementService)

	return service.CheckAccessByDoctor(me, measurementId)
}

func checkComputedEventAccess(c *shared.Context, measurementId int, eventId int) error {
	if e := checkMeasurementAccess(c, measurementId); e != nil {
		return e
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	if e := service.CheckComputedEventInMeasurement(measurementId, eventId); e != nil {
		return e
	} else {
		return nil
	}
}

func checkAnnotatedEventAccess(c *shared.Context, measurementId int, eventId int) error {
	if e := checkMeasurementAccess(c, measurementId); e != nil {
		return e
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	if e := service.CheckAnnotatedEventInMeasurement(measurementId, eventId); e != nil {
		return e
	} else {
		return nil
	}
}