package annotation

import (
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
	"github.com/spiker/spiker-server/route/shared"
)

func checkHospitalAccess(c *shared.Context) (*model.Hospital, error) {
	hospitalUuid := c.Param("hospital_uuid")

	service := shared.CreateService(S.HospitalService{}, c).(*S.HospitalService)

	return service.InquireByUuid(hospitalUuid)
}

func checkMeasurementAccess(c *shared.Context, measurementId int, h *model.Hospital) error {
	hospital, err := checkHospitalAccess(c)

	if err != nil {
		return err
	}

	if h != nil {
		*h = *hospital
	}

	service := shared.CreateService(S.MeasurementService{}, c).(*S.MeasurementService)

	return service.CheckAccessByHospital(hospital.Id, measurementId)
}

func checkComputedEventAccess(c *shared.Context, measurementId int, eventId int) error {
	if e := checkMeasurementAccess(c, measurementId, nil); e != nil {
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
	if e := checkMeasurementAccess(c, measurementId, nil); e != nil {
		return e
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	if e := service.CheckAnnotatedEventInMeasurement(measurementId, eventId); e != nil {
		return e
	} else {
		return nil
	}
}